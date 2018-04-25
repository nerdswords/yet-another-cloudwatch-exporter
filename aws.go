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

type ElbWrapper struct {
	Elb  *elb.LoadBalancerDescription
	Tags []*elb.Tag
}

func FilterELBThroughTags(elbTags []*elb.Tag, searchTags []searchTag) bool {
	tagMatches := 0

	for _, elbTag := range elbTags {
		for _, searchTag := range searchTags {
			if searchTag.Key == *elbTag.Key {
				if searchTag.Value == *elbTag.Value {
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

func describeInstances(tags []searchTag) (instances []*ec2.Instance) {
	c := createEC2Session("eu-west-1")

	filters := []*ec2.Filter{}

	for _, tag := range tags {
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
			instances = append(instances, i)
		}
	}

	return instances
}

func getCloudwatchMetric(dimensionName string, dimensionValue *string, namespace string, metric metric) float64 {
	c := createCloudwatchSession()

	period := int64(metric.Length)
	length := metric.Length
	endTime := time.Now()
	startTime := time.Now().Add(-time.Duration(length) * time.Minute)
	statistics := []*string{&metric.Statistics}

	resp, err := c.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			&cloudwatch.Dimension{
				Name:  &dimensionName,
				Value: dimensionValue,
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

func describeLoadBalancers() (output []ElbWrapper) {
	c := createELBSession("eu-west-1")
	resp, err := c.DescribeLoadBalancers(nil)
	if err != nil {
		panic(err)
	}

	for _, elb := range resp.LoadBalancerDescriptions {
		add := ElbWrapper{}
		add.Elb = elb
		add.Tags = describeELBTags(add.Elb.LoadBalancerName)
		output = append(output, add)
	}

	return output
}

func describeELBTags(name *string) []*elb.Tag {
	c := createELBSession("eu-west-1")

	input := &elb.DescribeTagsInput{LoadBalancerNames: []*string{name}}

	tagDescription, err := c.DescribeTags(input)

	if err != nil {
		panic(err)
	}

	return tagDescription.TagDescriptions[0].Tags
}
