package main

import (
	_ "fmt"
	"log"
	"regexp"
	"strings"
	"sync"
)

func scrapeAwsData(config conf) ([]*tagsData, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cloudwatchData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*tagsData, 0)

	var wg sync.WaitGroup
	for i := range config.Discovery {
		wg.Add(1)
		job := config.Discovery[i]
		go func() {
			region := &job.Region

			clientCloudwatch := cloudwatchInterface{
				client: createCloudwatchSession(region),
			}

			clientTag := tagsInterface{
				client: createTagSession(region),
			}

			resources, metrics := scrapeDiscoveryJob(job, clientTag, clientCloudwatch)

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

			clientCloudwatch := cloudwatchInterface{
				client: createCloudwatchSession(region),
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
			id := resource.Name
			service := strings.TrimPrefix(resource.Namespace, "AWS/")
			data := cloudwatchData{
				ID:         &id,
				Metric:     &metric.Name,
				Service:    &service,
				Statistics: metric.Statistics,
				NilToZero:  &metric.NilToZero,
				CustomTags: resource.CustomTags,
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
			wg.Done()
		}()
	}
	wg.Wait()
	return cw
}

func scrapeDiscoveryJob(job discovery, clientTag tagsInterface, clientCloudwatch cloudwatchInterface) (awsInfoData []*tagsData, cw []*cloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	resources, err := clientTag.get(job)
	if err != nil {
		log.Println("Couldn't describe resources: ", err.Error())
		return
	}

	for i := range resources {
		resource := resources[i]
		awsInfoData = append(awsInfoData, resource)

		for j := range job.Metrics {
			metric := job.Metrics[j]
			wg.Add(1)
			go func() {
				data := cloudwatchData{
					ID:         resource.ID,
					Metric:     &metric.Name,
					Service:    resource.Service,
					Statistics: metric.Statistics,
					NilToZero:  &metric.NilToZero,
				}

				filter := createGetMetricStatisticsInput(
					getDimensions(resource.Service, resource.ID, clientCloudwatch),
					getNamespace(resource.Service),
					metric,
				)

				data.Points = clientCloudwatch.get(filter)

				mux.Lock()
				cw = append(cw, &data)
				mux.Unlock()
				wg.Done()
			}()
		}
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
