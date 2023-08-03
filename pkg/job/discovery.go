package job

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/associator"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/maxdimassociator"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type resourceAssociator interface {
	AssociateMetricToResource(cwMetric *model.Metric) (*model.TaggedResource, bool)
}

func runDiscoveryJob(
	ctx context.Context,
	logger logging.Logger,
	job *config.Job,
	region string,
	accountID string,
	tagsOnMetrics model.ExportedTagsOnMetrics,
	clientTag tagging.Client,
	clientCloudwatch cloudwatch.Client,
	metricsPerQuery int,
	concurrencyLimit int,
) ([]*model.TaggedResource, []*model.CloudwatchData) {
	logger.Debug("Get tagged resources")

	cw := []*model.CloudwatchData{}

	resources, err := clientTag.GetResources(ctx, job, region)
	if err != nil {
		if errors.Is(err, tagging.ErrExpectedToFindResources) {
			logger.Error(err, "No tagged resources made it through filtering")
		} else {
			logger.Error(err, "Couldn't describe resources")
		}
		return resources, cw
	}

	if len(resources) == 0 {
		logger.Debug("No tagged resources", "region", region, "namespace", job.Type)
	}

	svc := config.SupportedServices.GetService(job.Type)
	getMetricDatas := getMetricDataForQueries(ctx, logger, job, svc, region, accountID, tagsOnMetrics, clientCloudwatch, resources)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Info("No metrics data found")
		return resources, cw
	}

	maxMetricCount := metricsPerQuery
	length := getMetricDataInputLength(job.Metrics)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))
	logger.Debug("GetMetricData partitions", "total", partition)

	var wg sync.WaitGroup
	wg.Add(partition)

	mu := sync.Mutex{}
	getMetricDataOutput := make([][]cloudwatch.MetricDataResult, 0, partitionSize)

	count := 0

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i, n int) {
			defer wg.Done()
			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			input := getMetricDatas[i:end]
			data := clientCloudwatch.GetMetricData(ctx, logger, input, svc.Namespace, length, job.Delay, job.RoundingPeriod)
			if data != nil {
				getMetricDataOutput[n] = data
			} else {
				logger.Warn("GetMetricData partition empty result", "partition", n, "start", i, "end", end)
			}
		}(i, count)
		count++
	}
	wg.Wait()

	// Update getMetricDatas slice with values and timestamps from API response.
	// We iterate through the response MetricDataResults and match the result ID
	// with what was sent in the API request.
	// In the event that the API response contains any ID we don't know about
	// (shouldn't really happen) we log a warning and move on. On the other hand,
	// in case the API response does not contain results for all the IDs we've
	// requested, unprocessed elements will be removed later on.
	for _, data := range getMetricDataOutput {
		if data == nil {
			continue
		}
		for _, metricDataResult := range data {
			idx := findGetMetricDataByID(getMetricDatas, metricDataResult.ID)
			if idx == -1 {
				logger.Warn("GetMetricData returned unknown metric ID", "metric_id", metricDataResult.ID)
				continue
			}
			// Copy to avoid a loop closure bug
			dataPoint := metricDataResult.Datapoint
			getMetricDatas[idx].GetMetricDataPoint = &dataPoint
			getMetricDatas[idx].GetMetricDataTimestamps = metricDataResult.Timestamp
			getMetricDatas[idx].MetricID = nil // mark as processed
		}
	}

	// Remove unprocessed/unknown elements in place, if any. Since getMetricDatas
	// is a slice of pointers, the compaction can be easily done in-place.
	getMetricDatas = compact(getMetricDatas, func(m *model.CloudwatchData) bool {
		return m.MetricID == nil
	})
	return resources, getMetricDatas
}

func xxx(output [][]*cloudwatch.MetricDataResult, datas []*model.CloudwatchData, logger logging.Logger) {
	// Update getMetricDatas slice with values and timestamps from API response.
	// We iterate through the response MetricDataResults and match the result ID
	// with what was sent in the API request.
	// In the event that the API response contains any ID we don't know about
	// (shouldn't really happen) we log a warning and move on. On the other hand,
	// in case the API response does not contain results for all the IDs we've
	// requested, unprocessed elements will be removed later on.
	for _, data := range output {
		if data == nil {
			continue
		}
		for _, metricDataResult := range data {
			idx := findGetMetricDataByID(datas, *metricDataResult.ID)
			if idx == -1 {
				logger.Warn("GetMetricData returned unknown metric ID", "metric_id", *metricDataResult.ID)
				continue
			}
			datas[idx].GetMetricDataPoint = metricDataResult.Datapoint
			datas[idx].GetMetricDataTimestamps = metricDataResult.Timestamp
			datas[idx].MetricID = nil // mark as processed
		}
	}
}

func getMetricDataInputLength(metrics []*config.Metric) int64 {
	var length int64
	for _, metric := range metrics {
		if metric.Length > length {
			length = metric.Length
		}
	}
	return length
}

func findGetMetricDataByID(getMetricDatas []*model.CloudwatchData, value string) int {
	for i := 0; i < len(getMetricDatas); i++ {
		if getMetricDatas[i].MetricID == nil {
			continue // skip elements that have been already marked
		}
		if *(getMetricDatas[i].MetricID) == value {
			return i
		}
	}
	return -1
}

