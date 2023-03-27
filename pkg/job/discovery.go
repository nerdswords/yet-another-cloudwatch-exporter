package job

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apicloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apitagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
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
) ([]*model.TaggedResource, []*model.CloudwatchData, []*promutil.PrometheusMetric) {
	clientCloudwatch := apicloudwatch.NewCloudWatchInterface(
		cache.GetCloudwatch(&region, role),
		logger,
	)

	clientTag := apitagging.TagsInterface{
		Client:               cache.GetTagging(&region, role),
		APIGatewayClient:     cache.GetAPIGateway(&region, role),
		AsgClient:            cache.GetASG(&region, role),
		DmsClient:            cache.GetDMS(&region, role),
		Ec2Client:            cache.GetEC2(&region, role),
		DynamoDBClient:       cache.GetDynamoDB(&region, role),
		StoragegatewayClient: cache.GetStorageGateway(&region, role),
		PrometheusClient:     cache.GetPrometheus(&region, role),
		Logger:               logger,
	}

	resources, cwData := scrapeDiscoveryJobUsingMetricData(ctx, job, region, account, exportedTags, clientTag, clientCloudwatch, metricsPerQuery, job.RoundingPeriod, tagSemaphore, logger)

	additionalMetrics, _ := scrapeAdditionalMetrics(job, resources, clientTag)

	return resources, cwData, additionalMetrics
}

func scrapeDiscoveryJobUsingMetricData(
	ctx context.Context,
	job *config.Job,
	region string,
	accountID *string,
	tagsOnMetrics model.ExportedTagsOnMetrics,
	clientTag apitagging.TagsInterface,
	clientCloudwatch *apicloudwatch.CloudwatchInterface,
	metricsPerQuery int,
	roundingPeriod *int64,
	tagSemaphore chan struct{},
	logger logging.Logger,
) (resources []*model.TaggedResource, cw []*model.CloudwatchData) {
	// Add the info tags of all the resources
	logger.Debug("Get tagged resources")
	tagSemaphore <- struct{}{}
	resources, err := clientTag.Get(ctx, job, region)
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
	clientCloudwatch *apicloudwatch.CloudwatchInterface,
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
		getMetricDatas = append(getMetricDatas, getFilteredMetricDatas(logger, region, accountID, discoveryJob.Type, discoveryJob.CustomTags, tagsOnMetrics, svc.DimensionRegexps, resources, metricsList.Metrics, discoveryJob.DimensionNameRequirements, metric)...)
	}
	return getMetricDatas
}

func getFilteredMetricDatas(logger logging.Logger, region string, accountID *string, namespace string, customTags []model.Tag, tagsOnMetrics model.ExportedTagsOnMetrics, dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource, metricsList []*cloudwatch.Metric, dimensionNameList []string, m *config.Metric) (getMetricsData []model.CloudwatchData) {
	type filterValues map[string]*model.TaggedResource
	dimensionsFilter := make(map[string]filterValues)
	for _, dimensionRegexp := range dimensionRegexps {
		names := dimensionRegexp.SubexpNames()
		for i, dimensionName := range names {
			if i != 0 {
				names[i] = strings.ReplaceAll(dimensionName, "_", " ")
				if _, ok := dimensionsFilter[names[i]]; !ok {
					dimensionsFilter[names[i]] = make(filterValues)
				}
			}
		}
		for _, r := range resources {
			if dimensionRegexp.Match([]byte(r.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(r.ARN)
				for i, value := range dimensionMatch {
					if i != 0 {
						dimensionsFilter[names[i]][value] = r
					}
				}
			}
		}
	}

	logger.Debug("FilterMetricData DimensionsFilter", "dimensionsFilter", dimensionsFilter)

	for _, cwMetric := range metricsList {
		skip := false
		alreadyFound := false
		r := &model.TaggedResource{
			ARN:       "global",
			Namespace: namespace,
		}
		if len(dimensionNameList) > 0 && !metricDimensionsMatchNames(cwMetric, dimensionNameList) {
			continue
		}

		for _, dimension := range cwMetric.Dimensions {
			if dimensionFilterValues, ok := dimensionsFilter[*dimension.Name]; ok {
				if d, ok := dimensionFilterValues[*dimension.Value]; !ok {
					if !alreadyFound {
						skip = true
					}
					break
				} else {
					alreadyFound = true
					r = d
				}
			}
		}

		if !skip {
			for _, stats := range m.Statistics {
				id := fmt.Sprintf("id_%d", rand.Int())
				metricTags := r.MetricTags(tagsOnMetrics)
				getMetricsData = append(getMetricsData, model.CloudwatchData{
					ID:                     &r.ARN,
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
