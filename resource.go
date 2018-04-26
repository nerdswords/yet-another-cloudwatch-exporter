package main

import (
	"fmt"
)

type resourceWrapper struct {
	Id      *string
	Tags    []*searchTag
	Service *string
}

func createPrometheusExportedTags(jobs []job) map[string][]string {
	exportedTags := map[string]map[string]bool{
		"rds": map[string]bool{},
		"ec2": map[string]bool{},
		"elb": map[string]bool{},
	}

	output := map[string][]string{
		"rds": []string{},
		"ec2": []string{},
		"elb": []string{},
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

func describeResources(discovery discovery) (resources []*resourceWrapper) {
	switch discovery.Type {
	case "ec2":
		resources = describeInstances(discovery)
	case "elb":
		resources = describeLoadBalancers(discovery)
	case "rds":
		resources = describeDatabases(discovery)
	default:
		fmt.Println("Not implemented resources:" + discovery.Type)
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
