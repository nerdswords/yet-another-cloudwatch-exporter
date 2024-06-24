package job

import (
	"context"
	"sync"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/getmetricdata"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func ScrapeAwsData(
	ctx context.Context,
	logger logging.Logger,
	jobsCfg model.JobsConfig,
	factory clients.Factory,
	metricsPerQuery int,
	cloudwatchConcurrency cloudwatch.ConcurrencyConfig,
	taggingAPIConcurrency int,
) ([]model.TaggedResourceResult, []model.CloudwatchMetricResult) {
	mux := &sync.Mutex{}
	cwData := make([]model.CloudwatchMetricResult, 0)
	awsInfoData := make([]model.TaggedResourceResult, 0)
	var wg sync.WaitGroup

	for _, discoveryJob := range jobsCfg.DiscoveryJobs {
		for _, role := range discoveryJob.Roles {
			for _, region := range discoveryJob.Regions {
				wg.Add(1)
				go func(discoveryJob model.DiscoveryJob, region string, role model.Role) {
					defer wg.Done()
					jobLogger := logger.With("job_type", discoveryJob.Type, "region", region, "arn", role.RoleArn)
					accountID, err := factory.GetAccountClient(region, role).GetAccount(ctx)
					if err != nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", accountID)

					cloudwatchClient := factory.GetCloudwatchClient(region, role, cloudwatchConcurrency)
					gmdProcessor := getmetricdata.NewDefaultProcessor(logger, cloudwatchClient, metricsPerQuery, cloudwatchConcurrency.GetMetricData)
					resources, metrics := runDiscoveryJob(ctx, jobLogger, discoveryJob, region, factory.GetTaggingClient(region, role, taggingAPIConcurrency), cloudwatchClient, gmdProcessor)
					addDataToOutput := len(metrics) != 0
					if config.FlagsFromCtx(ctx).IsFeatureEnabled(config.AlwaysReturnInfoMetrics) {
						addDataToOutput = addDataToOutput || len(resources) != 0
					}
					if addDataToOutput {
						sc := &model.ScrapeContext{
							Region:     region,
							AccountID:  accountID,
							CustomTags: discoveryJob.CustomTags,
						}
						metricResult := model.CloudwatchMetricResult{
							Context: sc,
							Data:    metrics,
						}
						resourceResult := model.TaggedResourceResult{
							Data: resources,
						}
						if discoveryJob.IncludeContextOnInfoMetrics {
							resourceResult.Context = sc
						}

						mux.Lock()
						awsInfoData = append(awsInfoData, resourceResult)
						cwData = append(cwData, metricResult)
						mux.Unlock()
					}
				}(discoveryJob, region, role)
			}
		}
	}

	for _, staticJob := range jobsCfg.StaticJobs {
		for _, role := range staticJob.Roles {
			for _, region := range staticJob.Regions {
				wg.Add(1)
				go func(staticJob model.StaticJob, region string, role model.Role) {
					defer wg.Done()
					jobLogger := logger.With("static_job_name", staticJob.Name, "region", region, "arn", role.RoleArn)
					accountID, err := factory.GetAccountClient(region, role).GetAccount(ctx)
					if err != nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", accountID)

					metrics := runStaticJob(ctx, jobLogger, staticJob, factory.GetCloudwatchClient(region, role, cloudwatchConcurrency))
					metricResult := model.CloudwatchMetricResult{
						Context: &model.ScrapeContext{
							Region:     region,
							AccountID:  accountID,
							CustomTags: staticJob.CustomTags,
						},
						Data: metrics,
					}
					mux.Lock()
					cwData = append(cwData, metricResult)
					mux.Unlock()
				}(staticJob, region, role)
			}
		}
	}

	for _, customNamespaceJob := range jobsCfg.CustomNamespaceJobs {
		for _, role := range customNamespaceJob.Roles {
			for _, region := range customNamespaceJob.Regions {
				wg.Add(1)
				go func(customNamespaceJob model.CustomNamespaceJob, region string, role model.Role) {
					defer wg.Done()
					jobLogger := logger.With("custom_metric_namespace", customNamespaceJob.Namespace, "region", region, "arn", role.RoleArn)
					accountID, err := factory.GetAccountClient(region, role).GetAccount(ctx)
					if err != nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", accountID)

					cloudwatchClient := factory.GetCloudwatchClient(region, role, cloudwatchConcurrency)
					gmdProcessor := getmetricdata.NewDefaultProcessor(logger, cloudwatchClient, metricsPerQuery, cloudwatchConcurrency.GetMetricData)
					metrics := runCustomNamespaceJob(ctx, jobLogger, customNamespaceJob, cloudwatchClient, gmdProcessor)
					metricResult := model.CloudwatchMetricResult{
						Context: &model.ScrapeContext{
							Region:     region,
							AccountID:  accountID,
							CustomTags: customNamespaceJob.CustomTags,
						},
						Data: metrics,
					}
					mux.Lock()
					cwData = append(cwData, metricResult)
					mux.Unlock()
				}(customNamespaceJob, region, role)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData
}
