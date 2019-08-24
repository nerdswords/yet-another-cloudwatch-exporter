package main

import (
	"context"
	"fmt"
	_ "fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
)

type tagsData struct {
	ID      *string
	Tags    []*tag
	Service *string
	Region  *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	client    resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asgClient autoscalingiface.AutoScalingAPI
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

func createASGSession(region *string, roleArn string) autoscalingiface.AutoScalingAPI {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	config := &aws.Config{Region: region}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	return autoscaling.New(sess, config)
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
	case "kinesis":
		hotfix := aws.String("kinesis:stream")
		filter = append(filter, hotfix)
	case "dynamodb":
		hotfix := aws.String("dynamodb:table")
		filter = append(filter, hotfix)
	case "emr":
		hotfix := aws.String("elasticmapreduce:cluster")
		filter = append(filter, hotfix)
	case "asg":
		return iface.getTaggedAutoscalingGroups(job)
	case "sqs":
		hotfix := aws.String("sqs")
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

// Once the resourcemappingapi supports ASGs then this workaround method can be deleted
func (iface tagsInterface) getTaggedAutoscalingGroups(job job) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.asgClient.DescribeAutoScalingGroupsPagesWithContext(ctx, &autoscaling.DescribeAutoScalingGroupsInput{},
		func(page *autoscaling.DescribeAutoScalingGroupsOutput, more bool) bool {
			pageNum++

			for _, asg := range page.AutoScalingGroups {
				resource := tagsData{}

				// Transform the ASG ARN into something which looks more like an ARN from the ResourceGroupTaggingAPI
				parts := strings.Split(*asg.AutoScalingGroupARN, ":")
				resource.ID = aws.String(fmt.Sprintf("arn:aws:autoscaling:%s:%s:%s", parts[3], parts[4], parts[7]))

				resource.Service = &job.Type
				resource.Region = &job.Region

				for _, t := range asg.Tags {
					resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				}
			}
			return pageNum < 100
		})
}

func migrateTagsToPrometheus(tagData []*tagsData) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

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

		p := PrometheusMetric{
			name:   &name,
			labels: promLabels,
			value:  &f,
		}

		output = append(output, &p)
	}

	return output
}
