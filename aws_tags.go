package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

type tagsData struct {
	ID        *string
	Tags      []*tag
	Namespace *string
	Region    *string
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

func (iface tagsInterface) get(job *job, region string) (resources []*tagsData, err error) {
	if len(supportedServices[job.Namespace].ResourceFilters) > 0 {
		var inputparams = r.GetResourcesInput{
			ResourceTypeFilters: supportedServices[job.Namespace].ResourceFilters,
		}
		c := iface.client
		ctx := context.Background()
		pageNum := 0

		err = c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
			pageNum++
			resourceGroupTaggingAPICounter.Inc()

			if len(page.ResourceTagMappingList) == 0 {
				log.Debugf("Resource tag list is empty. Tags must be defined for %s to be discovered.", job.Namespace)
			}

			for _, resourceTagMapping := range page.ResourceTagMappingList {
				resource := tagsData{
					ID:        resourceTagMapping.ResourceARN,
					Namespace: &job.Namespace,
					Region:    &region,
				}

				for _, t := range resourceTagMapping.Tags {
					resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				} else {
					log.Debugf("Skipping resource %s because search tags do not match", *resource.ID)
				}
			}
			return pageNum < 100
		})
	}
	switch job.Namespace {
	case "AWS/AutoScaling":
		return iface.getTaggedAutoscalingGroups(job, region)
	case "AWS/EC2Spot":
		return iface.getTaggedEC2SpotInstances(job, region)
	case "AWS/TransitGateway":
		resources, err = iface.getTaggedTransitGatewayAttachments(job, region)
		if err != nil {
			return nil, err
		}
	case "AWS/ApiGateway":
		resources, err = iface.getTaggedApiGateway(resources)
		if err != nil {
			return nil, err
		}
	}
	return resources, err
}

// Once the resourcemappingapi supports ASGs then this workaround method can be deleted
// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/
func (iface tagsInterface) getTaggedAutoscalingGroups(job *job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.asgClient.DescribeAutoScalingGroupsPagesWithContext(ctx, &autoscaling.DescribeAutoScalingGroupsInput{},
		func(page *autoscaling.DescribeAutoScalingGroupsOutput, more bool) bool {
			pageNum++
			autoScalingAPICounter.Inc()

			for _, asg := range page.AutoScalingGroups {
				resource := tagsData{
					ID:        asg.AutoScalingGroupARN,
					Namespace: &job.Namespace,
					Region:    &region,
				}

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

func (iface tagsInterface) getTaggedApiGateway(inputResources []*tagsData) (resources []*tagsData, err error) {
	ctx := context.Background()
	apiGatewayAPICounter.Inc()
	var limit int64 = 500 // max number of results per page. default=25, max=500
	const maxPages = 10
	input := apigateway.GetRestApisInput{Limit: &limit}
	output := apigateway.GetRestApisOutput{}
	var pageNum int
	err = iface.apiGatewayClient.GetRestApisPagesWithContext(ctx, &input, func(page *apigateway.GetRestApisOutput, lastPage bool) bool {
		pageNum++
		output.Items = append(output.Items, page.Items...)
		return pageNum <= maxPages
	})
	for _, resource := range inputResources {
		for i, gw := range output.Items {
			if strings.Contains(*resource.ID, *gw.Id) {
				r := resource
				r.ID = aws.String(strings.ReplaceAll(*resource.ID, *gw.Id, *gw.Name))
				resources = append(resources, r)
				output.Items = append(output.Items[:i], output.Items[i+1:]...)
				break
			}
		}
	}
	return resources, err
}

func (iface tagsInterface) getTaggedTransitGatewayAttachments(job *job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.ec2Client.DescribeTransitGatewayAttachmentsPagesWithContext(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{},
		func(page *ec2.DescribeTransitGatewayAttachmentsOutput, more bool) bool {
			pageNum++
			ec2APICounter.Inc()

			for _, tgwa := range page.TransitGatewayAttachments {
				resource := tagsData{
					ID:        aws.String(fmt.Sprintf("transit-gateway-attachment/%s/%s", *tgwa.TransitGatewayId, *tgwa.TransitGatewayAttachmentId)),
					Namespace: &job.Namespace,
					Region:    &region,
				}

				for _, t := range tgwa.Tags {
					resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				}
			}
			return pageNum < 100
		},
	)
}

func (iface tagsInterface) getTaggedEC2SpotInstances(job *job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.ec2Client.DescribeSpotFleetRequestsPagesWithContext(ctx, &ec2.DescribeSpotFleetRequestsInput{},
		func(page *ec2.DescribeSpotFleetRequestsOutput, more bool) bool {
			pageNum++
			ec2APICounter.Inc()

			for _, ec2Spot := range page.SpotFleetRequestConfigs {
				resource := tagsData{
					ID:        ec2Spot.SpotFleetRequestId,
					Namespace: &job.Namespace,
					Region:    &region,
				}

				for _, t := range ec2Spot.Tags {
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
			if !stringInSlice(entry.Key, tagList[*d.Namespace]) {
				tagList[*d.Namespace] = append(tagList[*d.Namespace], entry.Key)
			}
		}
	}

	for _, d := range tagData {
		name := promString(strings.ToLower(*d.Namespace)) + "_info"
		promLabels := make(map[string]string)
		promLabels["name"] = *d.ID

		for _, entry := range tagList[*d.Namespace] {
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
