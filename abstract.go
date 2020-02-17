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

	for i := range config.Discovery.Jobs {
		wg.Add(1)
		job := config.Discovery.Jobs[i]
		go func() {
			region := &job.Region
			roleArn := job.RoleArn

			clientCloudwatch := cloudwatchInterface{
				client: createCloudwatchSession(region, roleArn),
			}

			clientTag := tagsInterface{
				client:    createTagSession(region, roleArn),
				asgClient: createASGSession(region, roleArn),
			}
			var resources []*tagsData
			var metrics []*cloudwatchData
			resources, metrics = scrapeDiscoveryJobUsingMetricData(job, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch)
			mux.Lock()
			awsInfoData = append(awsInfoData, resources...)
			cwData = append(cwData, metrics...)
			mux.Unlock()
			wg.Done()

		}()
	}

	for i := range config.Static {
		wg.Add(1)
		job := config.Static[i]
		go func() {
			region := &job.Region
			roleArn := job.RoleArn

			clientCloudwatch := cloudwatchInterface{
				client: createCloudwatchSession(region, roleArn),
			}

			metrics := scrapeStaticJob(job, clientCloudwatch)

			mux.Lock()
			cwData = append(cwData, metrics...)
			mux.Unlock()

			wg.Done()
		}()
	}
	wg.Wait()
	return awsInfoData, cwData
}

func scrapeStaticJob(resource static, clientCloudwatch cloudwatchInterface) (cw []*cloudwatchData) {
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
				Region:                 &resource.Region,
			}

			filter := createGetMetricStatisticsInput(
				data.Dimensions,
				&resource.Namespace,
				metric,
			)

			data.Points = clientCloudwatch.get(filter)

			mux.Lock()
			cw = append(cw, &data)
			mux.Unlock()
		}()
	}
	wg.Wait()
	return cw
}

func scrapeDiscoveryJobUsingMetricData(job job, tagsOnMetrics exportedTagsOnMetrics, clientTag tagsInterface, clientCloudwatch cloudwatchInterface) (awsInfoData []*tagsData, cw []*cloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	var getMetricDatas []cloudwatchData
	var length int

	if job.Length == 0 {
		length = 120
	} else {
		length = job.Length
	}
	tagSemaphore <- struct{}{}
	resources, err := clientTag.get(job)
	<-tagSemaphore

	if err != nil {
		log.Printf("Couldn't describe resources for region %s: %s\n", job.Region, err.Error())
		return
	}
	commonJobDimensions := getAwsDimensions(job)

	for i := range resources {
		resource := resources[i]
		mux.Lock()
		awsInfoData = append(awsInfoData, resource)
		mux.Unlock()
		metricTags := resource.metricTags(tagsOnMetrics)
		dimensions := detectDimensionsByService(resource.Service, resource.ID, clientCloudwatch)
		for _, commonJobDimension := range commonJobDimensions {
			dimensions = append(dimensions, commonJobDimension)
		}

		wg.Add(len(job.Metrics))
		go func() {
			for j := range job.Metrics {
				metric := job.Metrics[j]
				dimensions = addAdditionalDimensions(dimensions, metric.AdditionalDimensions)
				resp := getMetricsList(dimensions, resource.Service, metric, clientCloudwatch)
				defer wg.Done()
				for _, fetchedMetrics := range resp.Metrics {
					for _, stats := range metric.Statistics {
						id := fmt.Sprintf("id_%d", rand.Int())
						period := int64(metric.Period)
						mux.Lock()
						getMetricDatas = append(getMetricDatas, cloudwatchData{
							ID:                     resource.ID,
							MetricID:               &id,
							Metric:                 &metric.Name,
							Service:                resource.Service,
							Statistics:             []string{stats},
							NilToZero:              &metric.NilToZero,
							AddCloudwatchTimestamp: &metric.AddCloudwatchTimestamp,
							Tags:                   metricTags,
							Dimensions:             fetchedMetrics.Dimensions,
							Region:                 &job.Region,
							Period:                 &period,
						})
						mux.Unlock()
					}
				}
			}
		}()
	}
	wg.Wait()
	maxMetricCount := 100
	metricDataLength := len(getMetricDatas)
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
