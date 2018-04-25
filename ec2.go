package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func createEC2Session(region string) *ec2.EC2 {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return ec2.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeInstances(discovery discovery) (resources []*resourceWrapper) {
	c := createEC2Session("eu-west-1")

	filters := []*ec2.Filter{}

	for _, tag := range discovery.SearchTags {
		filter := ec2.Filter{
			Name: aws.String("tag:" + tag.Key),
			Values: []*string{
				aws.String(tag.Value),
			},
		}

		filters = append(filters, &filter)
	}

	params := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	resp, err := c.DescribeInstances(params)
	if err != nil {
		panic(err)
	}

	for idx, _ := range resp.Reservations {
		for _, i := range resp.Reservations[idx].Instances {
			resource := resourceWrapper{Id: i.InstanceId}
			resource.Service = aws.String("ec2")
			fmt.Println("Add tags here.. to struct")
			resources = append(resources, &resource)
		}
	}

	return resources
}
