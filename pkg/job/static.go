package job

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apicloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/session"
)

func runStaticJob(
	ctx context.Context,
	logger logging.Logger,
	cache session.SessionCache,
	region string,
	role config.Role,
	job *config.Static,
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

	return scrapeStaticJob(ctx, job, region, account, clientCloudwatch, logger)
}

func scrapeStaticJob(ctx context.Context, resource *config.Static, region string, accountID *string, clientCloudwatch *apicloudwatch.MaxConcurrencyClient, logger logging.Logger) []*model.CloudwatchData {
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
				AccountID:              accountID,
			}

			filter := apicloudwatch.CreateGetMetricStatisticsInput(
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
