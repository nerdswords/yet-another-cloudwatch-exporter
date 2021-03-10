package main

import (
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
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
	var endtime time.Time
	var wg sync.WaitGroup

	for _, discoveryJob := range config.Discovery.Jobs {
		for _, roleArn := range discoveryJob.RoleArns {
			clientSts := createStsSession(roleArn)
			result, err := clientSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
			if err != nil {
				log.Printf("Couldn't get account Id for role %s: %s\n", roleArn, err.Error())
			}
			accountId := result.Account
			for _, region := range discoveryJob.Regions {
				wg.Add(1)

				go func(discoveryJob *job, region string, accountId *string, roleArn string) {
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
					resources, metrics, endtime = scrapeDiscoveryJobUsingMetricData(discoveryJob, region, accountId, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch, now)
					mux.Lock()
					awsInfoData = append(awsInfoData, resources...)
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(discoveryJob, region, accountId, roleArn)
			}
		}
	}

	for _, staticJob := range config.Static {
		for _, roleArn := range staticJob.RoleArns {
			clientSts := createStsSession(roleArn)
			result, err := clientSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
			if err != nil {
				log.Printf("Couldn't get account Id for role %s: %s\n", roleArn, err.Error())
			}
			accountId := result.Account
			for _, region := range staticJob.Regions {
				wg.Add(1)

				go func(staticJob *static, region string, accountId *string, roleArn string) {
					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, roleArn),
					}

					metrics := scrapeStaticJob(staticJob, region, accountId, clientCloudwatch)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()

					wg.Done()
				}(staticJob, region, accountId, roleArn)
			}
		}
	}
	wg.Wait()
	return awsInfoData, cwData, &endtime
}

func scrapeStaticJob(resource *static, region string, accountId *string, clientCloudwatch cloudwatchInterface) (cw []*cloudwatchData) {
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

func getMetricDataInputLength(job *job) int {
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
	discoveryJob *job,
	svc *serviceFilter,
	region string,
	accountId *string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientCloudwatch cloudwatchInterface,
	resources []*tagsData) []cloudwatchData {
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
	job *job,
	region string,
	accountId *string,
	tagsOnMetrics exportedTagsOnMetrics,
	clientTag tagsInterface,
	clientCloudwatch cloudwatchInterface, now time.Time) (resources []*tagsData, cw []*cloudwatchData, endtime time.Time) {

	// Add the info tags of all the resources
	tagSemaphore <- struct{}{}
	resources, err := clientTag.get(job, region)
	<-tagSemaphore
	if err != nil {
		log.Printf("Couldn't describe resources for region %s: %s\n", region, err.Error())
		return
	}

	svc := supportedServices.getService(job.Type)
	getMetricDatas := getMetricDataForQueries(job, svc, region, accountId, tagsOnMetrics, clientCloudwatch, resources)
	maxMetricCount := *metricsPerQuery
	metricDataLength := len(getMetricDatas)
	length := getMetricDataInputLength(job)
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
			filter := createGetMetricDataInput(getMetricDatas[i:end], &svc.Namespace, length, job.Delay, now)
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
			endtime = *filter.EndTime
		}(i)
	}
	//here set end time as start time
	wg.Wait()
	return resources, cw, endtime
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
	for _, tagName := range tagsOnMetrics[*r.Namespace] {
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
