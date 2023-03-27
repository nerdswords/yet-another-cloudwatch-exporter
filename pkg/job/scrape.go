package job

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/session"
)

func ScrapeAwsData(
	ctx context.Context,
	cfg config.ScrapeConf,
	metricsPerQuery int,
	cloudwatchSemaphore,
	tagSemaphore chan struct{},
	cache session.SessionCache,
	logger logging.Logger,
) ([]*model.TaggedResource, []*model.CloudwatchData, []*promutil.PrometheusMetric) {
	mux := &sync.Mutex{}

	cwData := make([]*model.CloudwatchData, 0)
	awsInfoData := make([]*model.TaggedResource, 0)
	additionalMetrics := make([]*promutil.PrometheusMetric, 0)
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
					result, err := cache.GetSTS(role).GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
					if err != nil || result.Account == nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", *result.Account)

					resources, metrics, otherOutput := runDiscoveryJob(ctx, jobLogger, cache, metricsPerQuery, tagSemaphore, discoveryJob, region, role, result.Account, cfg.Discovery.ExportedTagsOnMetrics)
					if len(resources) != 0 && len(metrics) != 0 {
						mux.Lock()
						awsInfoData = append(awsInfoData, resources...)
						cwData = append(cwData, metrics...)
						additionalMetrics = append(additionalMetrics, otherOutput...)
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
					result, err := cache.GetSTS(role).GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
					if err != nil || result.Account == nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", *result.Account)

					metrics := runStaticJob(ctx, jobLogger, cache, region, role, staticJob, result.Account, cloudwatchSemaphore)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(staticJob, region, role)
			}
		}
	}

	for _, customNamespaceJob := range cfg.CustomNamespace {
		for _, role := range customNamespaceJob.Roles {
			for _, region := range customNamespaceJob.Regions {
				wg.Add(1)
				go func(customNamespaceJob *config.CustomNamespace, region string, role config.Role) {
					defer wg.Done()
					jobLogger := logger.With("custom_metric_namespace", customNamespaceJob.Namespace, "region", region, "arn", role.RoleArn)
					result, err := cache.GetSTS(role).GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
					if err != nil || result.Account == nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", *result.Account)

					metrics := runCustomNamespaceJob(ctx, jobLogger, cache, metricsPerQuery, cloudwatchSemaphore, tagSemaphore, customNamespaceJob, region, role, result.Account)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(customNamespaceJob, region, role)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData, additionalMetrics
}
