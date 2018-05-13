package main

import (
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

type cloudwatchInfo struct {
	Dimensions []*cloudwatch.Dimension
	Namespace  *string
}

type cloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
}

func createCloudwatchSession(region *string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return cloudwatch.New(sess, &aws.Config{Region: region})
}

func (iface cloudwatchInterface) getCloudwatchData(resource *awsInfoData, metric metric) *cloudwatchData {
	c := iface.client

	cloudwatchInfo := getCloudwatchInfo(resource.Service, resource.Id)

	period := int64(metric.Length)
	length := metric.Length
	endTime := time.Now()
	startTime := time.Now().Add(-time.Duration(length) * time.Minute)
	statistics := []*string{&metric.Statistics}

	resp, err := c.GetMetricStatistics(&cloudwatch.GetMetricStatisticsInput{
		Dimensions: cloudwatchInfo.Dimensions,
		Namespace:  cloudwatchInfo.Namespace,
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     &period,
		MetricName: &metric.Name,
		Statistics: statistics,
	})

	atomic.AddUint64(&CloudwatchApiRequests, 1)

	if err != nil {
		panic(err)
	}

	points := sortDatapoints(resp.Datapoints, metric.Statistics)

	var output cloudwatchData

	output.Tags = resource.Tags
	output.Service = resource.Service
	output.Metric = &metric.Name
	output.Id = resource.Id
	output.Statistics = &metric.Statistics

	if len(points) != 0 {
		point := float64(*points[0])
		output.Value = &point
	} else {
		if metric.NilToZero {
			point := float64(0)
			output.Value = &point
		}
	}

	return &output
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

func getCloudwatchInfo(service *string, resourceArn *string) (c cloudwatchInfo) {
	arnParsed, err := arn.Parse(*resourceArn)

	if err != nil {
		panic(err)
	}

	switch *service {
	case "ec2":
		c.buildInfo(arnParsed.Resource, "AWS/EC2", "InstanceId", "instance/")
	case "elb":
		c.buildInfo(arnParsed.Resource, "AWS/ELB", "LoadBalancerName", "loadbalancer/")
	case "rds":
		c.buildInfo(arnParsed.Resource, "AWS/RDS", "DBInstanceIdentifier", "db:")
	case "ec":
		c.buildInfo(arnParsed.Resource, "AWS/ElastiCache", "CacheClusterId", "cluster:")
	case "es":
		c.buildInfo(arnParsed.Resource, "AWS/ES", "DomainName", "domain/")
		c.addDimension("ClientId", arnParsed.AccountID)
	default:
		log.Fatal("Not implemented cloudwatch metric:" + *service)
	}
	return c
}

func (c *cloudwatchInfo) buildInfo(identifier string, namespace string, dimensionKey string, prefix string) *cloudwatchInfo {
	c.Namespace = aws.String(namespace)
	helper := strings.TrimPrefix(identifier, prefix)
	c.Dimensions = append(c.Dimensions, buildDimension(&dimensionKey, &helper))
	return c
}

func (c *cloudwatchInfo) addDimension(key string, value string) *cloudwatchInfo {
	c.Dimensions = append(c.Dimensions, buildDimension(&key, &value))
	return c
}

func buildDimension(key *string, value *string) *cloudwatch.Dimension {
	dimension := cloudwatch.Dimension{
		Name:  key,
		Value: value,
	}
	return &dimension
}
