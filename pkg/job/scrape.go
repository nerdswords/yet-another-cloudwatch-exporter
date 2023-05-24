package job

import (
	"context"
	"sync"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func ScrapeAwsData(
	ctx context.Context,
	logger logging.Logger,
	cfg config.ScrapeConf,
	cache clients.Cache,
	metricsPerQuery int,
	cloudWatchAPIConcurrency int,
	taggingAPIConcurrency int,
) ([]*model.TaggedResource, []*model.CloudwatchData) {
	mux := &sync.Mutex{}
	cwData := make([]*model.CloudwatchData, 0)
	awsInfoData := make([]*model.TaggedResource, 0)
	var wg sync.WaitGroup

	// since we have called refresh, we have loaded all the credentials
	// into the clients and it is now safe to call concurrently. Defer the
	// clearing, so we always clear credentials before the next scrape
	cache.Refresh()
	defer cache.Clear()

	for _, discoveryJob := range cfg.Discovery.Jobs {
		for _, role := range discoveryJob.Roles {
			for _, region := range discoveryJob.Regions {
				wg.Add(1)
				go func(discoveryJob *config.Job, region string, role config.Role) {
					defer wg.Done()
					jobLogger := logger.With("job_type", discoveryJob.Type, "region", region, "arn", role.RoleArn)
					accountID, err := cache.GetAccountClient(region, role).GetAccount(ctx)
					if err != nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", accountID)

					resources, metrics := runDiscoveryJob(ctx, jobLogger, discoveryJob, region, accountID, cfg.Discovery.ExportedTagsOnMetrics, cache.GetTaggingClient(region, role, taggingAPIConcurrency), cache.GetCloudwatchClient(region, role, cloudWatchAPIConcurrency), metricsPerQuery)
					if len(metrics) != 0 {
						mux.Lock()
						awsInfoData = append(awsInfoData, resources...)
						cwData = append(cwData, metrics...)
						mux.Unlock()
					}
				}(discoveryJob, region, role)
			}
		}
	}

	for _, staticJob := range cfg.Static {
		for _, role := range staticJob.Roles {
			for _, region := range staticJob.Regions {
				wg.Add(1)
				go func(staticJob *config.Static, region string, role config.Role) {
					defer wg.Done()
					jobLogger := logger.With("static_job_name", staticJob.Name, "region", region, "arn", role.RoleArn)
					accountID, err := cache.GetAccountClient(region, role).GetAccount(ctx)
					if err != nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", accountID)

					metrics := runStaticJob(ctx, jobLogger, staticJob, region, accountID, cache.GetCloudwatchClient(region, role, cloudWatchAPIConcurrency))

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(staticJob, region, role)
			}
		}
	}

	for _, customNamespaceJob := range cfg.CustomNamespace {
		logger.Warn("Jobs of type 'customNamespace' are deprecated and will be removed soon", "job", customNamespaceJob.Name)

		for _, role := range customNamespaceJob.Roles {
			for _, region := range customNamespaceJob.Regions {
				wg.Add(1)
				go func(customNamespaceJob *config.CustomNamespace, region string, role config.Role) {
					defer wg.Done()
					jobLogger := logger.With("custom_metric_namespace", customNamespaceJob.Namespace, "region", region, "arn", role.RoleArn)
					accountID, err := cache.GetAccountClient(region, role).GetAccount(ctx)
					if err != nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", accountID)

					metrics := runCustomNamespaceJob(ctx, jobLogger, customNamespaceJob, region, accountID, cache.GetCloudwatchClient(region, role, cloudWatchAPIConcurrency), metricsPerQuery)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(customNamespaceJob, region, role)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData
}
