package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
)

func createTagSession(region string) *r.ResourceGroupsTaggingAPI {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return r.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeResources(discovery discovery) (resources awsResources) {
	c := createTagSession(discovery.Region)

	var filter []*string

	switch discovery.Type {
	case "ec2":
		hotfix := aws.String("ec2:instance")
		filter = append(filter, hotfix)
	case "elb":
		hotfix := aws.String("elasticloadbalancing:loadbalancer")
		filter = append(filter, hotfix)
	case "rds":
		hotfix := aws.String("rds:db")
		filter = append(filter, hotfix)
	case "es":
		hotfix := aws.String("es:domain")
		filter = append(filter, hotfix)
	case "ec":
		hotfix := aws.String("elasticache:cluster")
		filter = append(filter, hotfix)
	default:
		fmt.Println("Not implemented resources:" + discovery.Type)
	}

	inputparams := r.GetResourcesInput{ResourceTypeFilters: filter}

	ctx := context.Background()
	pageNum := 0
	c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
		pageNum++
		for _, resourceTagMapping := range page.ResourceTagMappingList {
			resource := awsResource{}

			resource.Id = resourceTagMapping.ResourceARN

			resource.Service = &discovery.Type
			resource.Region = &discovery.Region

			for _, t := range resourceTagMapping.Tags {
				tag := tag{Key: *t.Key, Value: *t.Value}
				resource.Tags = append(resource.Tags, &tag)
			}

			if resource.filterThroughTags(discovery.SearchTags) {
				resources.Resources = append(resources.Resources, &resource)
			}
		}
		return pageNum < 100
	})

	return resources
}
