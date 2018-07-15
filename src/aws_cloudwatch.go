package main

import (
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

type cloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
}

type CloudwatchData struct {
	Id         *string
	Metric     *string
	Service    *string
	Statistics []string
	Value      []*cloudwatch.Datapoint
}

func createCloudwatchSession(region *string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return cloudwatch.New(sess, &aws.Config{Region: region})
}

func prepareCloudwatchRequest(service *string, arn *string, metric metric) *cloudwatch.GetMetricStatisticsInput {
	period := int64(metric.Period)
	length := metric.Length
	endTime := time.Now()
	startTime := time.Now().Add(-time.Duration(length) * time.Minute)

	var statistics []*string
	for _, statistic := range metric.Statistics {
		statistics = append(statistics, &statistic)
	}

	return &cloudwatch.GetMetricStatisticsInput{
		Dimensions: getDimensions(service, arn),
		Namespace:  getNamespace(service),
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     &period,
		MetricName: &metric.Name,
		Statistics: statistics,
	}
}

func (iface cloudwatchInterface) get(service *string, id *string, metric metric) []*cloudwatch.Datapoint {
	c := iface.client

	filter := prepareCloudwatchRequest(service, id, metric)
	resp, err := c.GetMetricStatistics(filter)
	atomic.AddUint64(&CloudwatchApiRequests, 1)

	if err != nil {
		panic(err)
	}

	return resp.Datapoints
}

func getNamespace(service *string) *string {
	var ns string
	switch *service {
	case "ec2":
		ns = "AWS/EC2"
	case "elb":
		ns = "AWS/ELB"
	case "rds":
		ns = "AWS/RDS"
	case "ec":
		ns = "AWS/ElastiCache"
	case "es":
		ns = "AWS/ES"
	case "s3":
		ns = "AWS/S3"
	default:
		log.Fatal("Not implemented namespace for cloudwatch metric:" + *service)
	}
	return &ns
}

func getDimensions(service *string, resourceArn *string) (dimensions []*cloudwatch.Dimension) {
	arnParsed, err := arn.Parse(*resourceArn)

	if err != nil {
		panic(err)
	}

	switch *service {
	case "ec2":
		dimensions := buildBaseDimension(arnParsed.Resource, "InstanceId", "instance/")
	case "elb":
		dimensions := buildBaseDimension(arnParsed.Resource, "LoadBalancerName", "loadbalancer/")
	case "rds":
		dimensions := buildBaseDimension(arnParsed.Resource, "DBInstanceIdentifier", "db:")
	case "ec":
		dimensions := buildBaseDimension(arnParsed.Resource, "CacheClusterId", "cluster:")
	case "es":
		dimensions := buildBaseDimension(arnParsed.Resource, "DomainName", "domain/")
		dimensions = append(dimensions, buildDimension("ClientId", arnParsed.AccountID))
	case "s3":
		dimensions := buildBaseDimension(arnParsed.Resource, "BucketName", "")
		dimensions = append(dimensions, buildDimension("StorageType", "AllStorageTypes"))
	default:
		log.Fatal("Not implemented cloudwatch metric:" + *service)
	}
	return dimensions
}

func buildBaseDimension(identifier string, dimensionKey string, prefix string) (dimensions []*cloudwatch.Dimension) {
	helper := strings.TrimPrefix(identifier, prefix)
	dimensions = append(dimensions, buildDimension(dimensionKey, helper))

	return dimensions
}

func buildDimension(key string, value string) *cloudwatch.Dimension {
	dimension := cloudwatch.Dimension{
		Name:  &key,
		Value: &value,
	}
	return &dimension
}
