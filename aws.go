package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"sort"
	"time"
)

type resourceWrapper struct {
	Id      *string
	Tags    []*searchTag
	Service *string
}

func createEC2Session(region string) *ec2.EC2 {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return ec2.New(sess, &aws.Config{Region: aws.String(region)})
}

func createCloudwatchSession() *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return cloudwatch.New(sess)
}

func createELBSession(region string) *elb.ELB {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return elb.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeResources(discovery discovery) (resources []*resourceWrapper) {
	if discovery.Type == "ec2" {
		resources = describeInstances(discovery)
	} else if discovery.Type == "elb" {
		resources = describeLoadBalancers(discovery)
	} else {
		fmt.Println("Not implemented yet :(")
	}
	return resources
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

func getCloudwatchMetric(resource *resourceWrapper, metric metric) float64 {
	c := createCloudwatchSession()

	var dimensionName string
	var namespace string

	if *resource.Service == "ec2" {
		dimensionName = "InstanceId"
		namespace = "AWS/EC2"
	} else if *resource.Service == "elb" {
		dimensionName = "LoadBalancerName"
		namespace = "AWS/ELB"
	}

	period := int64(metric.Length)
	length := metric.Length
	endTime := time.Now()
	startTime := time.Now().Add(-time.Duration(length) * time.Minute)
	statistics := []*string{&metric.Statistics}

	resp, err := c.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			&cloudwatch.Dimension{
				Name:  &dimensionName,
				Value: resource.Id,
			},
		},
		Namespace:  &namespace,
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     &period,
		MetricName: aws.String(metric.Name),
		Statistics: statistics,
	})

	if err != nil {
		panic(err)
	}

	points := sortDatapoints(resp.Datapoints, metric.Statistics)

	if len(points) == 0 {
		fmt.Println("Did not found any data points..")
		return float64(-1)
	} else {
		return float64(*points[0])
	}

}

func sortDatapoints(datapoints []*cloudwatch.Datapoint, statistic string) (points []*float64) {
	for _, point := range datapoints {
		if statistic == "Sum" {
			points = append(points, point.Sum)
		} else if statistic == "Average" {
			points = append(points, point.Average)
		} else if statistic == "Maximum" {
			points = append(points, point.Maximum)
		} else if statistic == "Minimum" {
			points = append(points, point.Minimum)
		}
	}

	sort.Slice(points, func(i, j int) bool { return *points[i] > *points[j] })

	return points
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
		if filterElbThroughTags(resource.Tags, discovery.SearchTags) {
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

func filterElbThroughTags(elbTags []*searchTag, searchTags []searchTag) bool {
	tagMatches := 0

	for _, elbTag := range elbTags {
		for _, searchTag := range searchTags {
			if searchTag.Key == elbTag.Key {
				if searchTag.Value == elbTag.Value {
					tagMatches += 1
				}
			}
		}
	}

	if tagMatches == len(searchTags) {
		return true
	} else {
		return false
	}
}
