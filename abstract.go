package main

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	cloudwatchSemaphore chan struct{}
	tagSemaphore        chan struct{}
)

func scrapeAwsData(config conf, now time.Time) ([]*tagsData, []*cloudwatchData, *time.Time) {
	mux := &sync.Mutex{}

	cwData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*tagsData, 0)
	var nendtime time.Time
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
						elbv2Client:      createELBV2Session(&region, roleArn),
					}
					var resources []*tagsData
					var metrics []*cloudwatchData
					resources, metrics, nendtime = scrapeDiscoveryJobUsingMetricData(discoveryJob, region, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch, now)
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
	return awsInfoData, cwData, &nendtime
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

func getMetricDataForQueries(
	discoveryJob job,
	region string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientCloudwatch cloudwatchInterface,
	resources []*tagsData) []cloudwatchData {
	var getMetricDatas []cloudwatchData

	// Get the awsDimensions of the job configuration
	// Common for all the metrics of the job
	commonJobDimensions := getAwsDimensions(discoveryJob)
	namespace, _ := getNamespace(discoveryJob.Type)
	// For every metric of the job
	for _, metric := range discoveryJob.Metrics {
		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}
		fullMetricsList := getFullMetricsList(namespace, metric, clientCloudwatch)
		<-tagSemaphore

		// For every resource
		for _, resource := range resources {
			// Creates the dimensions with values for the resource depending on the namespace of the job (p.e. InstanceId=XXXXXXX)
			dimensionsWithValue := detectDimensionsByService(resource, fullMetricsList)

			// Adds the dimensions with values of that specific metric of the job
			dimensionsWithValue = addAdditionalDimensions(dimensionsWithValue, metric.AdditionalDimensions)

			// Filter the commonJob Dimensions by the discovered/added dimensions as duplicates cause no metrics to be discovered
			commonJobDimensions = filterDimensionsWithoutValueByDimensionsWithValue(commonJobDimensions, dimensionsWithValue)

			metricsToAdd := filterMetricsBasedOnDimensionsWithValues(dimensionsWithValue, commonJobDimensions, fullMetricsList)
			if metricsToAdd != nil && len(metricsToAdd.Metrics) > 0 {
				addCloudwatchTimestamp := discoveryJob.AddCloudwatchTimestamp || metric.AddCloudwatchTimestamp
				metricTags := resource.metricTags(tagsOnMetrics)
				for _, fetchedMetrics := range metricsToAdd.Metrics {
					for _, stats := range metric.Statistics {
						id := fmt.Sprintf("id_%d", rand.Int())
						name := metric.Name
						nilToZero := metric.NilToZero
						getMetricDatas = append(getMetricDatas, cloudwatchData{
							ID:                     resource.ID,
							MetricID:               &id,
							Metric:                 &name,
							Service:                resource.Service,
							Statistics:             []string{stats},
							NilToZero:              &nilToZero,
							AddCloudwatchTimestamp: &addCloudwatchTimestamp,
							Tags:                   metricTags,
							CustomTags:             discoveryJob.CustomTags,
							Dimensions:             fetchedMetrics.Dimensions,
							Region:                 &region,
							Period:                 getMetricPeriod(discoveryJob, metric),
						})
					}
				}
			}
		}
	}
	return getMetricDatas
}

func scrapeDiscoveryJobUsingMetricData(
	job job,
	region string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientTag tagsInterface,
	clientCloudwatch cloudwatchInterface, now time.Time) (resources []*tagsData, cw []*cloudwatchData, nendtime time.Time) {

	namespace, err := getNamespace(job.Type)
	if err != nil {
		log.Fatal(err.Error())
	}
	// Add the info tags of all the resources
	tagSemaphore <- struct{}{}
	resources, err = clientTag.get(job, region)
	<-tagSemaphore
	if err != nil {
		log.Printf("Couldn't describe resources for region %s: %s\n", region, err.Error())
		return
	}

	getMetricDatas := getMetricDataForQueries(job, region, tagsOnMetrics, clientCloudwatch, resources)
	maxMetricCount := *metricsPerQuery
	metricDataLength := len(getMetricDatas)
	length := getMetricDataInputLength(job)
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
			filter := createGetMetricDataInput(getMetricDatas[i:end], &namespace, length, job.Delay, now)
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
			nendtime = *filter.EndTime

			log.Println("Start time ", *filter.StartTime, "End time ", *filter.EndTime)
		}(i)
	}
	//here set end time as start time
	wg.Wait()
	return resources, cw, nendtime
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
