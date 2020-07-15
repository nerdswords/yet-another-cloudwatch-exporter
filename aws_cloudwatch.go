package main

import (
	"fmt"
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
	"github.com/fatih/structs"

	log "github.com/sirupsen/logrus"
)

var percentile = regexp.MustCompile(`^p(\d{1,2}(\.\d{0,2})?|100)$`)

type cloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
}

type cloudwatchData struct {
	ID                      *string
	MetricID                *string
	Metric                  *string
	Service                 *string
	Statistics              []string
	Points                  []*cloudwatch.Datapoint
	GetMetricDataPoint      *float64
	GetMetricDataTimestamps *time.Time
	NilToZero               *bool
	AddCloudwatchTimestamp  *bool
	CustomTags              []tag
	Tags                    []tag
	Dimensions              []*cloudwatch.Dimension
	Region                  *string
	Period                  int64
}

func createCloudwatchSession(region *string, roleArn string) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	maxCloudwatchRetries := 5

	config := &aws.Config{Region: region, MaxRetries: &maxCloudwatchRetries}

	if *debug {
		config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
	}

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

	if len(statistics) != 0 {
		log.Debug("CLI helper - " +
			"aws cloudwatch get-metric-statistics" +
			" --metric-name " + metric.Name +
			" --dimensions " + dimensionsToCliString(dimensions) +
			" --namespace " + *namespace +
			" --statistics " + *statistics[0] +
			" --period " + strconv.FormatInt(period, 10) +
			" --start-time " + startTime.Format(time.RFC3339) +
			" --end-time " + endTime.Format(time.RFC3339))
	}
	log.Debug(*output)
	return output
}

func findGetMetricDataById(getMetricDatas []cloudwatchData, value string) (cloudwatchData, error) {
	var g cloudwatchData
	for _, getMetricData := range getMetricDatas {
		if *getMetricData.MetricID == value {
			return getMetricData, nil
		}
	}
	return g, fmt.Errorf("Metric with id %s not found", value)
}

func createGetMetricDataInput(getMetricData []cloudwatchData, namespace *string, length int, delay int) (output *cloudwatch.GetMetricDataInput) {
	var metricsDataQuery []*cloudwatch.MetricDataQuery
	for _, data := range getMetricData {
		metricStat := &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Dimensions: data.Dimensions,
				MetricName: data.Metric,
				Namespace:  namespace,
			},
			Period: &data.Period,
			Stat:   &data.Statistics[0],
		}
		ReturnData := true
		metricsDataQuery = append(metricsDataQuery, &cloudwatch.MetricDataQuery{
			Id:         data.MetricID,
			MetricStat: metricStat,
			ReturnData: &ReturnData,
		})

	}
	endTime := time.Now().Add(-time.Duration(delay) * time.Second)
	startTime := time.Now().Add(-(time.Duration(length) + time.Duration(delay)) * time.Second)
	dataPointOrder := "TimestampDescending"
	output = &cloudwatch.GetMetricDataInput{
		EndTime:           &endTime,
		StartTime:         &startTime,
		MetricDataQueries: metricsDataQuery,
		ScanBy:            &dataPointOrder,
	}

	return output
}

func createListMetricsInput(dimensions []*cloudwatch.Dimension, namespace *string, metricsName *string) (output *cloudwatch.ListMetricsInput) {
	var dimensionsFilter []*cloudwatch.DimensionFilter

	for _, dim := range dimensions {
		if dim.Value != nil {
			dimensionsFilter = append(dimensionsFilter, &cloudwatch.DimensionFilter{Name: dim.Name, Value: dim.Value})
		}
	}
	output = &cloudwatch.ListMetricsInput{
		MetricName: metricsName,
		Dimensions: dimensionsFilter,
		Namespace:  namespace,
		NextToken:  nil,
	}
	return output
}

