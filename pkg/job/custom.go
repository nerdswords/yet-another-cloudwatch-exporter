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
	job *config.CustomNamespace,
	region string,
	role config.Role,
	account *string,
	cloudwatchAPIConcurrency int,
) []*model.CloudwatchData {
	clientCloudwatch := apicloudwatch.NewWithMaxConcurrency(
		apicloudwatch.NewClient(
			logger,
			cache.GetCloudwatch(&region, role),
		),
		cloudwatchAPIConcurrency,
	)

	return scrapeCustomNamespaceJobUsingMetricData(
		ctx,
		job,
		region,
		account,
		clientCloudwatch,
		logger,
		metricsPerQuery,
	)
}

func scrapeCustomNamespaceJobUsingMetricData(
	ctx context.Context,
	job *config.CustomNamespace,
	region string,
	accountID *string,
	clientCloudwatch *apicloudwatch.MaxConcurrencyClient,
	logger logging.Logger,
	metricsPerQuery int,
) []*model.CloudwatchData {
	cw := []*model.CloudwatchData{}

	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	getMetricDatas := getMetricDataForQueriesForCustomNamespace(ctx, job, region, accountID, clientCloudwatch, logger)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Debug("No metrics data found")
		return cw
	}

	maxMetricCount := metricsPerQuery
	length := getMetricDataInputLength(job.Metrics)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))

	wg.Add(partition)

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			defer wg.Done()

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
	clientCloudwatch *apicloudwatch.MaxConcurrencyClient,
	logger logging.Logger,
) []model.CloudwatchData {
	mux := &sync.Mutex{}
	var getMetricDatas []model.CloudwatchData

	var wg sync.WaitGroup
	wg.Add(len(customNamespaceJob.Metrics))

	for _, metric := range customNamespaceJob.Metrics {
		// For every metric of the job get the full list of metrics.
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data.

		go func(metric *config.Metric) {
			defer wg.Done()
			metricsList, err := clientCloudwatch.ListMetrics(ctx, customNamespaceJob.Namespace, metric)
			if err != nil {
				logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", customNamespaceJob.Namespace)
				return
			}

			var data []model.CloudwatchData

			for _, cwMetric := range metricsList.Metrics {
				if len(customNamespaceJob.DimensionNameRequirements) > 0 && !metricDimensionsMatchNames(cwMetric, customNamespaceJob.DimensionNameRequirements) {
					continue
				}

				for _, stats := range metric.Statistics {
					id := fmt.Sprintf("id_%d", rand.Int())
					data = append(data, model.CloudwatchData{
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

			mux.Lock()
			getMetricDatas = append(getMetricDatas, data...)
			mux.Unlock()
		}(metric)
	}

	wg.Wait()
	return getMetricDatas
}
