package main

import (
	"context"
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"log"
)

type tagsData struct {
	ID      *string
	Tags    []*tag
	Service *string
	Region  *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	client resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}

func createTagSession(region *string, roleArn string) *r.ResourceGroupsTaggingAPI {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	config := &aws.Config{Region: region}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	return r.New(sess, config)
}

func (iface tagsInterface) get(job job) (resources []*tagsData, err error) {
	c := iface.client

	var filter []*string

	switch job.Type {
	case "ec2":
		hotfix := aws.String("ec2:instance")
		filter = append(filter, hotfix)
	case "elb":
		hotfix := aws.String("elasticloadbalancing:loadbalancer")
		filter = append(filter, hotfix)
	case "alb":
		alb := aws.String("elasticloadbalancing:loadbalancer")
		tg := aws.String("elasticloadbalancing:targetgroup")
		filter = append(filter, alb)
		filter = append(filter, tg)
	case "vpn":
		connection := aws.String("ec2:vpn-connection")
		filter = append(filter, connection)
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
	case "efs":
		hotfix := aws.String("elasticfilesystem:file-system")
		filter = append(filter, hotfix)
	case "ebs":
		hotfix := aws.String("ec2:volume")
		filter = append(filter, hotfix)
	case "lambda":
		hotfix := aws.String("lambda:function")
		filter = append(filter, hotfix)
	default:
		log.Fatal("Not implemented resources:" + job.Type)
	}

	inputparams := r.GetResourcesInput{ResourceTypeFilters: filter}

	ctx := context.Background()
	pageNum := 0
	return resources, c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
		pageNum++
		for _, resourceTagMapping := range page.ResourceTagMappingList {
			resource := tagsData{}

			resource.ID = resourceTagMapping.ResourceARN

			resource.Service = &job.Type
			resource.Region = &job.Region

			for _, t := range resourceTagMapping.Tags {
				resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
			}

			if resource.filterThroughTags(job.SearchTags) {
				resources = append(resources, &resource)
			}
		}
		return pageNum < 100
	})
}

func migrateTagsToPrometheus(tagData []*tagsData) []*prometheusData {
	output := make([]*prometheusData, 0)

	tagList := make(map[string][]string)

	for _, d := range tagData {
		for _, entry := range d.Tags {
			if !stringInSlice(entry.Key, tagList[*d.Service]) {
				tagList[*d.Service] = append(tagList[*d.Service], entry.Key)
			}
		}
	}

	for _, d := range tagData {
		name := "aws_" + *d.Service + "_info"
		promLabels := make(map[string]string)
		promLabels["name"] = *d.ID

		for _, entry := range tagList[*d.Service] {
			labelKey := "tag_" + promStringTag(entry)
			promLabels[labelKey] = ""

			for _, rTag := range d.Tags {
				if entry == rTag.Key {
					promLabels[labelKey] = rTag.Value
				}
			}
		}

		var i int
		f := float64(i)

		p := prometheusData{
			name:   &name,
			labels: promLabels,
			value:  &f,
		}

		output = append(output, &p)
	}

	return output
}
