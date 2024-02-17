package job

import (
	"context"
	"sync"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func runStaticJob(
	ctx context.Context,
	logger logging.Logger,
	resource model.StaticJob,
	clientCloudwatch cloudwatch.Client,
) []*model.CloudwatchData {
	cw := []*model.CloudwatchData{}
	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	for j := range resource.Metrics {
		metric := resource.Metrics[j]
		wg.Add(1)
		go func() {
			defer wg.Done()

			data := model.CloudwatchData{
				MetricName:   metric.Name,
				ResourceName: resource.Name,
				Namespace:    resource.Namespace,
				Dimensions:   createStaticDimensions(resource.Dimensions),
				MetricMigrationParams: model.MetricMigrationParams{
					NilToZero:              metric.NilToZero,
					AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
				},
				Tags:                          nil,
				GetMetricDataProcessingParams: nil,
				GetMetricDataResult:           nil,
				GetMetricStatisticsResult:     nil,
			}

			data.GetMetricStatisticsResult = &model.GetMetricStatisticsResult{
				Datapoints: clientCloudwatch.GetMetricStatistics(ctx, logger, data.Dimensions, resource.Namespace, metric),
				Statistics: metric.Statistics,
			}

			if data.GetMetricStatisticsResult.Datapoints != nil {
				mux.Lock()
				cw = append(cw, &data)
				mux.Unlock()
			}
		}()
	}
	wg.Wait()
	return cw
}

func createStaticDimensions(dimensions []model.Dimension) []model.Dimension {
	out := make([]model.Dimension, 0, len(dimensions))
	for _, d := range dimensions {
		d := d
		out = append(out, model.Dimension{
			Name:  d.Name,
			Value: d.Value,
		})
	}

	return out
}
