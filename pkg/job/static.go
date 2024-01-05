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

			id := resource.Name
			data := model.CloudwatchData{
				ID:                     &id,
				Metric:                 &metric.Name,
				Namespace:              &resource.Namespace,
				Statistics:             metric.Statistics,
				NilToZero:              metric.NilToZero,
				AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
				Dimensions:             createStaticDimensions(resource.Dimensions),
			}

			data.Points = clientCloudwatch.GetMetricStatistics(ctx, logger, data.Dimensions, resource.Namespace, metric)

			if data.Points != nil {
				mux.Lock()
				cw = append(cw, &data)
				mux.Unlock()
			}
		}()
	}
	wg.Wait()
	return cw
}

func createStaticDimensions(dimensions []model.Dimension) []*model.Dimension {
	out := make([]*model.Dimension, 0, len(dimensions))
	for _, d := range dimensions {
		d := d
		out = append(out, &model.Dimension{
			Name:  d.Name,
			Value: d.Value,
		})
	}

	return out
}
