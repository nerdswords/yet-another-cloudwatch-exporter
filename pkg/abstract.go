package exporter

import (
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	log "github.com/sirupsen/logrus"
)

func scrapeAwsData(config ScrapeConf, now time.Time, metricsPerQuery int, fips, debug, floatingTimeWindow bool, cloudwatchSemaphore, tagSemaphore chan struct{}) ([]*tagsData, []*cloudwatchData, *time.Time) {
	mux := &sync.Mutex{}

	cwData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*tagsData, 0)
	var endtime time.Time
	var wg sync.WaitGroup

	for _, discoveryJob := range config.Discovery.Jobs {
		for _, roleArn := range discoveryJob.RoleArns {
			for _, region := range discoveryJob.Regions {
				wg.Add(1)

				go func(discoveryJob *Job, region string, roleArn string, debug bool) {
					defer wg.Done()
					clientSts := createStsSession(roleArn, debug)
					result, err := clientSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
					if err != nil {
						log.Printf("Couldn't get account Id for role %s: %s\n", roleArn, err.Error())
					}
					accountId := result.Account

					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, roleArn, fips, debug),
					}

					clientTag := tagsInterface{
						client:           createTagSession(&region, roleArn, fips),
						apiGatewayClient: createAPIGatewaySession(&region, roleArn, fips),
						asgClient:        createASGSession(&region, roleArn, fips),
						ec2Client:        createEC2Session(&region, roleArn, fips),
					}
					var resources []*tagsData
					var metrics []*cloudwatchData
					resources, metrics, endtime = scrapeDiscoveryJobUsingMetricData(discoveryJob, region, accountId, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch, now, metricsPerQuery, debug, floatingTimeWindow, tagSemaphore)
					mux.Lock()
					awsInfoData = append(awsInfoData, resources...)
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(discoveryJob, region, roleArn, debug)
			}
		}
	}

	for _, staticJob := range config.Static {
		for _, roleArn := range staticJob.RoleArns {
			for _, region := range staticJob.Regions {
				wg.Add(1)

				go func(staticJob *Static, region string, roleArn string, debug bool) {
					defer wg.Done()
					clientSts := createStsSession(roleArn, debug)
					result, err := clientSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
					if err != nil {
						log.Printf("Couldn't get account Id for role %s: %s\n", roleArn, err.Error())
					}
					accountId := result.Account

					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, roleArn, fips, debug),
					}

					metrics := scrapeStaticJob(staticJob, region, accountId, clientCloudwatch, cloudwatchSemaphore)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(staticJob, region, roleArn, debug)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData, &endtime
}

func scrapeStaticJob(resource *Static, region string, accountId *string, clientCloudwatch cloudwatchInterface, cloudwatchSemaphore chan struct{}) (cw []*cloudwatchData) {
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
			)

			data.Points = clientCloudwatch.get(filter)

			if data.Points != nil || *data.NilToZero {
				mux.Lock()
				cw = append(cw, &data)
				mux.Unlock()
			}
		}()
	}
	wg.Wait()
	return cw
}

func GetMetricDataInputLength(job *Job) int {
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

func getMetricDataForQueries(
	discoveryJob *Job,
	svc *serviceFilter,
	region string,
	accountId *string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientCloudwatch cloudwatchInterface,
	resources []*tagsData,
	tagSemaphore chan struct{}) []cloudwatchData {
	var getMetricDatas []cloudwatchData

	// For every metric of the job
	for _, metric := range discoveryJob.Metrics {
		// Get the full list of metrics
		// This includes, for this metric the possible combinations
		// of dimensions and value of dimensions with data
		tagSemaphore <- struct{}{}

		metricsList := getFullMetricsList(svc.Namespace, metric, clientCloudwatch)
		<-tagSemaphore
		if len(resources) == 0 {
			log.Debugf("No resources for metric %s on %s job", metric.Name, svc.Namespace)
		}
		getMetricDatas = append(getMetricDatas, getFilteredMetricDatas(region, accountId, discoveryJob.Type, discoveryJob.CustomTags, tagsOnMetrics, svc.DimensionRegexps, resources, metricsList.Metrics, metric)...)
	}
	return getMetricDatas
}

func scrapeDiscoveryJobUsingMetricData(
	job *Job,
	region string,
	accountId *string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientTag tagsInterface,
	clientCloudwatch cloudwatchInterface, now time.Time,
	metricsPerQuery int, debug, floatingTimeWindow bool,
	tagSemaphore chan struct{}) (resources []*tagsData, cw []*cloudwatchData, endtime time.Time) {

	// Add the info tags of all the resources
	tagSemaphore <- struct{}{}
	resources, err := clientTag.get(job, region)
	<-tagSemaphore
	if err != nil {
		log.Printf("Couldn't describe resources for region %s: %s\n", region, err.Error())
		return
	}

	svc := SupportedServices.GetService(job.Type)
	getMetricDatas := getMetricDataForQueries(job, svc, region, accountId, tagsOnMetrics, clientCloudwatch, resources, tagSemaphore)
	maxMetricCount := metricsPerQuery
	metricDataLength := len(getMetricDatas)
	length := GetMetricDataInputLength(job)
	partition := int(math.Ceil(float64(metricDataLength) / float64(maxMetricCount)))

	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(partition)

	if metricDataLength == 0 {
		log.Debugf("No metrics data for %s", job.Type)
	}

	for i := 0; i < metricDataLength; i += maxMetricCount {
		go func(i int) {
			defer wg.Done()
			end := i + maxMetricCount
			if end > metricDataLength {
				end = metricDataLength
			}
			filter := createGetMetricDataInput(getMetricDatas[i:end], &svc.Namespace, length, job.Delay, now, floatingTimeWindow)
			data := clientCloudwatch.getMetricData(filter, debug)
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
			endtime = *filter.EndTime
		}(i)
	}
	//here set end time as start time
	wg.Wait()
	return resources, cw, endtime
}

func (r tagsData) filterThroughTags(filterTags []Tag) bool {
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

func (r tagsData) metricTags(tagsOnMetrics exportedTagsOnMetrics) []Tag {
	tags := make([]Tag, 0)
	for _, tagName := range tagsOnMetrics[*r.Namespace] {
		tag := Tag{
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
