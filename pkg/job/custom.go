package job

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apicloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/session"
)

func runCustomNamespaceJob(
	ctx context.Context,
	logger logging.Logger,
	cache session.SessionCache,
	metricsPerQuery int,
	cloudwatchSemaphore chan struct{},
	tagSemaphore chan struct{},
	job *config.CustomNamespace,
	region string,
	role config.Role,
	account *string,
) []*model.CloudwatchData {
	clientCloudwatch := apicloudwatch.NewCloudWatchInterface(
		cache.GetCloudwatch(&region, role),
		logger,
	)

	return scrapeCustomNamespaceJobUsingMetricData(
		ctx,
		job,
		region,
		account,
		clientCloudwatch,
		cloudwatchSemaphore,
		tagSemaphore,
		logger,
		metricsPerQuery,
	)
}

func scrapeCustomNamespaceJobUsingMetricData(
	ctx context.Context,
	job *config.CustomNamespace,
	region string,
	accountID *string,
	clientCloudwatch *apicloudwatch.CloudwatchInterface,
	cloudwatchSemaphore chan struct{},
	tagSemaphore chan struct{},
	logger logging.Logger,
	metricsPerQuery int,
) (cw []*model.CloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	getMetricDatas := getMetricDataForQueriesForCustomNamespace(ctx, job, region, accountID, clientCloudwatch, tagSemaphore, logger)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Debug("No metrics data found")
		return
	}

	maxMetricCount := metricsPerQuery
	length := getMetricDataInputLength(job.Metrics)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))

	wg.Add(partition)

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			cloudwatchSemaphore <- struct{}{}

			defer func() {
				defer wg.Done()
				<-cloudwatchSemaphore
			}()

			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			input := getMetricDatas[i:end]
			filter := apicloudwatch.CreateGetMetricDataInput(input, &job.Namespace, length, job.Delay, job.RoundingPeriod, logger)
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
	return cw
}

func getMetricDataForQueriesForCustomNamespace(
	ctx context.Context,
	customNamespaceJob *config.CustomNamespace,
	region string,
	accountID *string,
	clientCloudwatch *apicloudwatch.CloudwatchInterface,
	tagSemaphore chan struct{},
	logger logging.Logger,
) []model.CloudwatchData {
	var getMetricDatas []model.CloudwatchData

	// For every metric of the job
	for _, metric := range customNamespaceJob.Metrics {
		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}

		metricsList, err := clientCloudwatch.ListMetrics(ctx, customNamespaceJob.Namespace, metric)
		<-tagSemaphore

		if err != nil {
			logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", customNamespaceJob.Namespace)
			continue
		}

		for _, cwMetric := range metricsList.Metrics {
			if len(customNamespaceJob.DimensionNameRequirements) > 0 && !metricDimensionsMatchNames(cwMetric, customNamespaceJob.DimensionNameRequirements) {
				continue
			}

			for _, stats := range metric.Statistics {
				id := fmt.Sprintf("id_%d", rand.Int())
				getMetricDatas = append(getMetricDatas, model.CloudwatchData{
					ID:                     &customNamespaceJob.Name,
					MetricID:               &id,
					Metric:                 &metric.Name,
					Namespace:              &customNamespaceJob.Namespace,
					Statistics:             []string{stats},
					NilToZero:              metric.NilToZero,
					AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
					CustomTags:             customNamespaceJob.CustomTags,
					Dimensions:             cwMetric.Dimensions,
					Region:                 &region,
					AccountID:              accountID,
					Period:                 metric.Period,
				})
			}
		}
	}
	return getMetricDatas
}
