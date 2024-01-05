package job

import (
	"context"
	"fmt"
	"math"
	"math/rand"
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
	metricsPerQuery int,
) []*model.CloudwatchData {
	cw := []*model.CloudwatchData{}

	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	getMetricDatas := getMetricDataForQueriesForCustomNamespace(ctx, job, clientCloudwatch, logger)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Debug("No metrics data found")
		return cw
	}

	maxMetricCount := metricsPerQuery
	length := getMetricDataInputLength(job.Metrics)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))
	logger.Debug("GetMetricData partitions", "total", partition)

	wg.Add(partition)

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			defer wg.Done()

			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			input := getMetricDatas[i:end]
			data := clientCloudwatch.GetMetricData(ctx, logger, input, job.Namespace, length, job.Delay, job.RoundingPeriod)

			if data != nil {
				output := make([]*model.CloudwatchData, 0)
				for _, result := range data {
					getMetricData, err := findGetMetricDataByIDForCustomNamespace(input, result.ID)
					if err == nil {
						// Copy to avoid a loop closure bug
						dataPoint := result.Datapoint
						getMetricData.GetMetricDataPoint = &dataPoint
						getMetricData.GetMetricDataTimestamps = result.Timestamp
						output = append(output, getMetricData)
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

func findGetMetricDataByIDForCustomNamespace(getMetricDatas []*model.CloudwatchData, value string) (*model.CloudwatchData, error) {
	for _, getMetricData := range getMetricDatas {
		if *getMetricData.MetricID == value {
			return getMetricData, nil
		}
	}
	return nil, fmt.Errorf("metric with id %s not found", value)
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

					for _, stats := range metric.Statistics {
						id := fmt.Sprintf("id_%d", rand.Int())
						data = append(data, &model.CloudwatchData{
							ID:                     &customNamespaceJob.Name,
							MetricID:               &id,
							Metric:                 &metric.Name,
							Namespace:              &customNamespaceJob.Namespace,
							Statistics:             []string{stats},
							NilToZero:              metric.NilToZero,
							AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
							Dimensions:             cwMetric.Dimensions,
							Period:                 metric.Period,
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
