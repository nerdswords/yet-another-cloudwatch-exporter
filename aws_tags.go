package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
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
		log.Fatal(err)
	}
	maxResourceGroupTaggingRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxResourceGroupTaggingRetries}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	return r.New(sess, config)
}

func createASGSession(region *string, roleArn string) autoscalingiface.AutoScalingAPI {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	maxAutoScalingAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAutoScalingAPIRetries}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	return autoscaling.New(sess, config)
}

func (iface tagsInterface) get(job job) (resources []*tagsData, err error) {
	c := iface.client

	var filter []*string

	switch job.Type {
	case "alb":
		filter = append(filter, aws.String("elasticloadbalancing:loadbalancer/app"))
		filter = append(filter, aws.String("elasticloadbalancing:targetgroup"))
	case "cf":
		filter = append(filter, aws.String("cloudfront"))
	case "asg":
		return iface.getTaggedAutoscalingGroups(job)
	case "dynamodb":
		filter = append(filter, aws.String("dynamodb:table"))
	case "ebs":
		filter = append(filter, aws.String("ec2:volume"))
	case "ec":
		filter = append(filter, aws.String("elasticache:cluster"))
	case "ec2":
		filter = append(filter, aws.String("ec2:instance"))
	case "ecs-svc", "ecs-containerinsights":
		filter = append(filter, aws.String("ecs:cluster"))
		filter = append(filter, aws.String("ecs:service"))
	case "efs":
		filter = append(filter, aws.String("elasticfilesystem:file-system"))
	case "elb":
		filter = append(filter, aws.String("elasticloadbalancing:loadbalancer"))
	case "emr":
		filter = append(filter, aws.String("elasticmapreduce:cluster"))
	case "es":
		filter = append(filter, aws.String("es:domain"))
	case "kinesis":
		filter = append(filter, aws.String("kinesis:stream"))
	case "lambda":
		filter = append(filter, aws.String("lambda:function"))
	case "ngw":
		filter = append(filter, aws.String("ec2:natgateway"))
	case "nlb":
		filter = append(filter, aws.String("elasticloadbalancing:loadbalancer/net"))
	case "rds":
		filter = append(filter, aws.String("rds:db"))
	case "r53r":
		filter = append(filter, aws.String("route53resolver"))
	case "s3":
		filter = append(filter, aws.String("s3"))
	case "sqs":
		filter = append(filter, aws.String("sqs"))
	case "tgw":
		filter = append(filter, aws.String("ec2:transit-gateway"))
	case "vpn":
		filter = append(filter, aws.String("ec2:vpn-connection"))
	case "kafka":
		filter = append(filter, aws.String("kafka:cluster"))
	default:
		log.Fatal("Not implemented resources:" + job.Type)
	}

	inputparams := r.GetResourcesInput{ResourceTypeFilters: filter}

	ctx := context.Background()
	pageNum := 0
	return resources, c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
		pageNum++
		resourceGroupTaggingAPICounter.Inc()
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
			autoScalingAPICounter.Inc()

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
		name := "aws_" + promString(*d.Service) + "_info"
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
