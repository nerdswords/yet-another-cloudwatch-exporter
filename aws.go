package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"sort"
	"time"
)

type resourceWrapper struct {
	Id      *string
	Tags    []*searchTag
	Service *string
}

func createCloudwatchSession() *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return cloudwatch.New(sess)
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
