package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

var percentile = regexp.MustCompile(`^p(\d{1,2}(\.\d{0,2})?|100)$`)

type cloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
}

type cloudwatchData struct {
	ID                     *string
	Metric                 *string
	Service                *string
	Statistics             []string
	Points                 []*cloudwatch.Datapoint
	NilToZero              *bool
	AddCloudwatchTimestamp *bool
	CustomTags             []tag
	Tags                   []tag
	Dimensions 			   []*cloudwatch.Dimension
}

func createCloudwatchSession(region *string, roleArn string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	config := &aws.Config{Region: region}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	return cloudwatch.New(sess, config)
}

func createGetMetricStatisticsInput(dimensions []*cloudwatch.Dimension, namespace *string, metric metric) (output *cloudwatch.GetMetricStatisticsInput) {
	period := int64(metric.Period)
	length := metric.Length
	delay := metric.Delay
	endTime := time.Now().Add(-time.Duration(delay) * time.Second)
	startTime := time.Now().Add(-(time.Duration(length) + time.Duration(delay)) * time.Second)

	var statistics []*string
	var extendedStatistics []*string
	for _, statistic := range metric.Statistics {
		if percentile.MatchString(statistic) {
			extendedStatistics = append(extendedStatistics, aws.String(statistic))
		} else {
			statistics = append(statistics, aws.String(statistic))
		}
	}

	output = &cloudwatch.GetMetricStatisticsInput{
		Dimensions:         dimensions,
		Namespace:          namespace,
		StartTime:          &startTime,
		EndTime:            &endTime,
		Period:             &period,
		MetricName:         &metric.Name,
		Statistics:         statistics,
		ExtendedStatistics: extendedStatistics,
	}

	if *debug {
		if len(statistics) != 0 {
			log.Println("CLI helper - " +
				"aws cloudwatch get-metric-statistics" +
				" --metric-name " + metric.Name +
				" --dimensions " + dimensionsToCliString(dimensions) +
				" --namespace " + *namespace +
				" --statistics " + *statistics[0] +
				" --period " + strconv.FormatInt(period, 10) +
				" --start-time " + startTime.Format(time.RFC3339) +
				" --end-time " + endTime.Format(time.RFC3339))
		}
		log.Println(*output)
	}
	return output
}

func createListMetricsInput(dimensions []*cloudwatch.Dimension, namespace *string) (output *cloudwatch.ListMetricsInput) {
	var dimensionsFilter []*cloudwatch.DimensionFilter

	for _, dim := range dimensions {
		dimensionsFilter = append(dimensionsFilter, &cloudwatch.DimensionFilter{Name: dim.Name, Value: dim.Value})
	}
	output = &cloudwatch.ListMetricsInput{
		MetricName: nil,
		Dimensions: dimensionsFilter,
		Namespace:  namespace,
		NextToken:  nil,
	}
	return output
}

func dimensionsToCliString(dimensions []*cloudwatch.Dimension) (output string) {
	for _, dim := range dimensions {
		output = output + "Name=" + *dim.Name + ",Value=" + *dim.Value
		fmt.Println(output)
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
	case "alb":
		ns = "AWS/ApplicationELB"
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
	case "vpn":
		ns = "AWS/VPN"
	case "lambda":
		ns = "AWS/Lambda"
	case "kinesis":
		ns = "AWS/Kinesis"
	default:
		log.Fatal("Not implemented namespace for cloudwatch metric: " + *service)
	}
	return &ns
}

func createStaticDimensions(dimensions []dimension) (output []*cloudwatch.Dimension) {
	for _, d := range dimensions {
		output = append(output, buildDimension(d.Name, d.Value))
	}

	return output
}

func getDimensionValueForName(name string, resp *cloudwatch.ListMetricsOutput) (value *string) {
	for _, metric := range resp.Metrics {
		for _, dim := range metric.Dimensions {
			if strings.Compare(*dim.Name, name) == 0 {
				return dim.Value
			}
		}
	}
	return nil
}

func getResourceValue(resourceName string, dimensions []*cloudwatch.Dimension, namespace *string, clientCloudwatch cloudwatchInterface) (dimensionResourceName *string) {
	c := clientCloudwatch.client
	filter := createListMetricsInput(dimensions, namespace)
	req, resp := c.ListMetricsRequest(filter)
	err := req.Send()

	if err != nil {
		panic(err)
	}

	cloudwatchAPICounter.Inc()
	return getDimensionValueForName(resourceName, resp)
}

