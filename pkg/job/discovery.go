package job

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apicloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apitagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/session"
)

func runDiscoveryJob(
	ctx context.Context,
	logger logging.Logger,
	cache session.SessionCache,
	metricsPerQuery int,
	tagSemaphore chan struct{},
	job *config.Job,
	region string,
	role config.Role,
	account *string,
	exportedTags model.ExportedTagsOnMetrics,
) ([]*model.TaggedResource, []*model.CloudwatchData) {
	clientCloudwatch := apicloudwatch.NewClient(
		logger,
		cache.GetCloudwatch(&region, role),
	)

	clientTag := apitagging.NewClient(
		logger,
		cache.GetTagging(&region, role),
		cache.GetASG(&region, role),
		cache.GetAPIGateway(&region, role),
		cache.GetEC2(&region, role),
		cache.GetDMS(&region, role),
		cache.GetPrometheus(&region, role),
		cache.GetStorageGateway(&region, role),
	)

	return scrapeDiscoveryJobUsingMetricData(ctx, job, region, account, exportedTags, clientTag, clientCloudwatch, metricsPerQuery, job.RoundingPeriod, tagSemaphore, logger)
}

func scrapeDiscoveryJobUsingMetricData(
	ctx context.Context,
	job *config.Job,
	region string,
	accountID *string,
	tagsOnMetrics model.ExportedTagsOnMetrics,
	clientTag apitagging.Client,
	clientCloudwatch *apicloudwatch.Client,
	metricsPerQuery int,
	roundingPeriod *int64,
	tagSemaphore chan struct{},
	logger logging.Logger,
) (resources []*model.TaggedResource, cw []*model.CloudwatchData) {
	// Add the info tags of all the resources
	logger.Debug("Get tagged resources")
	tagSemaphore <- struct{}{}
	resources, err := clientTag.GetResources(ctx, job, region)
	<-tagSemaphore
	if err != nil {
		logger.Error(err, "Couldn't describe resources")
		return
	}

	if len(resources) == 0 {
		logger.Info("No tagged resources made it through filtering")
		return
	}

	svc := config.SupportedServices.GetService(job.Type)
	getMetricDatas := getMetricDataForQueries(ctx, job, svc, region, accountID, tagsOnMetrics, clientCloudwatch, resources, tagSemaphore, logger)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Info("No metrics data found")
		return
	}

	maxMetricCount := metricsPerQuery
	length := getMetricDataInputLength(job.Metrics)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))

	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(partition)

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			defer wg.Done()
			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			input := getMetricDatas[i:end]
			filter := apicloudwatch.CreateGetMetricDataInput(input, &svc.Namespace, length, job.Delay, roundingPeriod, logger)
			data := clientCloudwatch.GetMetricData(ctx, filter)
			if data != nil {
				output := make([]*model.CloudwatchData, 0)
				for _, MetricDataResult := range data.MetricDataResults {
					getMetricData, err := findGetMetricDataByID(input, *MetricDataResult.Id)
					if err == nil {
						if len(MetricDataResult.Values) != 0 {
							getMetricData.GetMetricDataPoint = MetricDataResult.Values[0]
							getMetricData.GetMetricDataTimestamps = MetricDataResult.Timestamps[0]
						}
						output = append(output, &getMetricData)
					}
				}
				mux.Lock()
				cw = append(cw, output...)
				mux.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return resources, cw
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

func findGetMetricDataByID(getMetricDatas []model.CloudwatchData, value string) (model.CloudwatchData, error) {
	var g model.CloudwatchData
	for _, getMetricData := range getMetricDatas {
		if *getMetricData.MetricID == value {
			return getMetricData, nil
		}
	}
	return g, fmt.Errorf("metric with id %s not found", value)
}

func getMetricDataForQueries(
	ctx context.Context,
	discoveryJob *config.Job,
	svc *config.ServiceConfig,
	region string,
	accountID *string,
	tagsOnMetrics model.ExportedTagsOnMetrics,
	clientCloudwatch *apicloudwatch.Client,
	resources []*model.TaggedResource,
	tagSemaphore chan struct{},
	logger logging.Logger,
) []model.CloudwatchData {
	var getMetricDatas []model.CloudwatchData

	// For every metric of the job
	for _, metric := range discoveryJob.Metrics {
		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}

		metricsList, err := clientCloudwatch.ListMetrics(ctx, svc.Namespace, metric)
		<-tagSemaphore

		if err != nil {
			logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", svc.Namespace)
			continue
		}

		if len(resources) == 0 {
			logger.Debug("No resources for metric", "metric_name", metric.Name, "namespace", svc.Namespace)
		}
		getMetricDatas = append(getMetricDatas, getFilteredMetricDatas(ctx, logger, region, accountID, discoveryJob.Type, discoveryJob.CustomTags, tagsOnMetrics, svc.DimensionRegexps, resources, metricsList.Metrics, discoveryJob.DimensionNameRequirements, metric)...)
	}
	return getMetricDatas
}

type resourceAssociator interface {
	associateMetricsToResources(cwMetric *cloudwatch.Metric) (*model.TaggedResource, bool)
}

func getFilteredMetricDatas(ctx context.Context, logger logging.Logger, region string, accountID *string, namespace string, customTags []model.Tag, tagsOnMetrics model.ExportedTagsOnMetrics, dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource, metricsList []*cloudwatch.Metric, dimensionNameList []string, m *config.Metric) (getMetricsData []model.CloudwatchData) {
	var ra resourceAssociator

	if config.FlagsFromCtx(ctx).IsFeatureEnabled(config.EncodingResourceAssociator) {
		// if the feature is enabled, use the new encodingAssociator instead of the old one
		ra = newEncodingMetricsToResourceAssociator(dimensionRegexps, resources)
	} else {
		// default to the previous resource associator implementation
		ra = newMetricsToResourceAssociator(dimensionRegexps, resources)
	}

	logger.Debug("FilterMetricData DimensionsFilter", "dimensionsFilter", ra)

	for _, cwMetric := range metricsList {
		if len(dimensionNameList) > 0 && !metricDimensionsMatchNames(cwMetric, dimensionNameList) {
			continue
		}

		resource := &model.TaggedResource{
			ARN:       "global",
			Namespace: namespace,
		}

		// TODO: refactor this logic after failing scenarios are fixed
		matchedResource, skip := ra.associateMetricsToResources(cwMetric)
		if matchedResource != nil {
			resource = matchedResource
		}

		if !skip {
			for _, stats := range m.Statistics {
				id := fmt.Sprintf("id_%d", rand.Int())
				metricTags := resource.MetricTags(tagsOnMetrics)
				getMetricsData = append(getMetricsData, model.CloudwatchData{
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
					AccountID:              accountID,
					Period:                 m.Period,
				})
			}
		}
	}
	return getMetricsData
}

func metricDimensionsMatchNames(metric *cloudwatch.Metric, dimensionNameRequirements []string) bool {
	if len(dimensionNameRequirements) != len(metric.Dimensions) {
		return false
	}
	for _, dimension := range metric.Dimensions {
		foundMatch := false
		for _, dimensionName := range dimensionNameRequirements {
			if *dimension.Name == dimensionName {
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
