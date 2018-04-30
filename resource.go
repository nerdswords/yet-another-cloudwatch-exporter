package main

import (
	"fmt"
	"regexp"
)

type awsResources struct {
	Resources      []*awsResource
	CloudwatchInfo *cloudwatchInfo
}

type awsResource struct {
	Id         *string
	Tags       []*tag
	Service    *string
	Attributes map[string]*string
}

type cloudwatchInfo struct {
	DimensionName   *string
	Namespace       *string
	CustomDimension []*tag
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

func createPrometheusExportedAttributes(jobs []job) map[string][]string {
	exportedAttributes := map[string]map[string]bool{
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
		for _, attribute := range job.Discovery.ExportedAttributes {
			exportedAttributes[job.Discovery.Type][attribute] = true
		}
	}

	for k, v := range exportedAttributes {
		for kk, _ := range v {
			output[k] = append(output[k], kk)
		}
	}

	return output
}

func describeResources(discovery discovery) (resources awsResources) {
	switch discovery.Type {
	case "ec2":
		resources = describeInstances(discovery)
	case "elb":
		resources = describeLoadBalancers(discovery)
	case "rds":
		resources = describeDatabases(discovery)
	case "es":
		resources = describeElasticsearchDomains(discovery)
	case "ec":
		resources = describeElasticacheDomains(discovery)
	default:
		fmt.Println("Not implemented resources:" + discovery.Type)
	}

	return resources
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
