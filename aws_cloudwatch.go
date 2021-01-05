package main

import (
	"errors"
	"fmt"
	"math"
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
	endtime                time.Time
}

var labelMap = make(map[string][]string)

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

func createGetMetricDataInput(getMetricData []cloudwatchData, namespace *string, length int, delay int, now time.Time) (output *cloudwatch.GetMetricDataInput) {
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

	var endTime time.Time
	var startTime time.Time
	if now.IsZero() {
		//This is first run
		now = time.Now().Round(5 * time.Minute)
		endTime = now.Add(-time.Duration(delay) * time.Second)
		startTime = now.Add(-(time.Duration(length) + time.Duration(delay)) * time.Second)
	} else {
		endTime = now.Add(time.Duration(length) * time.Second)
		startTime = now
	}

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

func dimensionsToCliString(dimensions []*cloudwatch.Dimension) (output string) {
	for _, dim := range dimensions {
		output = output + "Name=" + *dim.Name + ",Value=" + *dim.Value
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
		log.Warningf("Unable to get metric statistics due to %v", err)
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
		log.Warningf("Unable to get metric data due to %v", err)
		return nil
	}
	return &resp
}

// https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/aws-services-cloudwatch-metrics.html
func getNamespace(service string) (string, error) {
	var ns string
	var ok bool

	namespaces := map[string]string{
		"alb":                   "AWS/ApplicationELB",
		"apigateway":            "AWS/ApiGateway",
		"appsync":               "AWS/AppSync",
		"asg":                   "AWS/AutoScaling",
		"cf":                    "AWS/CloudFront",
		"docdb":                 "AWS/DocDB",
		"dynamodb":              "AWS/DynamoDB",
		"ebs":                   "AWS/EBS",
		"ec":                    "AWS/ElastiCache",
		"ec2":                   "AWS/EC2",
		"ec2Spot":               "AWS/EC2Spot",
		"ecs-svc":               "AWS/ECS",
		"ecs-containerinsights": "ECS/ContainerInsights",
		"efs":                   "AWS/EFS",
		"elb":                   "AWS/ELB",
		"emr":                   "AWS/ElasticMapReduce",
		"es":                    "AWS/ES",
		"firehose":              "AWS/Firehose",
		"fsx":                   "AWS/FSx",
		"gamelift":              "AWS/GameLift",
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
		"wafv2":                 "AWS/WAFV2",
	}
	if ns, ok = namespaces[service]; !ok {
		return "", errors.New("Not implemented namespace for cloudwatch metric: " + service)
	}
	return ns, nil
}

func createStaticDimensions(dimensions []dimension) (output []*cloudwatch.Dimension) {
	for _, d := range dimensions {
		output = append(output, buildDimension(d.Name, d.Value))
	}

	return output
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

func getFullMetricsList(namespace string, metric metric, clientCloudwatch cloudwatchInterface) (resp *cloudwatch.ListMetricsOutput) {
	c := clientCloudwatch.client
	filter := createListMetricsInput(nil, &namespace, &metric.Name)
	var res cloudwatch.ListMetricsOutput
	err := c.ListMetricsPages(filter,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			res.Metrics = append(res.Metrics, page.Metrics...)
			return !lastPage
		})
	cloudwatchAPICounter.Inc()
	if err != nil {
		log.Fatalf("Unable to list metrics due to %v", err)
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

	if !strings.HasSuffix(*namespace, "ApplicationELB") && !strings.HasSuffix(*namespace, "NetworkELB") {
		log.Fatal("Not implemented queryAvailableDimensions: " + *namespace)
		return nil
	}

	if strings.HasPrefix(resource, "targetgroup/") {
		dimensions = append(dimensions, buildDimension("TargetGroup", resource))
		resp := filterMetricsBasedOnDimensionsWithValues(dimensions, []*cloudwatch.Dimension{buildDimensionWithoutValue("LoadBalancer")}, fullMetricsList)
		if resp != nil {
			dimensions = getDimensionfromMetric(resp)
		}

	} else if strings.HasPrefix(resource, "loadbalancer/") || strings.HasPrefix(resource, "net/") || strings.HasPrefix(resource, "app/") {
		trimmedDimensionValue := strings.Replace(resource, "loadbalancer/", "", -1)
		dimensions = append(dimensions, buildDimension("LoadBalancer", trimmedDimensionValue))
	}
	return dimensions
}

func detectDimensionsByService(resource *tagsData, fullMetricsList *cloudwatch.ListMetricsOutput) (dimensions []*cloudwatch.Dimension) {
	resourceArn := *resource.ID
	service := *resource.Service
	if service == "ec2Spot" {
		return dimensions

	}
	arnParsed, err := arn.Parse(resourceArn)

	if err != nil && service != "tgwa" {
		log.Warningf("Unable to parse ARN (%s) on %s due to %v", resourceArn, service, err)
		return dimensions
	}

	type baseParams struct {
		Key    string
		Prefix string
	}
	baseDimension := map[string]baseParams{
		"appsync":  {Key: "GraphQLAPIId", Prefix: "apis/"},
		"asg":      {Key: "AutoScalingGroupName", Prefix: "autoScalingGroupName/"},
		"dynamodb": {Key: "TableName", Prefix: "table/"},
		"ebs":      {Key: "VolumeId", Prefix: "volume/"},
		"ec":       {Key: "CacheClusterId", Prefix: "cluster:"},
		"ec2":      {Key: "InstanceId", Prefix: "instance/"},
		"efs":      {Key: "FileSystemId", Prefix: "file-system/"},
		"elb":      {Key: "LoadBalancerName", Prefix: "loadbalancer/"},
		"emr":      {Key: "JobFlowId", Prefix: "cluster/"},
		"firehose": {Key: "DeliveryStreamName", Prefix: "deliverystream/"},
		"fsx":      {Key: "FileSystemId", Prefix: "file-system/"},
		"gamelift": {Key: "FleetId", Prefix: "fleet/"},
		"kinesis":  {Key: "StreamName", Prefix: "stream/"},
		"lambda":   {Key: "FunctionName", Prefix: "function:"},
		"ngw":      {Key: "NatGatewayId", Prefix: "natgateway/"},
		"redshift": {Key: "ClusterIdentifier", Prefix: "cluster:"},
		"r53r":     {Key: "EndpointId", Prefix: "resolver-endpoint/"},
		"s3":       {Key: "BucketName", Prefix: ""},
		"sns":      {Key: "TopicName", Prefix: ""},
		"sqs":      {Key: "QueueName", Prefix: ""},
		"tgw":      {Key: "TransitGateway", Prefix: "transit-gateway/"},
		"vpn":      {Key: "VpnId", Prefix: "vpn-connection/"},
	}
	if params, ok := baseDimension[service]; ok {
		return buildBaseDimension(arnParsed.Resource, params.Key, params.Prefix)
	}
	switch service {
	case "alb", "nlb":
		namespace, _ := getNamespace(service)
		dimensions = queryAvailableDimensions(arnParsed.Resource, &namespace, fullMetricsList)
	case "apigateway":
		// https://docs.aws.amazon.com/apigateway/latest/developerguide/arn-format-reference.html
		dimensions = buildBaseDimension(*resource.Matcher, "ApiName", "")
		gatewayType := strings.Split(arnParsed.Resource, "/")[1]
		switch gatewayType {
		case "restapis", "apis":
			// /stages/stage-name
			stageRegex := regexp.MustCompile(`stages/(\S+)`)
			stageMatches := stageRegex.FindStringSubmatch(arnParsed.Resource)
			if len(stageMatches) > 0 {
				dimensions = append(dimensions, buildDimension("Stage", stageMatches[1]))
			}
			// /resources/resource-id
			resourceRegex := regexp.MustCompile(`resources/(\S+)`)
			resourceMatches := resourceRegex.FindStringSubmatch(arnParsed.Resource)
			if len(resourceMatches) > 0 {
				dimensions = append(dimensions, buildDimension("Resources", resourceMatches[1]))
			}
			// /methods/http-method
			// only for restapis
			if gatewayType == "restapis" {
				methodRegex := regexp.MustCompile(`methods/(\S+)`)
				methodMatches := methodRegex.FindStringSubmatch(arnParsed.Resource)
				if len(methodMatches) > 0 {
					dimensions = append(dimensions, buildDimension("Method", methodMatches[1]))
				}
			}
		}
	case "cf":
		dimensions = buildBaseDimension(arnParsed.Resource, "DistributionId", "distribution/")
		dimensions = append(dimensions, buildDimension("Region", "Global"))
	case "docdb":
		if strings.HasPrefix(arnParsed.Resource, "cluster:") {
			dimensions = buildBaseDimension(arnParsed.Resource, "DBClusterIdentifier", "cluster:")
		} else {
			dimensions = buildBaseDimension(arnParsed.Resource, "DBInstanceIdentifier", "db:")
		}
	case "ecs-svc", "ecs-containerinsights":
		parsedResource := strings.Split(arnParsed.Resource, "/")
		if parsedResource[0] == "service" {
			dimensions = append(dimensions, buildDimension("ClusterName", parsedResource[1]), buildDimension("ServiceName", parsedResource[2]))
		}
		if parsedResource[0] == "cluster" {
			dimensions = append(dimensions, buildDimension("ClusterName", parsedResource[1]))
		}
	case "es":
		dimensions = buildBaseDimension(arnParsed.Resource, "DomainName", "domain/")
		dimensions = append(dimensions, buildDimension("ClientId", arnParsed.AccountID))
	case "rds":
		if strings.HasPrefix(arnParsed.Resource, "cluster:") {
			dimensions = buildBaseDimension(arnParsed.Resource, "DBClusterIdentifier", "cluster:")
		} else {
			dimensions = buildBaseDimension(arnParsed.Resource, "DBInstanceIdentifier", "db:")
		}
	case "sfn":
		// The value of StateMachineArn returned is the Name, not the ARN
		// We are setting the value to the ARN in order to correlate dimensions with metric values
		// (StateMachineArn will be set back to the name later, once all the filtering is complete)
		// https://docs.aws.amazon.com/step-functions/latest/dg/procedure-cw-metrics.html
		dimensions = append(dimensions, buildDimension("StateMachineArn", resourceArn))
	case "tgwa":
		parsedResource := strings.Split(resourceArn, "/")
		dimensions = append(dimensions, buildDimension("TransitGateway", parsedResource[0]), buildDimension("TransitGatewayAttachment", parsedResource[1]))
	case "kafka":
		cluster := strings.Split(arnParsed.Resource, "/")[1]
		dimensions = append(dimensions, buildDimension("Cluster Name", cluster))
	case "wafv2":
		aclId := strings.Split(resourceArn, "/")[2]
		dimensions = append(dimensions, buildDimension("WebACL", aclId))
	default:
		log.Fatal("Not implemented cloudwatch metric: " + service)
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

	if *serviceName == "alb" || *serviceName == "nlb" {
		var albSuffix, tgSuffix string
		for _, dimension := range dimensions {
			if *dimension.Name == "TargetGroup" {
				tgSuffix = "tg"
			}
			if *dimension.Name == "LoadBalancer" {
				albSuffix = *serviceName
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
		log.Warningf("Unable to parse ARN (%s) due to %v", resourceArn, err)
		return ""
	}
	parsedResource := strings.Split(arnParsed.Resource, ":")
	return parsedResource[1]
}

func createPrometheusLabels(cwd *cloudwatchData) map[string]string {
	labels := make(map[string]string)
	labels["name"] = *cwd.ID
	labels["region"] = *cwd.Region

	// Inject the sfn name back as a label
	switch *cwd.Service {
	case "sfn":
		labels["dimension_"+promStringTag("StateMachineArn")] = getStateMachineNameFromArn(*cwd.ID)
	}

	for _, dimension := range cwd.Dimensions {
		labels["dimension_"+promStringTag(*dimension.Name)] = *dimension.Value
	}

	for _, label := range cwd.CustomTags {
		labels["custom_tag_"+promStringTag(label.Key)] = label.Value
	}
	for _, tag := range cwd.Tags {
		labels["tag_"+promStringTag(tag.Key)] = tag.Value
	}

	return labels
}

func recordLabelsForMetric(metricName string, promLabels map[string]string) {
	var workingLabelsCopy []string
	if _, ok := labelMap[metricName]; ok {
		workingLabelsCopy = append(workingLabelsCopy, labelMap[metricName]...)
	}

	for k, _ := range promLabels {
		workingLabelsCopy = append(workingLabelsCopy, k)
	}
	sort.Strings(workingLabelsCopy)
	j := 0
	for i := 1; i < len(workingLabelsCopy); i++ {
		if workingLabelsCopy[j] == workingLabelsCopy[i] {
			continue
		}
		j++
		workingLabelsCopy[j] = workingLabelsCopy[i]
	}
	labelMap[metricName] = workingLabelsCopy[:j+1]
}

func ensureLabelConsistencyForMetrics(metrics []*PrometheusMetric) []*PrometheusMetric {
	var updatedMetrics []*PrometheusMetric

	for _, prometheusMetric := range metrics {
		metricName := prometheusMetric.name
		metricLabels := prometheusMetric.labels

		consistentMetricLabels := make(map[string]string)

		for _, recordedLabel := range labelMap[*metricName] {
			if value, ok := metricLabels[recordedLabel]; ok {
				consistentMetricLabels[recordedLabel] = value
			} else {
				consistentMetricLabels[recordedLabel] = ""
			}
		}
		prometheusMetric.labels = consistentMetricLabels
		updatedMetrics = append(updatedMetrics, prometheusMetric)
	}
	return updatedMetrics
}

func sortByTimestamp(datapoints []*cloudwatch.Datapoint) []*cloudwatch.Datapoint {
	sort.Slice(datapoints, func(i, j int) bool {
		jTimestamp := *datapoints[j].Timestamp
		return datapoints[i].Timestamp.After(jTimestamp)
	})
	return datapoints
}

func getDatapoint(cwd *cloudwatchData, statistic string) (*float64, time.Time) {
	if cwd.GetMetricDataPoint != nil {
		return cwd.GetMetricDataPoint, *cwd.GetMetricDataTimestamps
	}
	var averageDataPoints []*cloudwatch.Datapoint

	// sorting by timestamps so we can consistently export the most updated datapoint
	// assuming Timestamp field in cloudwatch.Datapoint struct is never nil
	for _, datapoint := range sortByTimestamp(cwd.Points) {
		switch {
		case statistic == "Maximum":
			if datapoint.Maximum != nil {
				return datapoint.Maximum, *datapoint.Timestamp
			}
		case statistic == "Minimum":
			if datapoint.Minimum != nil {
				return datapoint.Minimum, *datapoint.Timestamp
			}
		case statistic == "Sum":
			if datapoint.Sum != nil {
				return datapoint.Sum, *datapoint.Timestamp
			}
		case statistic == "SampleCount":
			if datapoint.SampleCount != nil {
				return datapoint.SampleCount, *datapoint.Timestamp
			}
		case statistic == "Average":
			if datapoint.Average != nil {
				averageDataPoints = append(averageDataPoints, datapoint)
			}
		case percentile.MatchString(statistic):
			if data, ok := datapoint.ExtendedStatistics[statistic]; ok {
				return data, *datapoint.Timestamp
			}
		default:
			log.Fatal("Not implemented statistics: " + statistic)
		}
	}

	if len(averageDataPoints) > 0 {
		var total float64
		var timestamp time.Time

		for _, p := range averageDataPoints {
			if p.Timestamp.After(timestamp) {
				timestamp = *p.Timestamp
			}
			total += *p.Average
		}
		average := total / float64(len(averageDataPoints))
		return &average, timestamp
	}
	return nil, time.Time{}
}

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			includeTimestamp := *c.AddCloudwatchTimestamp
			exportedDatapoint, timestamp := getDatapoint(c, statistic)
			if exportedDatapoint == nil {
				var nan float64 = math.NaN()
				exportedDatapoint = &nan
				includeTimestamp = false
				if *c.NilToZero {
					var zero float64 = 0
					exportedDatapoint = &zero
				}
			}
			serviceName := fixServiceName(c.Service, c.Dimensions)
			name := "aws_" + serviceName + "_" + strings.ToLower(promString(*c.Metric)) + "_" + strings.ToLower(promString(statistic))
			if exportedDatapoint != nil {

				promLabels := createPrometheusLabels(c)
				recordLabelsForMetric(name, promLabels)
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
