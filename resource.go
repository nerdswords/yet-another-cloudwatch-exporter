package main

import (
	"fmt"
)

type resourceWrapper struct {
	Id      *string
	Tags    []*searchTag
	Service *string
}

func describeResources(discovery discovery) (resources []*resourceWrapper) {
	if discovery.Type == "ec2" {
		resources = describeInstances(discovery)
	} else if discovery.Type == "elb" {
		resources = describeLoadBalancers(discovery)
	} else if discovery.Type == "rds" {
		resources = describeDatabases(discovery)
	} else {
		fmt.Println("Not implemented yet :(")
	}
	return resources
}

func (r resourceWrapper) filterThroughTags(filterTags []searchTag) bool {
	tagMatches := 0

	for _, resourceTag := range r.Tags {
		for _, filterTag := range filterTags {
			if resourceTag.Key == filterTag.Key {
				if resourceTag.Value == filterTag.Value {
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