func getMetricDataForQueries(
	ctx context.Context,
	logger logging.Logger,
	discoveryJob *config.Job,
	svc *config.ServiceConfig,
	region string,
	accountID string,
	tagsOnMetrics model.ExportedTagsOnMetrics,
	clientCloudwatch cloudwatch.Client,
	resources []*model.TaggedResource,
) []*model.CloudwatchData {
	mux := &sync.Mutex{}
	var getMetricDatas []*model.CloudwatchData

	var wg sync.WaitGroup
	wg.Add(len(discoveryJob.Metrics))

	// For every metric of the job call the ListMetrics API
	// to fetch the existing combinations of dimensions and
	// value of dimensions with data.

	if config.FlagsFromCtx(ctx).IsFeatureEnabled(config.ListMetricsCallback) {
		for _, metric := range discoveryJob.Metrics {
			go func(metric *config.Metric) {
				defer wg.Done()

				var assoc resourceAssociator
				if config.FlagsFromCtx(ctx).IsFeatureEnabled(config.MaxDimensionsAssociator) {
					assoc = maxdimassociator.NewAssociator(logger, svc.DimensionRegexps, resources)
				} else {
					assoc = associator.NewAssociator(svc.DimensionRegexps, resources)
				}

				_, err := clientCloudwatch.ListMetrics(ctx, svc.Namespace, metric, discoveryJob.RecentlyActiveOnly, func(page []*model.Metric) {
					data := getFilteredMetricDatas(logger, region, accountID, discoveryJob.Type, discoveryJob.CustomTags, tagsOnMetrics, page, discoveryJob.DimensionNameRequirements, metric, assoc)

					mux.Lock()
					getMetricDatas = append(getMetricDatas, data...)
					mux.Unlock()
				})
				if err != nil {
					logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", svc.Namespace)
					return
				}
			}(metric)
		}
	} else {
		for _, metric := range discoveryJob.Metrics {
			go func(metric *config.Metric) {
				defer wg.Done()

				var assoc resourceAssociator
				if config.FlagsFromCtx(ctx).IsFeatureEnabled(config.MaxDimensionsAssociator) {
					assoc = maxdimassociator.NewAssociator(logger, svc.DimensionRegexps, resources)
				} else {
					assoc = associator.NewAssociator(svc.DimensionRegexps, resources)
				}

				metricsList, err := clientCloudwatch.ListMetrics(ctx, svc.Namespace, metric, discoveryJob.RecentlyActiveOnly, nil)
				if err != nil {
					logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", svc.Namespace)
					return
				}

				data := getFilteredMetricDatas(logger, region, accountID, discoveryJob.Type, discoveryJob.CustomTags, tagsOnMetrics, metricsList, discoveryJob.DimensionNameRequirements, metric, assoc)

				mux.Lock()
				getMetricDatas = append(getMetricDatas, data...)
				mux.Unlock()
			}(metric)
		}
	}

	wg.Wait()
	return getMetricDatas
}

func getFilteredMetricDatas(
	logger logging.Logger,
	region string,
	accountID string,
	namespace string,
	customTags []model.Tag,
	tagsOnMetrics model.ExportedTagsOnMetrics,
	metricsList []*model.Metric,
	dimensionNameList []string,
	m *config.Metric,
	assoc resourceAssociator,
) []*model.CloudwatchData {
	getMetricsData := make([]*model.CloudwatchData, 0, len(metricsList))
	for _, cwMetric := range metricsList {
		if len(dimensionNameList) > 0 && !metricDimensionsMatchNames(cwMetric, dimensionNameList) {
			continue
		}

		matchedResource, skip := assoc.AssociateMetricToResource(cwMetric)
		if skip {
			if logger.IsDebugEnabled() {
				dimensions := make([]string, 0, len(cwMetric.Dimensions))
				for _, dim := range cwMetric.Dimensions {
					dimensions = append(dimensions, fmt.Sprintf("%s=%s", dim.Name, dim.Value))
				}
				logger.Debug("skipping metric unmatched by associator", "metric", m.Name, "dimensions", strings.Join(dimensions, ","))
			}
			continue
		}

		resource := matchedResource
		if resource == nil {
			resource = &model.TaggedResource{
				ARN:       "global",
				Namespace: namespace,
			}
		}
		metricTags := resource.MetricTags(tagsOnMetrics)

		for _, stats := range m.Statistics {
			id := fmt.Sprintf("id_%d", rand.Int())

			getMetricsData = append(getMetricsData, &model.CloudwatchData{
				ID:                     &resource.ARN,
				MetricID:               &id,
				Metric:                 &m.Name,
				Namespace:              &namespace,
				Statistics:             []string{stats},
				NilToZero:              m.NilToZero,
				AddCloudwatchTimestamp: m.AddCloudwatchTimestamp,
				Tags:                   metricTags,
				CustomTags:             customTags,
				Dimensions:             cwMetric.Dimensions,
				Region:                 &region,
				AccountID:              &accountID,
				Period:                 m.Period,
			})
		}
	}
	return getMetricsData
}

func metricDimensionsMatchNames(metric *model.Metric, dimensionNameRequirements []string) bool {
	if len(dimensionNameRequirements) != len(metric.Dimensions) {
		return false
	}
	for _, dimension := range metric.Dimensions {
		foundMatch := false
		for _, dimensionName := range dimensionNameRequirements {
			if dimension.Name == dimensionName {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return false
		}
	}
	return true
}
