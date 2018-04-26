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

func describeLoadBalancers(discovery discovery) (resources []*resourceWrapper) {
	c := createELBSession("eu-west-1")
	resp, err := c.DescribeLoadBalancers(nil)
	if err != nil {
		panic(err)
	}

	for _, elb := range resp.LoadBalancerDescriptions {
		resource := resourceWrapper{}
		resource.Id = elb.LoadBalancerName
		resource.Service = aws.String("elb")
		resource.Tags = describeELBTags(resource.Id)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources = append(resources, &resource)
		}
	}

	return resources
}

func describeELBTags(name *string) (tags []*searchTag) {
	c := createELBSession("eu-west-1")

	input := &elb.DescribeTagsInput{LoadBalancerNames: []*string{name}}

	tagDescription, err := c.DescribeTags(input)

	if err != nil {
		panic(err)
	}

	for _, elbTag := range tagDescription.TagDescriptions[0].Tags {
		tag := searchTag{Key: *elbTag.Key, Value: *elbTag.Value}
		tags = append(tags, &tag)
	}

	return tags
}