func queryAvailableDimensions(resource string, namespace *string, clientCloudwatch cloudwatchInterface) (dimensions []*cloudwatch.Dimension) {

	if !strings.HasSuffix(*namespace, "ApplicationELB") {
		log.Fatal("Not implemented queryAvailableDimensions: " + *namespace)
		return nil
	}

	if strings.HasPrefix(resource, "targetgroup/") {
		dimensions = append(dimensions, buildDimension("TargetGroup", resource))
		loadBalancerName := getResourceValue("LoadBalancer", dimensions, namespace, clientCloudwatch)
		if loadBalancerName != nil {
			dimensions = append(dimensions, buildDimension("LoadBalancer", *loadBalancerName))
		}

	} else if strings.HasPrefix(resource, "loadbalancer/") || strings.HasPrefix(resource, "app/") {
		trimmedDimensionValue := strings.Replace(resource, "loadbalancer/", "", -1)
		dimensions = append(dimensions, buildDimension("LoadBalancer", trimmedDimensionValue))
	}

	return dimensions
}

func detectDimensionsByService(service *string, resourceArn *string, clientCloudwatch cloudwatchInterface) (dimensions []*cloudwatch.Dimension) {
	arnParsed, err := arn.Parse(*resourceArn)

	if err != nil {
		panic(err)
	}

	switch *service {
	case "ec2":
		dimensions = buildBaseDimension(arnParsed.Resource, "InstanceId", "instance/")
	case "elb":
		dimensions = buildBaseDimension(arnParsed.Resource, "LoadBalancerName", "loadbalancer/")
	case "alb":
		dimensions = queryAvailableDimensions(arnParsed.Resource, getNamespace(service), clientCloudwatch)
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
	case "vpn":
		dimensions = buildBaseDimension(arnParsed.Resource, "VpnId", "vpn-connection/")
	case "lambda":
		dimensions = buildBaseDimension(arnParsed.Resource, "FunctionName", "function:")
	case "kinesis":
		dimensions = buildBaseDimension(arnParsed.Resource, "StreamName", "stream/")
	default:
		log.Fatal("Not implemented cloudwatch metric: " + *service)
	}

	return dimensions
}

func addAdditionalDimensions(startingDimensions []*cloudwatch.Dimension, additionalDimensions []dimension) (dimensions []*cloudwatch.Dimension) {
	dimensions = startingDimensions
	for _, dimension := range additionalDimensions {
		dimensions = append(dimensions, buildDimension(dimension.Name, dimension.Value))
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

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)
	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			name := "aws_" + strings.ToLower(*c.Service) + "_" + strings.ToLower(promString(*c.Metric)) + "_" + strings.ToLower(promString(statistic))

			var points []*float64

			var timestamp time.Time

			for _, point := range c.Points {
				if point.Timestamp != nil && timestamp.Before(*point.Timestamp) {
					timestamp = *point.Timestamp
				}

				switch {
				case statistic == "Maximum":
					if point.Maximum != nil {
						points = append(points, point.Maximum)
					}
				case statistic == "Minimum":
					if point.Minimum != nil {
						points = append(points, point.Minimum)
					}
				case statistic == "Sum":
					if point.Sum != nil {
						points = append(points, point.Sum)
					}
				case statistic == "Average":
					if point.Average != nil {
						points = append(points, point.Average)
					}
				case percentile.MatchString(statistic):
					if data, ok := point.ExtendedStatistics[statistic]; ok {
						points = append(points, data)
					}
				default:
					log.Fatal("Not implemented statistics: " + statistic)
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

				for _, label := range c.CustomTags {
					promLabels["custom_tag_"+label.Key] = label.Value
				}
				for _, tag := range c.Tags {
					promLabels["tag_"+promStringTag(tag.Key)] = tag.Value
				}
				for _, dimension := range c.Dimensions {
					promLabels["dimension_"+promStringTag(*dimension.Name)] = *dimension.Value
				}

				var value float64 = 0

				if statistic == "Average" {
					var total float64 = 0
					for _, p := range points {
						total += *p
					}
					value = total / float64(len(points))
				} else {
					value = *points[len(points)-1]
				}
				p := PrometheusMetric{
					name:             &name,
					labels:           promLabels,
					value:            &value,
					timestamp:        timestamp,
					includeTimestamp: *c.AddCloudwatchTimestamp,
				}
				output = append(output, &p)
			}
		}
	}

	return output
}
