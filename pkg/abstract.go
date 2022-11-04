package exporter

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/aws/aws-sdk-go/service/sts"
)

func scrapeAwsData(
	ctx context.Context,
	config ScrapeConf,
	metricsPerQuery int,
	cloudwatchSemaphore,
	tagSemaphore chan struct{},
	cache SessionCache,
	logger Logger,
) ([]*taggedResource, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cwData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*taggedResource, 0)
	var wg sync.WaitGroup

	// since we have called refresh, we have loaded all the credentials
	// into the clients and it is now safe to call concurrently. Defer the
	// clearing, so we always clear credentials before the next scrape
	cache.Refresh()
	defer cache.Clear()

	for _, discoveryJob := range config.Discovery.Jobs {
		for _, role := range discoveryJob.Roles {
			for _, region := range discoveryJob.Regions {
				wg.Add(1)
				go func(discoveryJob *Job, region string, role Role) {
					defer wg.Done()
					jobLogger := logger.With("job_type", discoveryJob.Type, "region", region, "arn", role.RoleArn)
					result, err := cache.GetSTS(role).GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
					if err != nil || result.Account == nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", *result.Account)

					clientCloudwatch := cloudwatchInterface{
						client: cache.GetCloudwatch(&region, role),
						logger: jobLogger,
					}

					clientTag := tagsInterface{
						client:           cache.GetTagging(&region, role),
						apiGatewayClient: cache.GetAPIGateway(&region, role),
						asgClient:        cache.GetASG(&region, role),
						dmsClient:        cache.GetDMS(&region, role),
						ec2Client:        cache.GetEC2(&region, role),
						logger:           jobLogger,
					}

					resources, metrics := scrapeDiscoveryJobUsingMetricData(ctx, discoveryJob, region, result.Account, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch, metricsPerQuery, discoveryJob.RoundingPeriod, tagSemaphore, jobLogger)
					if len(resources) != 0 && len(metrics) != 0 {
						mux.Lock()
						awsInfoData = append(awsInfoData, resources...)
						cwData = append(cwData, metrics...)
						mux.Unlock()
					}
				}(discoveryJob, region, role)
			}
		}
	}

	for _, staticJob := range config.Static {
		for _, role := range staticJob.Roles {
			for _, region := range staticJob.Regions {
				wg.Add(1)
				go func(staticJob *Static, region string, role Role) {
					defer wg.Done()
					jobLogger := logger.With("static_job_name", staticJob.Name, "region", region, "arn", role.RoleArn)
					result, err := cache.GetSTS(role).GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
					if err != nil || result.Account == nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", *result.Account)

					clientCloudwatch := cloudwatchInterface{
						client: cache.GetCloudwatch(&region, role),
						logger: jobLogger,
					}

					metrics := scrapeStaticJob(ctx, staticJob, region, result.Account, clientCloudwatch, cloudwatchSemaphore, jobLogger)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(staticJob, region, role)
			}
		}
	}

	for _, customMetricJob := range config.CustomMetrics {
		for _, role := range customMetricJob.Roles {
			for _, region := range customMetricJob.Regions {
				wg.Add(1)
				go func(staticJob *CustomMetrics, region string, role Role) {
					defer wg.Done()
					jobLogger := logger.With("custom_metric_namespace", customMetricJob.Namespace, "region", region, "arn", role.RoleArn)
					result, err := cache.GetSTS(role).GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
					if err != nil || result.Account == nil {
						jobLogger.Error(err, "Couldn't get account Id")
						return
					}
					jobLogger = jobLogger.With("account", *result.Account)

					clientCloudwatch := cloudwatchInterface{
						client: cache.GetCloudwatch(&region, role),
						logger: jobLogger,
					}

					metrics := scrapeCustomMetricJobUsingMetricData(
						ctx,
						customMetricJob,
						region,
						result.Account,
						clientCloudwatch,
						cloudwatchSemaphore,
						tagSemaphore,
						jobLogger,
						metricsPerQuery,
					)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(customMetricJob, region, role)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData
}

func scrapeStaticJob(ctx context.Context, resource *Static, region string, accountId *string, clientCloudwatch cloudwatchInterface, cloudwatchSemaphore chan struct{}, logger Logger) (cw []*cloudwatchData) {
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
			data := cloudwatchData{
				ID:                     &id,
				Metric:                 &metric.Name,
				Namespace:              &resource.Namespace,
				Statistics:             metric.Statistics,
				NilToZero:              metric.NilToZero,
				AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
				CustomTags:             resource.CustomTags,
				Dimensions:             createStaticDimensions(resource.Dimensions),
				Region:                 &region,
				AccountId:              accountId,
			}

			filter := createGetMetricStatisticsInput(
				data.Dimensions,
				&resource.Namespace,
				metric,
				logger,
			)

			data.Points = clientCloudwatch.get(ctx, filter)

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

func GetMetricDataInputLength(job *Job) int64 {
	length := defaultLengthSeconds

	if job.Length > 0 {
		length = job.Length
	}
	for _, metric := range job.Metrics {
		if metric.Length > length {
			length = metric.Length
		}
	}
	return length
}

func getMetricDataForQueries(
	ctx context.Context,
	discoveryJob *Job,
	svc *serviceFilter,
	region string,
	accountId *string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientCloudwatch cloudwatchInterface,
	resources []*taggedResource,
	tagSemaphore chan struct{},
	logger Logger) []cloudwatchData {
	var getMetricDatas []cloudwatchData

	// For every metric of the job
	for _, metric := range discoveryJob.Metrics {
		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}

		metricsList, err := getFullMetricsList(ctx, svc.Namespace, metric, clientCloudwatch)
		<-tagSemaphore

		if err != nil {
			logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", svc.Namespace)
			continue
		}

		if len(resources) == 0 {
			logger.Debug("No resources for metric", "metric_name", metric.Name, "namespace", svc.Namespace)
		}
		getMetricDatas = append(getMetricDatas, getFilteredMetricDatas(region, accountId, discoveryJob.Type, discoveryJob.CustomTags, tagsOnMetrics, svc.DimensionRegexps, resources, metricsList.Metrics, discoveryJob.DimensionNameRequirements, metric)...)
	}
	return getMetricDatas
}

func scrapeDiscoveryJobUsingMetricData(
	ctx context.Context,
	job *Job,
	region string,
	accountId *string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientTag tagsInterface,
	clientCloudwatch cloudwatchInterface,
	metricsPerQuery int,
	roundingPeriod *int64,
	tagSemaphore chan struct{},
	logger Logger) (resources []*taggedResource, cw []*cloudwatchData) {

	// Add the info tags of all the resources
	tagSemaphore <- struct{}{}
	resources, err := clientTag.get(ctx, job, region)
	<-tagSemaphore
	if err != nil {
		logger.Error(err, "Couldn't describe resources")
		return
	}

	if len(resources) == 0 {
		logger.Info("No tagged resources made it through filtering")
		return
	}

	svc := SupportedServices.GetService(job.Type)
	getMetricDatas := getMetricDataForQueries(ctx, job, svc, region, accountId, tagsOnMetrics, clientCloudwatch, resources, tagSemaphore, logger)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Debug("No metrics data found")
		return
	}

	maxMetricCount := metricsPerQuery
	length := GetMetricDataInputLength(job)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))

	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(partition)

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			defer wg.Done()
			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			input := getMetricDatas[i:end]
			filter := createGetMetricDataInput(input, &svc.Namespace, length, job.Delay, roundingPeriod, logger)
			data := clientCloudwatch.getMetricData(ctx, filter)
			if data != nil {
				output := make([]*cloudwatchData, 0)
				for _, MetricDataResult := range data.MetricDataResults {
					getMetricData, err := findGetMetricDataById(input, *MetricDataResult.Id)
					if err == nil {
						if len(MetricDataResult.Values) != 0 {
							getMetricData.GetMetricDataPoint = MetricDataResult.Values[0]
							getMetricData.GetMetricDataTimestamps = MetricDataResult.Timestamps[0]
						}
						output = append(output, &getMetricData)
					}
				}
				mux.Lock()
				cw = append(cw, output...)
				mux.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return resources, cw
}

func scrapeCustomMetricJobUsingMetricData(
	ctx context.Context,
	customMetricsJob *CustomMetrics,
	region string,
	accountId *string,
	clientCloudwatch cloudwatchInterface,
	cloudwatchSemaphore chan struct{},
	tagSemaphore chan struct{},
	logger Logger,
	metricsPerQuery int) (cw []*cloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	getMetricDatas := getMetricDataForQueriesForCustomMetrics(ctx, customMetricsJob, region, accountId, clientCloudwatch, tagSemaphore, logger)
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		logger.Debug("No metrics data found")
		return
	}

	maxMetricCount := metricsPerQuery
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))

	wg.Add(partition)

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			cloudwatchSemaphore <- struct{}{}

			defer func() {
				defer wg.Done()
				<-cloudwatchSemaphore
			}()

			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			input := getMetricDatas[i:end]
			filter := createGetMetricDataInput(input, &customMetricsJob.Namespace, customMetricsJob.Length, customMetricsJob.Delay, customMetricsJob.RoundingPeriod, logger)
			data := clientCloudwatch.getMetricData(ctx, filter)
			if data != nil {
				output := make([]*cloudwatchData, 0)
				for _, MetricDataResult := range data.MetricDataResults {
					getMetricData, err := findGetMetricDataById(input, *MetricDataResult.Id)
					if err == nil {
						if len(MetricDataResult.Values) != 0 {
							getMetricData.GetMetricDataPoint = MetricDataResult.Values[0]
							getMetricData.GetMetricDataTimestamps = MetricDataResult.Timestamps[0]
						}
						output = append(output, &getMetricData)
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

func getMetricDataForQueriesForCustomMetrics(
	ctx context.Context,
	customMetricJob *CustomMetrics,
	region string,
	accountId *string,
	clientCloudwatch cloudwatchInterface,
	tagSemaphore chan struct{},
	logger Logger) []cloudwatchData {
	var getMetricDatas []cloudwatchData

	// For every metric of the job
	for _, metric := range customMetricJob.Metrics {
		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}

		metricsList, err := getFullMetricsList(ctx, customMetricJob.Namespace, metric, clientCloudwatch)
		<-tagSemaphore

		if err != nil {
			logger.Error(err, "Failed to get full metric list", "metric_name", metric.Name, "namespace", customMetricJob.Namespace)
			continue
		}

		for _, cwMetric := range metricsList.Metrics {
			if len(customMetricJob.DimensionNameRequirements) > 0 && !metricDimensionsMatchNames(cwMetric, customMetricJob.DimensionNameRequirements) {
				continue
			}

			for _, stats := range metric.Statistics {
				id := fmt.Sprintf("id_%d", rand.Int())
				getMetricDatas = append(getMetricDatas, cloudwatchData{
					ID:                     &customMetricJob.Name,
					MetricID:               &id,
					Metric:                 &metric.Name,
					Namespace:              &customMetricJob.Namespace,
					Statistics:             []string{stats},
					NilToZero:              metric.NilToZero,
					AddCloudwatchTimestamp: metric.AddCloudwatchTimestamp,
					CustomTags:             customMetricJob.CustomTags,
					Dimensions:             cwMetric.Dimensions,
					Region:                 &region,
					AccountId:              accountId,
					Period:                 metric.Period,
				})
			}
		}
	}
	return getMetricDatas
}
