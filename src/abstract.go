package main

import (
	_ "fmt"
	"log"
	"regexp"
	"sync"
)

func scrapeAwsData(jobs []job) ([]*tagsData, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cloudwatchData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*tagsData, 0)

	var wg sync.WaitGroup
	for i := range jobs {
		wg.Add(1)
		job := jobs[i]
		go func() {
			region := &job.Discovery.Region

			clientCloudwatch := cloudwatchInterface{
				client: createCloudwatchSession(region),
			}

			clientTag := tagsInterface{
				client: createTagSession(region),
			}

			resources, metrics := scrapeJob(job, clientTag, clientCloudwatch)

			mux.Lock()
			awsInfoData = append(awsInfoData, resources...)
			cloudwatchData = append(cloudwatchData, metrics...)
			mux.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	return awsInfoData, cloudwatchData
}

func scrapeJob(job job, clientTag tagsInterface, clientCloudwatch cloudwatchInterface) (awsInfoData []*tagsData, cw []*cloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	resources, err := clientTag.get(job.Discovery)
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

				data.Points = clientCloudwatch.get(resource.Service, resource.ID, metric)

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
