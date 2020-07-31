package main

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var (
	cloudwatchSemaphore chan struct{}
	tagSemaphore        chan struct{}
)

func scrapeAwsData(config conf) ([]*tagsData, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cwData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*tagsData, 0)

	var wg sync.WaitGroup

	for _, discoveryJob := range config.Discovery.Jobs {
		for _, roleArn := range discoveryJob.RoleArns {
			for _, region := range discoveryJob.Regions {
				wg.Add(1)

				go func(discoveryJob job, region string, roleArn string) {
					defer wg.Done()
					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, roleArn),
					}

					clientTag := tagsInterface{
						client:           createTagSession(&region, roleArn),
						apiGatewayClient: createAPIGatewaySession(&region, roleArn),
						asgClient:        createASGSession(&region, roleArn),
						ec2Client:        createEC2Session(&region, roleArn),
					}
					var resources []*tagsData
					var metrics []*cloudwatchData
					resources, metrics = scrapeDiscoveryJobUsingMetricData(discoveryJob, region, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch)
					mux.Lock()
					awsInfoData = append(awsInfoData, resources...)
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(discoveryJob, region, roleArn)
			}
		}
	}

	for _, staticJob := range config.Static {
		for _, roleArn := range staticJob.RoleArns {
			for _, region := range staticJob.Regions {
				wg.Add(1)

				go func(staticJob static, region string, roleArn string) {
					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, roleArn),
					}

					metrics := scrapeStaticJob(staticJob, region, clientCloudwatch)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()

					wg.Done()
				}(staticJob, region, roleArn)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData
}

