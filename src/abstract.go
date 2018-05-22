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

type cloudwatchData struct {
	Id         *string
	Metric     *string
	Service    *string
	Region     *string
	Statistics *string
	Value      *float64
	Empty      bool
	Tags       []*tag
}

func scrapeAwsData(jobs []job) ([]*awsInfoData, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cloudwatchData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*awsInfoData, 0)

	var wg sync.WaitGroup
	for i, _ := range jobs {
		wg.Add(1)
		job := jobs[i]
		go func() {
			clientTag := tagsInterface{
				client: createTagSession(job.Discovery.Region),
			}

			resources := clientTag.get(job.Discovery)

			for _, resource := range resources {
				mux.Lock()
				awsInfoData = append(awsInfoData, resource)
				mux.Unlock()
				clientCloudwatch := cloudwatchInterface{
					client: createCloudwatchSession(resource.Region),
				}

				for _, metric := range job.Metrics {
					data := clientCloudwatch.get(resource, metric)
					mux.Lock()
					cloudwatchData = append(cloudwatchData, data)
					mux.Unlock()
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return awsInfoData, cloudwatchData
}

func (r awsInfoData) filterThroughTags(filterTags []tag) bool {
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

func findExportedTags(resources []*awsInfoData) map[string][]string {
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
