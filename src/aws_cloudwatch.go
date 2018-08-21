package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"log"
	"strings"
	"time"
)

type cloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
}

type cloudwatchData struct {
	ID         *string
	Metric     *string
	Service    *string
	Statistics []string
	Points     []*cloudwatch.Datapoint
	NilToZero  *bool
}

func createCloudwatchSession(region *string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return cloudwatch.New(sess, &aws.Config{Region: region})
}

func prepareCloudwatchRequest(dimensions []*cloudwatch.Dimension, namespace *string, metric metric) (output *cloudwatch.GetMetricStatisticsInput) {
	period := int64(metric.Period)
	length := metric.Length
	endTime := time.Now()
	startTime := time.Now().Add(-time.Duration(length) * time.Second)

	var statistics []*string
	for _, statistic := range metric.Statistics {
		statistics = append(statistics, &statistic)
	}

	output = &cloudwatch.GetMetricStatisticsInput{
		Dimensions: dimensions,
		Namespace:  namespace,
		StartTime:  &startTime,
		EndTime:    &endTime,
		Period:     &period,
		MetricName: &metric.Name,
		Statistics: statistics,
	}
	if *debug {
		log.Println(*output)
	}
	return output
}

func (iface cloudwatchInterface) get(filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint {
	c := iface.client

	if *debug {
		log.Println(filter)
	}

	resp, err := c.GetMetricStatistics(filter)

	if *debug {
		log.Println(resp)
	}

	cloudwatchAPICounter.Inc()

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
	case "efs":
		ns = "AWS/EFS"
	case "ebs":
		ns = "AWS/EBS"
	default:
		log.Fatal("Not implemented namespace for cloudwatch metric:" + *service)
	}
	return &ns
}

func createStaticDimensions(dimensions []dimension) (output []*cloudwatch.Dimension) {
	for _, d := range dimensions {
		output = append(output, buildDimension(d.Name, d.Value))
	}

	return output
}

func getDimensions(service *string, resourceArn *string) (dimensions []*cloudwatch.Dimension) {
	arnParsed, err := arn.Parse(*resourceArn)

	if err != nil {
		panic(err)
	}

	switch *service {
	case "ec2":
		dimensions = buildBaseDimension(arnParsed.Resource, "InstanceId", "instance/")
	case "elb":
		dimensions = buildBaseDimension(arnParsed.Resource, "LoadBalancerName", "loadbalancer/")
	case "rds":
		dimensions = buildBaseDimension(arnParsed.Resource, "DBInstanceIdentifier", "db:")
	case "ec":
		dimensions = buildBaseDimension(arnParsed.Resource, "CacheClusterId", "cluster:")
	case "es":
		dimensions = buildBaseDimension(arnParsed.Resource, "DomainName", "domain/")
		dimensions = append(dimensions, buildDimension("ClientId", arnParsed.AccountID))
	case "s3":
		dimensions = buildBaseDimension(arnParsed.Resource, "BucketName", "")
		dimensions = append(dimensions, buildDimension("StorageType", "AllStorageTypes"))
	case "efs":
		dimensions = buildBaseDimension(arnParsed.Resource, "FileSystemId", "file-system/")
	case "ebs":
		dimensions = buildBaseDimension(arnParsed.Resource, "VolumeId", "volume/")
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

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData) []*prometheusData {
	output := make([]*prometheusData, 0)
	for _, c := range cwd {

		for _, statistic := range c.Statistics {
			name := "aws_" + strings.ToLower(*c.Service) + "_" + strings.ToLower(promString(*c.Metric)) + "_" + strings.ToLower(promString(statistic))

			var points []*float64

			for _, point := range c.Points {
				switch statistic {
				case "Maximum":
					if point.Maximum != nil {
						points = append(points, point.Maximum)
					}
				case "Minimum":
					if point.Minimum != nil {
						points = append(points, point.Minimum)
					}
				case "Sum":
					if point.Sum != nil {
						points = append(points, point.Sum)
					}
				default:
					log.Fatal("Not implemented statistics" + statistic)
				}
			}

			if len(points) == 0 {
				if *c.NilToZero {
					helper := float64(0)
					sliceHelper := []*float64{&helper}
					points = sliceHelper
				}
			}

			if len(points) > 0 {
				promLabels := make(map[string]string)
				promLabels["name"] = *c.ID

				lastValue := points[len(points)-1]

				p := prometheusData{
					name:   &name,
					labels: promLabels,
					value:  lastValue,
				}
				output = append(output, &p)
			}
		}
	}

	return output
}
