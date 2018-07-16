package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"log"
)

type TagsData struct {
	Id      *string
	Tags    []*tag
	Service *string
	Region  *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	client resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}

func createTagSession(region *string) *r.ResourceGroupsTaggingAPI {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return r.New(sess, &aws.Config{Region: region})
}

func (iface tagsInterface) get(discovery discovery) (resources []*TagsData) {
	c := iface.client

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
	case "s3":
		hotfix := aws.String("s3")
		filter = append(filter, hotfix)
	default:
		log.Fatal("Not implemented resources:" + discovery.Type)
	}

	inputparams := r.GetResourcesInput{ResourceTypeFilters: filter}

	ctx := context.Background()
	pageNum := 0
	c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
		pageNum++
		for _, resourceTagMapping := range page.ResourceTagMappingList {
			resource := TagsData{}

			resource.Id = resourceTagMapping.ResourceARN

			resource.Service = &discovery.Type
			resource.Region = &discovery.Region

			for _, t := range resourceTagMapping.Tags {
				tag := tag{Key: *t.Key, Value: *t.Value}
				resource.Tags = append(resource.Tags, &tag)
			}

			if resource.filterThroughTags(discovery.SearchTags) {
				resources = append(resources, &resource)
			}
		}
		return pageNum < 100
	})

	return resources
}

func migrateTagsToPrometheus(tagData []*TagsData) []*PrometheusData {
	output := make([]*PrometheusData, 0)

	var tagList []string

	for _, d := range tagData {
		for _, entry := range d.Tags {
			if !stringInSlice(entry.Key, tagList) {
				tagList = append(tagList, entry.Key)
			}
		}
	}

	for _, d := range tagData {
		name := "aws_" + *d.Service + "_info"
		promLabels := make(map[string]string)
		promLabels["name"] = *d.Id

		for _, entry := range tagList {
			labelKey := PromString(entry)
			promLabels[labelKey] = ""

			for _, rTag := range d.Tags {
				if entry == rTag.Key {
					promLabels[labelKey] = rTag.Value
				}
			}
		}

		var i int = 0
		var f float64 = float64(i)

		p := PrometheusData{
			name:   &name,
			labels: promLabels,
			value:  &f,
		}

		output = append(output, &p)
	}

	return output
}
