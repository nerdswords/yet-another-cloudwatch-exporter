package main

import (
	_ "fmt"
	"regexp"
	"sync"
)

type awsInfoData struct {
	Id      *string
	Tags    []*tag
	Service *string
	Region  *string
}

func scrapeAwsData(jobs []job) ([]*TagsData, []*CloudwatchData) {
	mux := &sync.Mutex{}

	cloudwatchData := make([]*CloudwatchData, 0)
	awsInfoData := make([]*TagsData, 0)

	var wg sync.WaitGroup
	for i, _ := range jobs {
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

func scrapeJob(job job, clientTag tagsInterface, clientCloudwatch cloudwatchInterface) (awsInfoData []*TagsData, cloudwatchData []*CloudwatchData) {
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	resources := clientTag.get(job.Discovery)

	for i, _ := range resources {
		resource := resources[i]
		awsInfoData = append(awsInfoData, resource)

		for j, _ := range job.Metrics {
			metric := job.Metrics[j]
			wg.Add(1)
			go func() {
				data := CloudwatchData{
					Id:         resource.Id,
					Metric:     &metric.Name,
					Service:    resource.Service,
					Statistics: metric.Statistics,
					NilToZero:  &metric.NilToZero,
				}

				data.Points = clientCloudwatch.get(resource.Service, resource.Id, metric)

				mux.Lock()
				cloudwatchData = append(cloudwatchData, &data)
				mux.Unlock()
				wg.Done()
			}()
		}
	}
	wg.Wait()
	return awsInfoData, cloudwatchData
}

func (r TagsData) filterThroughTags(filterTags []tag) bool {
	tagMatches := 0

	for _, resourceTag := range r.Tags {
		for _, filterTag := range filterTags {
			if resourceTag.Key == filterTag.Key {
				r, _ := regexp.Compile(filterTag.Value)
				if r.MatchString(resourceTag.Value) {
					tagMatches += 1
				}
			}
		}
	}

	if tagMatches == len(filterTags) {
		return true
	} else {
		return false
	}
}

func findExportedTags(resources []*TagsData) map[string][]string {
	m := make(map[string][]string)

	for _, r := range resources {
		for _, t := range r.Tags {
			value := t.Key
			if !stringInSlice(value, m[*r.Service]) {
				m[*r.Service] = append(m[*r.Service], value)
			}
		}
	}

	return m
}