func scrapeStaticJob(resource static, region string, clientCloudwatch cloudwatchInterface) (cw []*cloudwatchData) {
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
			service := strings.TrimPrefix(resource.Namespace, "AWS/")
			data := cloudwatchData{
				ID:                     &id,
				Metric:                 &metric.Name,
				Service:                &service,
				Statistics:             metric.Statistics,
				NilToZero:              &metric.NilToZero,
				AddCloudwatchTimestamp: &metric.AddCloudwatchTimestamp,
				CustomTags:             resource.CustomTags,
				Dimensions:             createStaticDimensions(resource.Dimensions),
				Region:                 &region,
			}

			filter := createGetMetricStatisticsInput(
				data.Dimensions,
				&resource.Namespace,
				metric,
			)

			data.Points = clientCloudwatch.get(filter)

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

func getMetricDataInputLength(job job) int {
	var length int

	// Why is this here? 120?
	if job.Length == 0 {
		length = 120
	} else {
		length = job.Length
	}
	for _, metric := range job.Metrics {
		if metric.Length > length {
			length = metric.Length
		}
	}
	return length
}

func getMetricPeriod(job job, metric metric) int64 {
	if metric.Period != 0 {
		return int64(metric.Period)
	}
	if job.Period != 0 {
		return int64(job.Period)
	}
	return int64(300)
}

func scrapeDiscoveryJobUsingMetricData(
	job job,
	region string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientTag tagsInterface,
	clientCloudwatch cloudwatchInterface) (awsInfoData []*tagsData, cw []*cloudwatchData) {

	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	var getMetricDatas []cloudwatchData

	tagSemaphore <- struct{}{}
	resources, err := clientTag.get(job, region)
	<-tagSemaphore

	// Add the info tags of all the resources
	for _, resource := range resources {
		mux.Lock()
		awsInfoData = append(awsInfoData, resource)
		mux.Unlock()
	}

	if err != nil {
		log.Printf("Couldn't describe resources for region %s: %s\n", region, err.Error())
		return
	}
	// Get the awsDimensions of the job configuration
	// Common for all the metrics of the job
	commonJobDimensions := getAwsDimensions(job)

	// For every metric of the job
	for j := range job.Metrics {
		metric := job.Metrics[j]

		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}
		fullMetricsList := getFullMetricsList(&job.Type, metric, clientCloudwatch)
		<-tagSemaphore

		// For every resource
		for i := range resources {
			resource := resources[i]
			metricTags := resource.metricTags(tagsOnMetrics)

			// Creates the dimensions with values for the resource depending on the namespace of the job (p.e. InstanceId=XXXXXXX)
			dimensionsWithValue := detectDimensionsByService(resource, fullMetricsList)

			// Adds the dimensions with values of that specific metric of the job
			dimensionsWithValue = addAdditionalDimensions(dimensionsWithValue, metric.AdditionalDimensions)

			// Filter the commonJob Dimensions by the discovered/added dimensions as duplicates cause no metrics to be discovered
			commonJobDimensions = filterDimensionsWithoutValueByDimensionsWithValue(commonJobDimensions, dimensionsWithValue)

			metricsToAdd := filterMetricsBasedOnDimensionsWithValues(dimensionsWithValue, commonJobDimensions, fullMetricsList)

			// If the job property inlyInfoIfData is true
			if metricsToAdd != nil {
				for _, fetchedMetrics := range metricsToAdd.Metrics {
					for _, stats := range metric.Statistics {
						id := fmt.Sprintf("id_%d", rand.Int())

						addCloudwatchTimestamp := job.AddCloudwatchTimestamp || metric.AddCloudwatchTimestamp

						mux.Lock()
						getMetricDatas = append(getMetricDatas, cloudwatchData{
							ID:                     resource.ID,
							MetricID:               &id,
							Metric:                 &metric.Name,
							Service:                resource.Service,
							Statistics:             []string{stats},
							NilToZero:              &metric.NilToZero,
							AddCloudwatchTimestamp: &addCloudwatchTimestamp,
							Tags:                   metricTags,
							CustomTags:             job.CustomTags,
							Dimensions:             fetchedMetrics.Dimensions,
							Region:                 &region,
							Period:                 getMetricPeriod(job, metric),
						})
						mux.Unlock()
					}
				}
			}
		}
	}
	wg.Wait()
	maxMetricCount := *metricsPerQuery
	metricDataLength := len(getMetricDatas)
	length := getMetricDataInputLength(job)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))
	wg.Add(partition)
	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			defer wg.Done()
			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			filter := createGetMetricDataInput(
				getMetricDatas[i:end],
				getNamespace(resources[0].Service),
				length,
				job.Delay,
			)

			data := clientCloudwatch.getMetricData(filter)
			if data != nil {
				for _, MetricDataResult := range data.MetricDataResults {
					getMetricData, err := findGetMetricDataById(getMetricDatas[i:end], *MetricDataResult.Id)
					if err == nil {
						if len(MetricDataResult.Values) != 0 {
							getMetricData.GetMetricDataPoint = MetricDataResult.Values[0]
							getMetricData.GetMetricDataTimestamps = MetricDataResult.Timestamps[0]
						}
						mux.Lock()
						cw = append(cw, &getMetricData)
						mux.Unlock()
					}
				}
			}
		}(i)
	}
	wg.Wait()
	return awsInfoData, cw
}

func (r tagsData) filterThroughTags(filterTags []tag) bool {
	tagMatches := 0

	for _, resourceTag := range r.Tags {
		for _, filterTag := range filterTags {
			if resourceTag.Key == filterTag.Key {
				r, _ := regexp.Compile(filterTag.Value)
				if r.MatchString(resourceTag.Value) {
					tagMatches++
				}
			}
		}
	}

	return tagMatches == len(filterTags)
}

func (r tagsData) metricTags(tagsOnMetrics exportedTagsOnMetrics) []tag {
	tags := make([]tag, 0)
	for _, tagName := range tagsOnMetrics[*r.Service] {
		tag := tag{
			Key: tagName,
		}
		for _, resourceTag := range r.Tags {
			if resourceTag.Key == tagName {
				tag.Value = resourceTag.Value
				break
			}
		}

		// Always add the tag, even if it's empty, to ensure the same labels are present on all metrics for a single service
		tags = append(tags, tag)
	}
	return tags
}
