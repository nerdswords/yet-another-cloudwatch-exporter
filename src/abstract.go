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
	Statistics *string
	Value      *float64
	Empty      bool
	Tags       []*tag
}

func scrapeData(conf conf) ([]*awsInfoData, []*cloudwatchData) {
	mux := &sync.Mutex{}

	cloudwatchData := make([]*cloudwatchData, 0)
	awsInfoData := make([]*awsInfoData, 0)

	var wg sync.WaitGroup
	for i, _ := range conf.Jobs {
		wg.Add(1)
		job := conf.Jobs[i]
		go func() {
			resources := describeResources(job.Discovery)

			for _, resource := range resources {
				mux.Lock()
				awsInfoData = append(awsInfoData, resource)
				mux.Unlock()
				client := cloudwatchInterface{
					client: createCloudwatchSession(resource.Region),
				}

				for _, metric := range job.Metrics {
					data := client.getCloudwatchData(resource, metric)
					cloudwatchData = append(cloudwatchData, data)
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
