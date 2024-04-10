package job

import (
	"context"
	"sync"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func runCustomNamespaceJob(
	ctx context.Context,
	logger logging.Logger,
	job model.CustomNamespaceJob,
	clientCloudwatch cloudwatch.Client,
	gmdProcessor getMetricDataProcessor,
) []*model.CloudwatchData {
	cloudwatchDatas := getMetricDataForQueriesForCustomNamespace(ctx, job, clientCloudwatch, logger)
	if len(cloudwatchDatas) == 0 {
		logger.Debug("No metrics data found")
		return nil
	}

	jobLength := getLargestLengthForMetrics(job.Metrics)
	var err error
	cloudwatchDatas, err = gmdProcessor.Run(ctx, logger, job.Namespace, jobLength, job.Delay, job.RoundingPeriod, cloudwatchDatas)
	if err != nil {
		logger.Error(err, "Failed to get metric data")
		return nil
	}

	return cloudwatchDatas
}

func getMetricDataForQueriesForCustomNamespace(
	ctx context.Context,
	customNamespaceJob model.CustomNamespaceJob,
	clientCloudwatch cloudwatch.Client,
	logger logging.Logger,
) []*model.CloudwatchData {
	mux := &sync.Mutex{}
	var getMetricDatas []*model.CloudwatchData

	var wg sync.WaitGroup
	wg.Add(len(customNamespaceJob.Metrics))

	for _, metric := range customNamespaceJob.Metrics {
		// For every metric of the job get the full list of metrics.
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data.

		go func(metric *model.MetricConfig) {
			defer wg.Done()
			err := clientCloudwatch.ListMetrics(ctx, customNamespaceJob.Namespace, metric, customNamespaceJob.RecentlyActiveOnly, func(page []*model.Metric) {
				var data []*model.CloudwatchData

				for _, cwMetric := range page {
					if len(customNamespaceJob.DimensionNameRequirements) > 0 && !metricDimensionsMatchNames(cwMetric, customNamespaceJob.DimensionNameRequirements) {
						continue
					}

					for _, stat := range metric.Statistics {
						data = append(data, &model.CloudwatchData{
							MetricName:   metric.Name,
							ResourceName: customNamespaceJob.Name,
							Namespace:    customNamespaceJob.Namespace,
							Dimensions:   cwMetric.Dimensions,
							GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{
								Period:    metric.Period,
								Length:    metric.Length,
								Delay:     metric.Delay,
								Statistic: stat,
							},
							MetricMigrationParams: model.MetricMigrationParams{
								NilToZero:              metric.NilToZero,
								AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
							},
							Tags:                      nil,
							GetMetricDataResult:       nil,
							GetMetricStatisticsResult: nil,
						})
					}
				}

				mux.Lock()
				getMetricDatas = append(getMetricDatas, data...)
				mux.Unlock()
			})
			if err != nil {
				logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", customNamespaceJob.Namespace)
				return
			}
		}(metric)
	}

	wg.Wait()
	return getMetricDatas
}
