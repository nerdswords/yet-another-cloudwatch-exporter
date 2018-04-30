package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
)

func createELBSession(region string) *elb.ELB {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return elb.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeLoadBalancers(discovery discovery) (resources awsResources) {
	c := createELBSession(discovery.Region)
	resp, err := c.DescribeLoadBalancers(nil)
	if err != nil {
		panic(err)
	}

	for _, elb := range resp.LoadBalancerDescriptions {
		resource := awsResource{}
		resource.Id = elb.LoadBalancerName
		resource.Service = aws.String("elb")
		resource.Tags = describeELBTags(c, resource.Id)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources.Resources = append(resources.Resources, &resource)
		}
	}

	resources.CloudwatchInfo = getElbCloudwatchInfo()

	return resources
}

func describeELBTags(c *elb.ELB, name *string) (tags []*tag) {
	input := &elb.DescribeTagsInput{LoadBalancerNames: []*string{name}}

	tagDescription, err := c.DescribeTags(input)

	if err != nil {
		panic(err)
	}

	for _, elbTag := range tagDescription.TagDescriptions[0].Tags {
		tag := tag{Key: *elbTag.Key, Value: *elbTag.Value}
		tags = append(tags, &tag)
	}

	return tags
}

func getElbCloudwatchInfo() *cloudwatchInfo {
	output := cloudwatchInfo{}
	output.DimensionName = aws.String("LoadBalancerName")
	output.Namespace = aws.String("AWS/ELB")
	output.CustomDimension = []*tag{}
	return &output
}
