package exporter

import (
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	log "github.com/sirupsen/logrus"
)

func scrapeAwsData(config ScrapeConf, now time.Time, metricsPerQuery int, fips, floatingTimeWindow bool, cloudwatchSemaphore, tagSemaphore chan struct{}) ([]*tagsData, []*cloudwatchData, *time.Time) {
	mux := &sync.Mutex{}

	cwData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*tagsData, 0)
	var endtime time.Time
	var wg sync.WaitGroup

	for _, discoveryJob := range config.Discovery.Jobs {
		for _, role := range discoveryJob.Roles {
			for _, region := range discoveryJob.Regions {
				wg.Add(1)
				go func(discoveryJob *Job, region string, role Role) {
					defer wg.Done()
					clientSts := createStsSession(role)
					result, err := clientSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
					if err != nil {
						log.Printf("Couldn't get account Id for role %s: %s\n", role.RoleArn, err.Error())

					}
					accountId := result.Account

					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, role, fips),
					}

					clientTag := tagsInterface{
						account:          *accountId,
						client:           createTagSession(&region, role, fips),
						apiGatewayClient: createAPIGatewaySession(&region, role, fips),
						asgClient:        createASGSession(&region, role, fips),
						ec2Client:        createEC2Session(&region, role, fips),
					}

					resources, metrics, end := scrapeDiscoveryJobUsingMetricData(discoveryJob, region, accountId, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch, now, metricsPerQuery, floatingTimeWindow, tagSemaphore)
					mux.Lock()
					awsInfoData = append(awsInfoData, resources...)
					cwData = append(cwData, metrics...)
					endtime = end
					mux.Unlock()
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
					clientSts := createStsSession(role)
					result, err := clientSts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
					if err != nil {
						log.Printf("Couldn't get account Id for role %s: %s\n", role.RoleArn, err.Error())
					}
					accountId := result.Account

					clientCloudwatch := cloudwatchInterface{
						client: createCloudwatchSession(&region, role, fips),
					}

					metrics := scrapeStaticJob(staticJob, region, accountId, clientCloudwatch, cloudwatchSemaphore)

					mux.Lock()
					cwData = append(cwData, metrics...)
					mux.Unlock()
				}(staticJob, region, role)
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
	metricsPerQuery int, floatingTimeWindow bool,
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
	metricDataLength := len(getMetricDatas)
	if metricDataLength == 0 {
		log.Debugf("No metrics data for %s", job.Type)
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
			filter := createGetMetricDataInput(input, &svc.Namespace, length, job.Delay, now, floatingTimeWindow)
			data := clientCloudwatch.getMetricData(filter)
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
			mux.Lock()
			endtime = *filter.EndTime
			mux.Unlock()
		}(i)
	}
	//here set end time as start time
	wg.Wait()
	return resources, cw, endtime
}

func (r tagsData) filterThroughTags(filterTags map[string]string) bool {
	tagMatches := 0

	for _, resourceTag := range r.Tags {
		if _, ok := filterTags[resourceTag]; ok {
			rexp, _ := regexp.Compile(filterTags[resourceTag])
			if rexp.MatchString(r.Tags[resourceTag]) {
				tagMatches++
			}
		}
	}

	return tagMatches == len(filterTags)
}

func (r tagsData) metricTags(tagsOnMetrics exportedTagsOnMetrics) map[string]string {
	tags := make(map[string]string)
	for _, tagName := range tagsOnMetrics[*r.Namespace] {
		// Always add the tag, even if it's empty, to ensure the same labels are present on all metrics for a single service
		tags[tagName] = tags[r.Tags[tagName]]
	}
	return tags
}
