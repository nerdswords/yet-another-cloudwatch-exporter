package main

import (
	_ "fmt"
	"regexp"
)

type awsResources struct {
	Resources []*awsResource
}

type awsResource struct {
	Id      *string
	Tags    []*tag
	Service *string
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
