package main

import (
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

func describeInstances(discovery discovery) (resources awsResources) {
	c := createEC2Session(discovery.Region)

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
			resource := awsResource{Id: i.InstanceId}
			resource.Service = aws.String("ec2")
			resource.Tags = getInstanceTags(i.Tags)
			resources.Resources = append(resources.Resources, &resource)
		}
	}

	resources.CloudwatchInfo = getEc2CloudwatchInfo()

	return resources
}

func getInstanceTags(awsTags []*ec2.Tag) (output []*tag) {
	for _, awsTag := range awsTags {
		tag := tag{Key: *awsTag.Key, Value: *awsTag.Value}
		output = append(output, &tag)
	}
	return output
}

func getEc2CloudwatchInfo() *cloudwatchInfo {
	output := cloudwatchInfo{}
	output.DimensionName = aws.String("InstanceId")
	output.Namespace = aws.String("AWS/EC2")
	output.CustomDimension = []*tag{}
	return &output
}
