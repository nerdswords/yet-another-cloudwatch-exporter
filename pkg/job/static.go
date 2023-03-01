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
	cloudwatchSemaphore chan struct{},
) []*model.CloudwatchData {
	clientCloudwatch := apicloudwatch.NewClient(
		logger,
		cache.GetCloudwatch(&region, role),
	)

	return scrapeStaticJob(ctx, job, region, account, clientCloudwatch, cloudwatchSemaphore, logger)
}

func scrapeStaticJob(ctx context.Context, resource *config.Static, region string, accountID *string, clientCloudwatch *apicloudwatch.Client, cloudwatchSemaphore chan struct{}, logger logging.Logger) (cw []*model.CloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	for j := range resource.Metrics {
		metric := resource.Metrics[j]
		wg.Add(1)
		go func() {
			defer wg.Done()

			cloudwatchSemaphore <- struct{}{}
			defer func() {
				<-cloudwatchSemaphore
			}()

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

func createStaticDimensions(dimensions []config.Dimension) (output []*cloudwatch.Dimension) {
	for _, d := range dimensions {
		d := d
		output = append(output, &cloudwatch.Dimension{
			Name:  &d.Name,
			Value: &d.Value,
		})
	}

	return output
}
