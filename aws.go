package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sort"
	"time"
)

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

func describeInstances(tags []Tag) (instances []*ec2.Instance) {
	c := createEC2Session("eu-west-1")

	filters := []*ec2.Filter{}

	for _, tag := range tags {
		filter := ec2.Filter{
			Name: aws.String("tag:" + tag.key),
			Values: []*string{
				aws.String(tag.value),
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

func getCloudwatchMetricEC2(instance *ec2.Instance, metric Metric) float64 {
	c := createCloudwatchSession()

	period := int64(60)
	endTime := time.Now()
	startTime := time.Now().Add(-24 * time.Hour)
	statistics := []*string{&metric.statistics}

	resp, err := c.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Dimensions: []*cloudwatch.Dimension{
			&cloudwatch.Dimension{
				Name:  aws.String("InstanceId"),
				Value: instance.InstanceId,
			},
		},
		Namespace:  aws.String("AWS/EC2"),
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     &period,
		MetricName: aws.String(metric.name),
		Statistics: statistics,
	})

	if err != nil {
		panic(err)
	}

	points := sortDatapoints(resp.Datapoints)

	return float64(*points[0])
}

func sortDatapoints(datapoints []*cloudwatch.Datapoint) (points []*float64) {
	for _, point := range datapoints {
		points = append(points, point.Average)
	}

	sort.Slice(points, func(i, j int) bool { return *points[i] < *points[j] })

	return points
}
