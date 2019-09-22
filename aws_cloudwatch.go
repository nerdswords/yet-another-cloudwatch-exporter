package main

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"sort"
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
	Dimensions             []*cloudwatch.Dimension
	Region                 *string
}

func createCloudwatchSession(region *string, roleArn string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	maxCloudwatchRetries := 5

	config := &aws.Config{Region: region, MaxRetries: &maxCloudwatchRetries}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}

	return cloudwatch.New(sess, config)
}

func createGetMetricDataInput(dimensions []*cloudwatch.Dimension, namespace *string, metric metric) (output *cloudwatch.GetMetricDataInput) {
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

	var queries []*cloudwatch.MetricDataQuery

	metric = &cloudwatch.Metric{
		MetricName: &metric.Name,
		Dimensions: dimensions,
		Namespace:  namespace,
	}

	metricStat = &cloudwatch.MetricStat{
		metric: &metric,
		Period: &period,

		// The statistic to return. It can include any CloudWatch statistic or extended
		// statistic.
		Stat: statistics,
	}

	query = &cloudwatch.metricDataQuery{
		Id:         aws.String("static"),
		MetricStat: &metricStat,
	}

	query := cloudwatch.MetricDataQuery{}

	queries = append(*queries, query)

	metricDataInput = &cloudwatch.GetMetricDataInput{
		StartTime:         &startTime,
		EndTime:           &endTime,
		MetricDataQueries: queries,
	}

	output, err := cloudwatch.GetMetricData(metricDataInput)

	if err != nil {
		panic(err)
	}

	fmt.Println(output)

	output = &cloudwatch.GetMetricStatisticsInput{
		Dimensions:         dimensions,
		Namespace:          namespace,
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

func createListMetricsInput(dimensions []*cloudwatch.Dimension, namespace *string, metricsName *string) (output *cloudwatch.ListMetricsInput) {
	var dimensionsFilter []*cloudwatch.DimensionFilter

	for _, dim := range dimensions {
		dimensionsFilter = append(dimensionsFilter, &cloudwatch.DimensionFilter{Name: dim.Name, Value: dim.Value})
	}
	output = &cloudwatch.ListMetricsInput{
		MetricName: metricsName,
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
	case "ecs-svc":
		ns = "AWS/ECS"
	case "nlb":
		ns = "AWS/NetworkELB"
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
	case "dynamodb":
		ns = "AWS/DynamoDB"
	case "emr":
		ns = "AWS/ElasticMapReduce"
	case "asg":
		ns = "AWS/AutoScaling"
	case "sqs":
		ns = "AWS/SQS"
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

func keysofDimension(dimensions []*cloudwatch.Dimension) (keys []string) {
	for _, dimension := range dimensions {
		keys = append(keys, *dimension.Name)
	}
	return keys
}

func filterMetricsBasedOnDimensions(dimensions []*cloudwatch.Dimension, resp *cloudwatch.ListMetricsOutput) *cloudwatch.ListMetricsOutput {
	var output cloudwatch.ListMetricsOutput
	selectedDimensionKeys := keysofDimension(dimensions)
	sort.Strings(selectedDimensionKeys)
	for _, metric := range resp.Metrics {
		metricsDimensionkeys := keysofDimension(metric.Dimensions)
		sort.Strings(metricsDimensionkeys)
		if reflect.DeepEqual(metricsDimensionkeys, selectedDimensionKeys) {
			output.Metrics = append(output.Metrics, metric)
		}
	}
	return &output
}

func getResourceValue(resourceName string, dimensions []*cloudwatch.Dimension, namespace *string, clientCloudwatch cloudwatchInterface) (dimensionResourceName *string) {
	c := clientCloudwatch.client
	filter := createListMetricsInput(dimensions, namespace, nil)
	req, resp := c.ListMetricsRequest(filter)
	err := req.Send()

	if err != nil {
		panic(err)
	}

	cloudwatchAPICounter.Inc()
	return getDimensionValueForName(resourceName, resp)
}

func getAwsDimensions(job job) (dimensions []*cloudwatch.Dimension) {
	for _, awsDimension := range job.AwsDimensions {
		dimensions = append(dimensions, buildDimensionWithoutValue(awsDimension))
	}
	return dimensions
}

func getMetricsList(dimensions []*cloudwatch.Dimension, serviceName *string, metric metric, clientCloudwatch cloudwatchInterface) (resp *cloudwatch.ListMetricsOutput) {
	c := clientCloudwatch.client
	filter := createListMetricsInput(dimensions, getNamespace(serviceName), &metric.Name)
	req, resp := c.ListMetricsRequest(filter)
	cloudwatchAPICounter.Inc()
	err := req.Send()

	if err != nil {
		panic(err)
	}

	resp = filterMetricsBasedOnDimensions(dimensions, resp)
	return resp
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
	case "ecs-svc":
		cluster := strings.Split(arnParsed.Resource, "/")[1]
		service := strings.Split(arnParsed.Resource, "/")[2]
		dimensions = append(dimensions, buildDimension("ClusterName", cluster))
		dimensions = append(dimensions, buildDimension("ServiceName", service))
	case "alb":
		dimensions = queryAvailableDimensions(arnParsed.Resource, getNamespace(service), clientCloudwatch)
	case "nlb":
		dimensions = buildBaseDimension(arnParsed.Resource, "LoadBalancer", "loadbalancer/")
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
	case "dynamodb":
		dimensions = buildBaseDimension(arnParsed.Resource, "TableName", "table/")
	case "emr":
		dimensions = buildBaseDimension(arnParsed.Resource, "JobFlowId", "cluster/")
	case "asg":
		dimensions = buildBaseDimension(arnParsed.Resource, "AutoScalingGroupName", "autoScalingGroupName/")
	case "sqs":
		dimensions = buildBaseDimension(arnParsed.Resource, "QueueName", "")
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

func buildDimensionWithoutValue(key string) *cloudwatch.Dimension {
	dimension := cloudwatch.Dimension{
		Name: &key,
	}
	return &dimension
}

func buildDimension(key string, value string) *cloudwatch.Dimension {
	dimension := cloudwatch.Dimension{
		Name:  &key,
		Value: &value,
	}
	return &dimension
}

func fixServiceName(serviceName *string, dimensions []*cloudwatch.Dimension) string {
	var suffixName string
	if *serviceName == "alb" {
		for _, dimension := range dimensions {
			if *dimension.Name == "TargetGroup" {
				suffixName = "tg"
			}
		}
	}
	if *serviceName == "elb" {
		for _, dimension := range dimensions {
			if *dimension.Name == "AvailabilityZone" {
				suffixName = "_az"
			}
		}
	}
	return promString(*serviceName) + suffixName
}

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)
	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			name := "aws_" + fixServiceName(c.Service, c.Dimensions) + "_" + strings.ToLower(promString(*c.Metric)) + "_" + strings.ToLower(promString(statistic))

			datapoints := c.Points
			// sorting by timestamps so we can consistently export the most updated datapoint
			// assuming Timestamp field in cloudwatch.Datapoint struct is never nil
			sort.Slice(datapoints, func(i, j int) bool {
				jTimestamp := *datapoints[j].Timestamp
				return datapoints[i].Timestamp.Before(jTimestamp)
			})

			var exportedDatapoint *float64
			var averageDataPoints []*float64
			var timestamp time.Time
			for _, datapoint := range datapoints {
				switch {
				case statistic == "Maximum":
					if datapoint.Maximum != nil {
						exportedDatapoint = datapoint.Maximum
						timestamp = *datapoint.Timestamp
						break
					}
				case statistic == "Minimum":
					if datapoint.Minimum != nil {
						exportedDatapoint = datapoint.Minimum
						timestamp = *datapoint.Timestamp
						break
					}
				case statistic == "Sum":
					if datapoint.Sum != nil {
						exportedDatapoint = datapoint.Sum
						timestamp = *datapoint.Timestamp
						break
					}
				case statistic == "Average":
					if datapoint.Average != nil {
						if datapoint.Timestamp.After(timestamp) {
							timestamp = *datapoint.Timestamp
						}
						averageDataPoints = append(averageDataPoints, datapoint.Average)
					}
				case percentile.MatchString(statistic):
					if data, ok := datapoint.ExtendedStatistics[statistic]; ok {
						exportedDatapoint = data
						timestamp = *datapoint.Timestamp
						break
					}
				default:
					log.Fatal("Not implemented statistics: " + statistic)
				}
			}

			var exportedAverage float64
			if len(averageDataPoints) > 0 {
				var total float64
				for _, p := range averageDataPoints {
					total += *p
				}
				exportedAverage = total / float64(len(averageDataPoints))
				exportedDatapoint = &exportedAverage
			}
			var zero float64
			includeTimestamp := *c.AddCloudwatchTimestamp
			if exportedDatapoint == nil && *c.NilToZero {
				exportedDatapoint = &zero
				includeTimestamp = false
			}
			if exportedDatapoint != nil {
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

				promLabels["region"] = *c.Region

				p := PrometheusMetric{
					name:             &name,
					labels:           promLabels,
					value:            exportedDatapoint,
					timestamp:        timestamp,
					includeTimestamp: includeTimestamp,
				}
				output = append(output, &p)
			}
		}
	}

	return output
}
