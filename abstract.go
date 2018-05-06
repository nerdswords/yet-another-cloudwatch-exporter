package main

import (
	_ "fmt"
	"regexp"
)

type awsResources struct {
	Resources []*awsResource
}

type awsResource struct {
	Id         *string
	Tags       []*tag
	Service    *string
	Attributes map[string]*string
}

func createPrometheusExportedTags(jobs []job) map[string][]string {
	exportedTags := map[string]map[string]bool{
		"ec":  map[string]bool{},
		"ec2": map[string]bool{},
		"elb": map[string]bool{},
		"es":  map[string]bool{},
		"rds": map[string]bool{},
	}

	output := map[string][]string{
		"ec":  []string{},
		"ec2": []string{},
		"elb": []string{},
		"es":  []string{},
		"rds": []string{},
	}

	for _, job := range jobs {
		for _, tag := range job.Discovery.ExportedTags {
			exportedTags[job.Discovery.Type][tag] = true
		}
	}

	for k, v := range exportedTags {
		for kk, _ := range v {
			output[k] = append(output[k], kk)
		}
	}

	return output
}

func (r awsResource) filterThroughTags(filterTags []tag) bool {
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
