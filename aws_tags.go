package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

type tagsData struct {
	ID      *string
	Matcher *string
	Tags    []*tag
	Service *string
	Region  *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	client           resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asgClient        autoscalingiface.AutoScalingAPI
	apiGatewayClient apigatewayiface.APIGatewayAPI
	ec2Client        ec2iface.EC2API
}

func createSession(roleArn string, config *aws.Config) *session.Session {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session due to %v", err)
	}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}
	return sess
}

func createTagSession(region *string, roleArn string) *r.ResourceGroupsTaggingAPI {
	maxResourceGroupTaggingRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxResourceGroupTaggingRetries}
	if *fips {
		// ToDo: Resource Groups Tagging API does not have FIPS compliant endpoints
		// https://docs.aws.amazon.com/general/latest/gr/arg.html
		// endpoint := fmt.Sprintf("https://tagging-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}
	return r.New(createSession(roleArn, config), config)
}

func createASGSession(region *string, roleArn string) autoscalingiface.AutoScalingAPI {
	maxAutoScalingAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAutoScalingAPIRetries}
	if *fips {
		// ToDo: Autoscaling does not have a FIPS endpoint
		// https://docs.aws.amazon.com/general/latest/gr/autoscaling_region.html
		// endpoint := fmt.Sprintf("https://autoscaling-plans-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}
	return autoscaling.New(createSession(roleArn, config), config)
}

func createEC2Session(region *string, roleArn string) ec2iface.EC2API {
	maxEC2APIRetries := 10
	config := &aws.Config{Region: region, MaxRetries: &maxEC2APIRetries}
	if *fips {
		// https://docs.aws.amazon.com/general/latest/gr/ec2-service.html
		endpoint := fmt.Sprintf("https://ec2-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}
	return ec2.New(createSession(roleArn, config), config)
}

func createAPIGatewaySession(region *string, roleArn string) apigatewayiface.APIGatewayAPI {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	maxApiGatewaygAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxApiGatewaygAPIRetries}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}
	if *fips {
		// https://docs.aws.amazon.com/general/latest/gr/apigateway.html
		endpoint := fmt.Sprintf("https://apigateway-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	return apigateway.New(sess, config)
}

func (iface tagsInterface) get(job job, region string) (resources []*tagsData, err error) {
	c := iface.client

	var filter []*string

	switch job.Type {
	case "alb":
		filter = append(filter, aws.String("elasticloadbalancing:loadbalancer/app"))
		filter = append(filter, aws.String("elasticloadbalancing:targetgroup"))
	case "apigateway":
		filter = append(filter, aws.String("apigateway"))
	case "appsync":
		filter = append(filter, aws.String("appsync"))
	case "cf":
		filter = append(filter, aws.String("cloudfront"))
	case "asg":
		return iface.getTaggedAutoscalingGroups(job, region)
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
	case "firehose":
		filter = append(filter, aws.String("firehose"))
	case "fsx":
		filter = append(filter, aws.String("fsx:file-system"))
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
	case "redshift":
		filter = append(filter, aws.String("redshift:cluster"))
	case "r53r":
		filter = append(filter, aws.String("route53resolver"))
	case "s3":
		filter = append(filter, aws.String("s3"))
	case "sfn":
		filter = append(filter, aws.String("states"))
	case "sns":
		filter = append(filter, aws.String("sns"))
	case "sqs":
		filter = append(filter, aws.String("sqs"))
	case "tgw":
		filter = append(filter, aws.String("ec2:transit-gateway"))
	case "tgwa":
		return iface.getTaggedTransitGatewayAttachments(job, region)
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
	resourcePages := c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
		pageNum++
		resourceGroupTaggingAPICounter.Inc()
		for _, resourceTagMapping := range page.ResourceTagMappingList {
			resource := tagsData{}

			resource.ID = resourceTagMapping.ResourceARN

			resource.Service = &job.Type
			resource.Region = &region

			for _, t := range resourceTagMapping.Tags {
				resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
			}

			if resource.filterThroughTags(job.SearchTags) {
				resources = append(resources, &resource)
			}
		}
		return pageNum < 100
	})

	switch job.Type {
	case "apigateway":
		// Get all the api gateways from aws
		apiGateways, _ := iface.getTaggedApiGateway()
		var filteredResources []*tagsData
		for _, r := range resources {
			// For each tagged resource, find the associated restApi
			// And swap out the ID with the name
			if strings.Contains(*r.ID, "/restapis") {
				restApiId := strings.Split(*r.ID, "/")[2]
				for _, apiGateway := range apiGateways.Items {
					if *apiGateway.Id == restApiId {
						r.Matcher = apiGateway.Name
					}
				}
				filteredResources = append(filteredResources, r)
			}
		}
		resources = filteredResources
	}

	return resources, resourcePages
}

// Once the resourcemappingapi supports ASGs then this workaround method can be deleted
// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/
func (iface tagsInterface) getTaggedAutoscalingGroups(job job, region string) (resources []*tagsData, err error) {
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
				resource.ID = aws.String(fmt.Sprintf("arn:%s:autoscaling:%s:%s:%s", parts[1], parts[3], parts[4], parts[7]))

				resource.Service = &job.Type
				resource.Region = &region

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

// Get all ApiGateways REST
func (iface tagsInterface) getTaggedApiGateway() (*apigateway.GetRestApisOutput, error) {
	ctx := context.Background()
	apiGatewayAPICounter.Inc()
	return iface.apiGatewayClient.GetRestApisWithContext(ctx, &apigateway.GetRestApisInput{})
}

func (iface tagsInterface) getTaggedTransitGatewayAttachments(job job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.ec2Client.DescribeTransitGatewayAttachmentsPagesWithContext(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{},
		func(page *ec2.DescribeTransitGatewayAttachmentsOutput, more bool) bool {
			pageNum++
			ec2APICounter.Inc()

			for _, tgwa := range page.TransitGatewayAttachments {
				resource := tagsData{}

				resource.ID = aws.String(fmt.Sprintf("%s/%s", *tgwa.TransitGatewayId, *tgwa.TransitGatewayAttachmentId))

				resource.Service = &job.Type
				resource.Region = &region

				for _, t := range tgwa.Tags {
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