func createListMetricsOutput(dimensions []*cloudwatch.Dimension, namespace *string, metricsName *string) (output *cloudwatch.ListMetricsOutput) {
	Metrics := []*cloudwatch.Metric{{
		MetricName: metricsName,
		Dimensions: dimensions,
		Namespace:  namespace,
	}}
	output = &cloudwatch.ListMetricsOutput{
		Metrics:   Metrics,
		NextToken: nil,
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

	log.Debug(filter)

	resp, err := c.GetMetricStatistics(filter)

	log.Debug(resp)

	cloudwatchAPICounter.Inc()
	cloudwatchGetMetricStatisticsAPICounter.Inc()

	if err != nil {
		log.Warning(err)
		return nil
	}

	return resp.Datapoints
}

func (iface cloudwatchInterface) getMetricData(filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput {
	c := iface.client

	var resp cloudwatch.GetMetricDataOutput

	if *debug {
		log.Println(filter)
	}

	// Using the paged version of the function
	err := c.GetMetricDataPages(filter,
		func(page *cloudwatch.GetMetricDataOutput, lastPage bool) bool {
			cloudwatchAPICounter.Inc()
			cloudwatchGetMetricDataAPICounter.Inc()
			resp.MetricDataResults = append(resp.MetricDataResults, page.MetricDataResults...)
			return !lastPage
		})

	if *debug {
		log.Println(resp)
	}

	if err != nil {
		log.Warning(err)
		return nil
	}
	return &resp
}

func getNamespace(service *string) *string {
	var ns string
	var ok bool

	namespaces := map[string]string{
		"alb":                   "AWS/ApplicationELB",
		"appsync":               "AWS/AppSync",
		"asg":                   "AWS/AutoScaling",
		"cf":                    "AWS/CloudFront",
		"dynamodb":              "AWS/DynamoDB",
		"ebs":                   "AWS/EBS",
		"ec":                    "AWS/ElastiCache",
		"ec2":                   "AWS/EC2",
		"ecs-svc":               "AWS/ECS",
		"ecs-containerinsights": "ECS/ContainerInsights",
		"efs":                   "AWS/EFS",
		"elb":                   "AWS/ELB",
		"emr":                   "AWS/ElasticMapReduce",
		"es":                    "AWS/ES",
		"firehose":              "AWS/Firehose",
		"fsx":                   "AWS/FSx",
		"kafka":                 "AWS/Kafka",
		"kinesis":               "AWS/Kinesis",
		"lambda":                "AWS/Lambda",
		"ngw":                   "AWS/NATGateway",
		"nlb":                   "AWS/NetworkELB",
		"rds":                   "AWS/RDS",
		"redshift":              "AWS/Redshift",
		"r53r":                  "AWS/Route53Resolver",
		"s3":                    "AWS/S3",
		"sfn":                   "AWS/States",
		"sns":                   "AWS/SNS",
		"sqs":                   "AWS/SQS",
		"tgw":                   "AWS/TransitGateway",
		"tgwa":                  "AWS/TransitGateway",
		"vpn":                   "AWS/VPN",
	}
	if ns, ok = namespaces[*service]; !ok {
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

func getDimensionValueForResource(name string, fullMetricsList *cloudwatch.ListMetricsOutput) (value *string) {
	for _, metric := range fullMetricsList.Metrics {
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

func filterDimensionsWithoutValueByDimensionsWithValue(
	dimensionsWithoutValue []*cloudwatch.Dimension,
	dimensionsWithValue []*cloudwatch.Dimension) (dimensions []*cloudwatch.Dimension) {

	for _, dimension := range dimensionsWithoutValue {
		if !dimensionIsInListWithoutValues(dimension, dimensionsWithValue) {
			dimensions = append(dimensions, dimension)
		}
	}
	return dimensions
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
	callListMetrics := false
	for _, dimension := range dimensions {
		if structs.HasZero(dimension) {
			callListMetrics = true
			break
		}
	}
	if callListMetrics {
		var res cloudwatch.ListMetricsOutput
		err := c.ListMetricsPages(filter,
			func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
				res.Metrics = append(res.Metrics, page.Metrics...)
				return !lastPage
			})
		cloudwatchAPICounter.Inc()
		if err != nil {
			log.Warning(err)
		}
		resp = filterMetricsBasedOnDimensions(dimensions, &res)
	} else {
		resp = createListMetricsOutput(dimensions, getNamespace(serviceName), &metric.Name)
	}
	return resp
}

func getFullMetricsList(serviceName *string, metric metric, clientCloudwatch cloudwatchInterface) (resp *cloudwatch.ListMetricsOutput) {
	c := clientCloudwatch.client
	filter := createListMetricsInput(nil, getNamespace(serviceName), &metric.Name)
	var res cloudwatch.ListMetricsOutput
	err := c.ListMetricsPages(filter,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			res.Metrics = append(res.Metrics, page.Metrics...)
			return !lastPage
		})
	cloudwatchAPICounter.Inc()
	if err != nil {
		log.Fatal(err)
	}
	return &res
}

func filterMetricsBasedOnDimensionsWithValues(
	dimensionsWithValue []*cloudwatch.Dimension,
	dimensionsWithoutValue []*cloudwatch.Dimension,
	metricsToFilter *cloudwatch.ListMetricsOutput) *cloudwatch.ListMetricsOutput {

	var numberOfDimensions = len(dimensionsWithValue) + len(dimensionsWithoutValue)
	var output cloudwatch.ListMetricsOutput
	for _, metric := range metricsToFilter.Metrics {
		if len(metric.Dimensions) == numberOfDimensions {
			shouldAddMetric := true
			for _, metricDimension := range metric.Dimensions {
				shouldAddMetric = shouldAddMetric &&
					(dimensionIsInListWithValues(metricDimension, dimensionsWithValue) ||
						dimensionIsInListWithoutValues(metricDimension, dimensionsWithoutValue))
				if !shouldAddMetric {
					break
				}
			}
			if shouldAddMetric {
				output.Metrics = append(output.Metrics, metric)
			}
		}
	}
	return &output
}

func dimensionIsInListWithValues(
	dimension *cloudwatch.Dimension,
	dimensionsList []*cloudwatch.Dimension) bool {
	for _, dimensionInList := range dimensionsList {
		if *dimension.Name == *dimensionInList.Name &&
			*dimension.Value == *dimensionInList.Value {
			return true
		}
	}
	return false
}

func dimensionIsInListWithoutValues(
	dimension *cloudwatch.Dimension,
	dimensionsList []*cloudwatch.Dimension) bool {
	for _, dimensionInList := range dimensionsList {
		if *dimension.Name == *dimensionInList.Name {
			return true
		}
	}
	return false
}

func getDimensionfromMetric(resp *cloudwatch.ListMetricsOutput) []*cloudwatch.Dimension {
	for _, metric := range resp.Metrics {
		return metric.Dimensions
	}
	return nil
}

func queryAvailableDimensions(resource string, namespace *string, fullMetricsList *cloudwatch.ListMetricsOutput) (dimensions []*cloudwatch.Dimension) {

	if !strings.HasSuffix(*namespace, "ApplicationELB") {
		log.Fatal("Not implemented queryAvailableDimensions: " + *namespace)
		return nil
	}

	if strings.HasPrefix(resource, "targetgroup/") {
		dimensions = append(dimensions, buildDimension("TargetGroup", resource))
		resp := filterMetricsBasedOnDimensionsWithValues(dimensions, []*cloudwatch.Dimension{buildDimensionWithoutValue("LoadBalancer")}, fullMetricsList)
		if resp != nil {
			dimensions = getDimensionfromMetric(resp)
		}

	} else if strings.HasPrefix(resource, "loadbalancer/") || strings.HasPrefix(resource, "app/") {
		trimmedDimensionValue := strings.Replace(resource, "loadbalancer/", "", -1)
		dimensions = append(dimensions, buildDimension("LoadBalancer", trimmedDimensionValue))
	}

	return dimensions
}

func detectDimensionsByService(service *string, resourceArn *string, fullMetricsList *cloudwatch.ListMetricsOutput) (dimensions []*cloudwatch.Dimension) {
	arnParsed, err := arn.Parse(*resourceArn)

	if err != nil && *service != "tgwa" {
		log.Warning(err)
		return (dimensions)
	}

	switch *service {
	case "alb":
		dimensions = queryAvailableDimensions(arnParsed.Resource, getNamespace(service), fullMetricsList)
	case "appsync":
		dimensions = buildBaseDimension(arnParsed.Resource, "GraphQLAPIId", "apis/")
	case "asg":
		dimensions = buildBaseDimension(arnParsed.Resource, "AutoScalingGroupName", "autoScalingGroupName/")
	case "cf":
		dimensions = buildBaseDimension(arnParsed.Resource, "DistributionId", "distribution/")
		dimensions = append(dimensions, buildDimension("Region", "Global"))
	case "dynamodb":
		dimensions = buildBaseDimension(arnParsed.Resource, "TableName", "table/")
	case "ebs":
		dimensions = buildBaseDimension(arnParsed.Resource, "VolumeId", "volume/")
	case "ec":
		dimensions = buildBaseDimension(arnParsed.Resource, "CacheClusterId", "cluster:")
	case "ec2":
		dimensions = buildBaseDimension(arnParsed.Resource, "InstanceId", "instance/")
	case "ecs-svc", "ecs-containerinsights":
		parsedResource := strings.Split(arnParsed.Resource, "/")
		if parsedResource[0] == "service" {
			dimensions = append(dimensions, buildDimension("ClusterName", parsedResource[1]), buildDimension("ServiceName", parsedResource[2]))
		}
		if parsedResource[0] == "cluster" {
			dimensions = append(dimensions, buildDimension("ClusterName", parsedResource[1]))
		}
	case "efs":
		dimensions = buildBaseDimension(arnParsed.Resource, "FileSystemId", "file-system/")
	case "elb":
		dimensions = buildBaseDimension(arnParsed.Resource, "LoadBalancerName", "loadbalancer/")
	case "emr":
		dimensions = buildBaseDimension(arnParsed.Resource, "JobFlowId", "cluster/")
	case "es":
		dimensions = buildBaseDimension(arnParsed.Resource, "DomainName", "domain/")
		dimensions = append(dimensions, buildDimension("ClientId", arnParsed.AccountID))
	case "firehose":
		dimensions = buildBaseDimension(arnParsed.Resource, "DeliveryStreamName", "deliverystream/")
	case "fsx":
		dimensions = buildBaseDimension(arnParsed.Resource, "FileSystemId", "file-system/")
	case "kinesis":
		dimensions = buildBaseDimension(arnParsed.Resource, "StreamName", "stream/")
	case "lambda":
		dimensions = buildBaseDimension(arnParsed.Resource, "FunctionName", "function:")
	case "ngw":
		dimensions = buildBaseDimension(arnParsed.Resource, "NatGatewayId", "natgateway/")
	case "nlb":
		dimensions = buildBaseDimension(arnParsed.Resource, "LoadBalancer", "loadbalancer/")
	case "rds":
		dimensions = buildBaseDimension(arnParsed.Resource, "DBInstanceIdentifier", "db:")
	case "redshift":
		dimensions = buildBaseDimension(arnParsed.Resource, "ClusterIdentifier", "cluster:")
	case "r53r":
		dimensions = buildBaseDimension(arnParsed.Resource, "EndpointId", "resolver-endpoint/")
	case "s3":
		dimensions = buildBaseDimension(arnParsed.Resource, "BucketName", "")
	case "sfn":
		// The value of StateMachineArn returned is the Name, not the ARN
		// We are setting the value to the ARN in order to correlate dimensions with metric values
		// (StateMachineArn will be set back to the name later, once all the filtering is complete)
		// https://docs.aws.amazon.com/step-functions/latest/dg/procedure-cw-metrics.html
		dimensions = append(dimensions, buildDimension("StateMachineArn", *resourceArn))
	case "sns":
		dimensions = buildBaseDimension(arnParsed.Resource, "TopicName", "")
	case "sqs":
		dimensions = buildBaseDimension(arnParsed.Resource, "QueueName", "")
	case "tgw":
		dimensions = buildBaseDimension(arnParsed.Resource, "TransitGateway", "transit-gateway/")
	case "tgwa":
		parsedResource := strings.Split(*resourceArn, "/")
		dimensions = append(dimensions, buildDimension("TransitGateway", parsedResource[0]), buildDimension("TransitGatewayAttachment", parsedResource[1]))
	case "vpn":
		dimensions = buildBaseDimension(arnParsed.Resource, "VpnId", "vpn-connection/")
	case "kafka":
		cluster := strings.Split(arnParsed.Resource, "/")[1]
		dimensions = append(dimensions, buildDimension("Cluster Name", cluster))
	default:
		log.Fatal("Not implemented cloudwatch metric: " + *service)
	}

	return dimensions
}

func addAdditionalDimensions(startingDimensions []*cloudwatch.Dimension, additionalDimensions []dimension) (dimensions []*cloudwatch.Dimension) {
	// Copy startingDimensions before appending additionalDimensions, since append(x, ...) can modify x
	dimensions = append(dimensions, startingDimensions...)
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
		var albSuffix, tgSuffix string
		for _, dimension := range dimensions {
			if *dimension.Name == "TargetGroup" {
				tgSuffix = "tg"
			}
			if *dimension.Name == "LoadBalancer" {
				albSuffix = "alb"
			}
		}
		if albSuffix != "" && tgSuffix != "" {
			return albSuffix + "_" + tgSuffix
		} else if albSuffix == "" && tgSuffix != "" {
			return tgSuffix
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

func getStateMachineNameFromArn(resourceArn string) string {
	arnParsed, err := arn.Parse(resourceArn)
	if err != nil {
		log.Warning(err)
		return ""
	}
	parsedResource := strings.Split(arnParsed.Resource, ":")
	return parsedResource[1]
}

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			serviceName := fixServiceName(c.Service, c.Dimensions)
			name := "aws_" + serviceName + "_" + strings.ToLower(promString(*c.Metric)) + "_" + strings.ToLower(promString(statistic))
			var exportedDatapoint *float64
			var averageDataPoints []*float64
			var timestamp time.Time
			if c.GetMetricDataPoint != nil {
				exportedDatapoint = c.GetMetricDataPoint
				timestamp = *c.GetMetricDataTimestamps
			} else {
				datapoints := c.Points
				// sorting by timestamps so we can consistently export the most updated datapoint
				// assuming Timestamp field in cloudwatch.Datapoint struct is never nil
				sort.Slice(datapoints, func(i, j int) bool {
					jTimestamp := *datapoints[j].Timestamp
					return datapoints[i].Timestamp.Before(jTimestamp)
				})

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
					promLabels["custom_tag_"+promStringTag(label.Key)] = label.Value
				}
				for _, tag := range c.Tags {
					promLabels["tag_"+promStringTag(tag.Key)] = tag.Value
				}

				for _, dimension := range c.Dimensions {
					promLabels["dimension_"+promStringTag(*dimension.Name)] = *dimension.Value
				}

				// Inject the sfn name back as a label
				switch serviceName {
				case "sfn":
					promLabels["dimension_StateMachineArn"] = getStateMachineNameFromArn(*c.ID)
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
