package main

import (
	_ "fmt"
	"log"
	"regexp"
	"strings"
	"sync"
)

var (
	cloudwatchSemaphore = make(chan struct{}, 5)
	tagSemaphore        = make(chan struct{}, 5)
)

func scrapeAwsData(config conf) ([]*tagsData, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cloudwatchData := make([]*cloudwatchData, 0)
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

			resources, metrics := scrapeDiscoveryJob(job, config.Discovery.ExportedTagsOnMetrics, clientTag, clientCloudwatch)

			mux.Lock()
			awsInfoData = append(awsInfoData, resources...)
			cloudwatchData = append(cloudwatchData, metrics...)
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
			cloudwatchData = append(cloudwatchData, metrics...)
			mux.Unlock()

			wg.Done()
		}()
	}
	wg.Wait()
	return awsInfoData, cloudwatchData
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
			}

			filter := createGetMetricStatisticsInput(
				createStaticDimensions(resource.Dimensions),
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

func scrapeDiscoveryJob(job job, tagsOnMetrics exportedTagsOnMetrics, clientTag tagsInterface, clientCloudwatch cloudwatchInterface) (awsInfoData []*tagsData, cw []*cloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup

	tagSemaphore <- struct{}{}
	defer func() {
		<-tagSemaphore // Unlock
	}()

	resources, err := clientTag.get(job)
	if err != nil {
		log.Println("Couldn't describe resources: ", err.Error())
		return
	}

	for i := range resources {
		resource := resources[i]
		awsInfoData = append(awsInfoData, resource)
		metricTags := resource.metricTags(tagsOnMetrics)

		wg.Add(len(job.Metrics))
		go func() {
			for j := range job.Metrics {
				metric := job.Metrics[j]
				dimensions := detectDimensionsByService(resource.Service, resource.ID, clientCloudwatch)
				dimensions = addAdditionalDimensions(dimensions, metric.AdditionalDimensions)
				go func() {
					defer wg.Done()

					cloudwatchSemaphore <- struct{}{}
					defer func() {
						<-cloudwatchSemaphore
					}()

					data := cloudwatchData{
						ID:                     resource.ID,
						Metric:                 &metric.Name,
						Service:                resource.Service,
						Statistics:             metric.Statistics,
						NilToZero:              &metric.NilToZero,
						AddCloudwatchTimestamp: &metric.AddCloudwatchTimestamp,
						Tags:                   metricTags,
						Dimensions:             dimensions,
					}

					filter := createGetMetricStatisticsInput(
						dimensions,
						getNamespace(resource.Service),
						metric,
					)

					data.Points = clientCloudwatch.get(filter)

					mux.Lock()
					cw = append(cw, &data)
					mux.Unlock()
				}()
			}
		}()
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
