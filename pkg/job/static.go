package job

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func runStaticJob(
	ctx context.Context,
	logger logging.Logger,
	resource *config.Static,
	region string,
	accountID string,
	clientCloudwatch cloudwatch_client.Client,
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
				CustomTags:             resource.CustomTags,
				Dimensions:             createStaticDimensions(resource.Dimensions),
				Region:                 &region,
				AccountID:              &accountID,
			}

			filter := cloudwatch_client.CreateGetMetricStatisticsInput(
				data.Dimensions,
				&resource.Namespace,
				metric,
				logger,
			)

			data.Points = clientCloudwatch.GetMetricStatistics(ctx, filter)

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

func createStaticDimensions(dimensions []config.Dimension) []*cloudwatch.Dimension {
	out := make([]*cloudwatch.Dimension, 0, len(dimensions))
	for _, d := range dimensions {
		d := d
		out = append(out, &cloudwatch.Dimension{
			Name:  &d.Name,
			Value: &d.Value,
		})
	}

	return out
}
