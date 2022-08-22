package exporter

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

var percentile = regexp.MustCompile(`^p(\d{1,2}(\.\d{0,2})?|100)$`)

const timeFormat = "2006-01-02T15:04:05.999999-07:00"

type cloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
	logger Logger
}

type cloudwatchData struct {
	ID                      *string
	MetricID                *string
	Metric                  *string
	Namespace               *string
	Statistics              []string
	Points                  []*cloudwatch.Datapoint
	GetMetricDataPoint      *float64
	GetMetricDataTimestamps *time.Time
	NilToZero               *bool
	AddCloudwatchTimestamp  *bool
	CustomTags              []Tag
	Tags                    []Tag
	Dimensions              []*cloudwatch.Dimension
	Region                  *string
	AccountId               *string
	Period                  int64
}

func createGetMetricStatisticsInput(dimensions []*cloudwatch.Dimension, namespace *string, metric *Metric, logger Logger) (output *cloudwatch.GetMetricStatisticsInput) {
	period := metric.Period
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

	logger.Debug("CLI helper - " +
		"aws cloudwatch get-metric-statistics" +
		" --metric-name " + metric.Name +
		" --dimensions " + dimensionsToCliString(dimensions) +
		" --namespace " + *namespace +
		" --statistics " + *statistics[0] +
		" --period " + strconv.FormatInt(period, 10) +
		" --start-time " + startTime.Format(time.RFC3339) +
		" --end-time " + endTime.Format(time.RFC3339))

	logger.Debug("createGetMetricStatisticsInput", "output", *output)
	return output
}

func findGetMetricDataById(getMetricDatas []cloudwatchData, value string) (cloudwatchData, error) {
	var g cloudwatchData
	for _, getMetricData := range getMetricDatas {
		if *getMetricData.MetricID == value {
			return getMetricData, nil
		}
	}
	return g, fmt.Errorf("metric with id %s not found", value)
}

func createGetMetricDataInput(getMetricData []cloudwatchData, namespace *string, length int64, delay int64, configuredRoundingPeriod *int64, logger Logger) (output *cloudwatch.GetMetricDataInput) {
	var metricsDataQuery []*cloudwatch.MetricDataQuery
	roundingPeriod := defaultPeriodSeconds
	for _, data := range getMetricData {
		if data.Period < roundingPeriod {
			roundingPeriod = data.Period
		}
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

	if configuredRoundingPeriod != nil {
		roundingPeriod = *configuredRoundingPeriod
	}

	startTime, endTime := determineGetMetricDataWindow(
		TimeClock{},
		time.Duration(roundingPeriod)*time.Second,
		time.Duration(length)*time.Second,
		time.Duration(delay)*time.Second)
	logger.Debug("GetMetricData Window", "start_time", startTime.Format(timeFormat), "end_time", endTime.Format(timeFormat))

	dataPointOrder := "TimestampDescending"
	output = &cloudwatch.GetMetricDataInput{
		EndTime:           &endTime,
		StartTime:         &startTime,
		MetricDataQueries: metricsDataQuery,
		ScanBy:            &dataPointOrder,
	}

	return output
}

// Clock small interface which allows for stubbing the time.Now() function for unit testing
type Clock interface {
	Now() time.Time
}

// TimeClock implementation of Clock interface which delegates to Go's Time package
type TimeClock struct{}

func (tc TimeClock) Now() time.Time {
	return time.Now()
}

// determineGetMetricDataWindow computes the start and end time for the GetMetricData request to AWS
// Always uses the wall clock time as starting point for calculations to ensure that
// a variety of exporter configurations will work reliably.
func determineGetMetricDataWindow(clock Clock, roundingPeriod time.Duration, length time.Duration, delay time.Duration) (time.Time, time.Time) {
	now := clock.Now()
	if roundingPeriod > 0 {
		// Round down the time to a factor of the period - rounding is recommended by AWS:
		// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_GetMetricData.html#API_GetMetricData_RequestParameters
		now = now.Add(-roundingPeriod / 2).Round(roundingPeriod)
	}

	startTime := now.Add(-(length + delay))
	endTime := now.Add(-delay)
	return startTime, endTime
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
		output = output + "Name=" + *dim.Name + ",Value=" + *dim.Value + " "
	}
	return output
}

func (iface cloudwatchInterface) get(ctx context.Context, filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint {
	c := iface.client

	iface.logger.Debug("GetMetricStatistics", "input", filter)

	resp, err := c.GetMetricStatisticsWithContext(ctx, filter)

	iface.logger.Debug("GetMetricStatistics", "output", resp)

	cloudwatchAPICounter.Inc()
	cloudwatchGetMetricStatisticsAPICounter.Inc()

	if err != nil {
		iface.logger.Error(err, "Failed to get metric statistics")
		return nil
	}

	return resp.Datapoints
}

func (iface cloudwatchInterface) getMetricData(ctx context.Context, filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput {
	c := iface.client

	var resp cloudwatch.GetMetricDataOutput

	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("GetMetricData", "input", filter)
	}

	// Using the paged version of the function
	err := c.GetMetricDataPagesWithContext(ctx, filter,
		func(page *cloudwatch.GetMetricDataOutput, lastPage bool) bool {
			cloudwatchAPICounter.Inc()
			cloudwatchGetMetricDataAPICounter.Inc()
			resp.MetricDataResults = append(resp.MetricDataResults, page.MetricDataResults...)
			return !lastPage
		})

	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("GetMetricData", "output", resp)
	}

	if err != nil {
		iface.logger.Error(err, "Failed to get metric data")
		return nil
	}
	return &resp
}

func createStaticDimensions(dimensions []Dimension) (output []*cloudwatch.Dimension) {
	for _, d := range dimensions {
		d := d
		output = append(output, &cloudwatch.Dimension{
			Name:  &d.Name,
			Value: &d.Value,
		})
	}

	return output
}

func getFullMetricsList(ctx context.Context, namespace string, metric *Metric, clientCloudwatch cloudwatchInterface) (resp *cloudwatch.ListMetricsOutput, err error) {
	c := clientCloudwatch.client
	filter := createListMetricsInput(nil, &namespace, &metric.Name)
	var res cloudwatch.ListMetricsOutput
	err = c.ListMetricsPagesWithContext(ctx, filter,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			res.Metrics = append(res.Metrics, page.Metrics...)
			return !lastPage
		})
	if err != nil {
		cloudwatchAPIErrorCounter.Inc()
		return nil, err
	}
	cloudwatchAPICounter.Inc()
	return &res, nil
}

func getFilteredMetricDatas(region string, accountId *string, namespace string, customTags []Tag, tagsOnMetrics exportedTagsOnMetrics, dimensionRegexps []*string, resources []*taggedResource, metricsList []*cloudwatch.Metric, dimensionNameList []string, m *Metric) (getMetricsData []cloudwatchData) {
	type filterValues map[string]*taggedResource
	dimensionsFilter := make(map[string]filterValues)
	for _, dr := range dimensionRegexps {
		dimensionRegexp := regexp.MustCompile(*dr)
		names := dimensionRegexp.SubexpNames()
		for i, dimensionName := range names {
			if i != 0 {
				names[i] = strings.ReplaceAll(dimensionName, "_", " ")
				if _, ok := dimensionsFilter[names[i]]; !ok {
					dimensionsFilter[names[i]] = make(filterValues)
				}
			}
		}
		for _, r := range resources {
			if dimensionRegexp.Match([]byte(r.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(r.ARN)
				for i, value := range dimensionMatch {
					if i != 0 {
						dimensionsFilter[names[i]][value] = r
					}
				}
			}
		}
	}
	for _, cwMetric := range metricsList {
		skip := false
		alreadyFound := false
		r := &taggedResource{
			ARN:       "global",
			Namespace: namespace,
		}
		if len(dimensionNameList) > 0 && !metricDimensionsMatchNames(cwMetric, dimensionNameList) {
			continue
		}

		/**
		This loop takes a list of dimensions for an individual metric returned from AWS ResourceGroupsTaggingApi#GetResources.
		It filters those dimensions against a user-supplied list of dimensions by name and value, and if they match,
		adds the metric to a list of metrics to have its values retrieved.
		*/
		for _, dimension := range cwMetric.Dimensions {
			if dimensionFilterValues, ok := dimensionsFilter[*dimension.Name]; ok {
				if d, ok := dimensionFilterValues[*dimension.Value]; !ok {
					if !alreadyFound {
						skip = true
					}
					break
				} else {
					alreadyFound = true
					r = d
				}
			} else {
				skip = true
				break
			}
		}

		if !skip {
			for _, stats := range m.Statistics {
				id := fmt.Sprintf("id_%d", rand.Int())
				metricTags := r.metricTags(tagsOnMetrics)
				getMetricsData = append(getMetricsData, cloudwatchData{
					ID:                     &r.ARN,
					MetricID:               &id,
					Metric:                 &m.Name,
					Namespace:              &namespace,
					Statistics:             []string{stats},
					NilToZero:              m.NilToZero,
					AddCloudwatchTimestamp: m.AddCloudwatchTimestamp,
					Tags:                   metricTags,
					CustomTags:             customTags,
					Dimensions:             cwMetric.Dimensions,
					Region:                 &region,
					AccountId:              accountId,
					Period:                 int64(m.Period),
				})
			}
		}
	}
	return getMetricsData
}

func metricDimensionsMatchNames(metric *cloudwatch.Metric, dimensionNameRequirements []string) bool {
	if len(dimensionNameRequirements) != len(metric.Dimensions) {
		return false
	}
	for _, dimension := range metric.Dimensions {
		foundMatch := false
		for _, dimensionName := range dimensionNameRequirements {
			if *dimension.Name == dimensionName {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return false
		}
	}
	return true
}

func createPrometheusLabels(cwd *cloudwatchData, labelsSnakeCase bool) map[string]string {
	labels := make(map[string]string)
	labels["name"] = *cwd.ID
	labels["region"] = *cwd.Region
	labels["account_id"] = *cwd.AccountId

	// Inject the sfn name back as a label
	for _, dimension := range cwd.Dimensions {
		labels["dimension_"+promStringTag(*dimension.Name, labelsSnakeCase)] = *dimension.Value
	}

	for _, label := range cwd.CustomTags {
		labels["custom_tag_"+promStringTag(label.Key, labelsSnakeCase)] = label.Value
	}
	for _, tag := range cwd.Tags {
		labels["tag_"+promStringTag(tag.Key, labelsSnakeCase)] = tag.Value
	}

	return labels
}

// recordLabelsForMetric adds any missing labels from promLabels in to the LabelSet for the metric name and returns
// the updated observedMetricLabels
func recordLabelsForMetric(metricName string, promLabels map[string]string, observedMetricLabels map[string]LabelSet) map[string]LabelSet {
	if _, ok := observedMetricLabels[metricName]; !ok {
		observedMetricLabels[metricName] = make(LabelSet)
	}
	for label := range promLabels {
		if _, ok := observedMetricLabels[metricName][label]; !ok {
			observedMetricLabels[metricName][label] = struct{}{}
		}
	}

	return observedMetricLabels
}

// ensureLabelConsistencyForMetrics ensures that every metric has the same set of labels based on the data
// in observedMetricLabels. Prometheus requires that all metrics with the same name have the same set of labels
func ensureLabelConsistencyForMetrics(metrics []*PrometheusMetric, observedMetricLabels map[string]LabelSet) []*PrometheusMetric {
	for _, prometheusMetric := range metrics {
		for observedLabel := range observedMetricLabels[*prometheusMetric.name] {
			if _, ok := prometheusMetric.labels[observedLabel]; !ok {
				prometheusMetric.labels[observedLabel] = ""
			}
		}
	}
	return metrics
}

func sortByTimestamp(datapoints []*cloudwatch.Datapoint) []*cloudwatch.Datapoint {
	sort.Slice(datapoints, func(i, j int) bool {
		jTimestamp := *datapoints[j].Timestamp
		return datapoints[i].Timestamp.After(jTimestamp)
	})
	return datapoints
}

func getDatapoint(cwd *cloudwatchData, statistic string) (*float64, time.Time, error) {
	if cwd.GetMetricDataPoint != nil {
		return cwd.GetMetricDataPoint, *cwd.GetMetricDataTimestamps, nil
	}
	var averageDataPoints []*cloudwatch.Datapoint

	// sorting by timestamps so we can consistently export the most updated datapoint
	// assuming Timestamp field in cloudwatch.Datapoint struct is never nil
	for _, datapoint := range sortByTimestamp(cwd.Points) {
		switch {
		case statistic == "Maximum":
			if datapoint.Maximum != nil {
				return datapoint.Maximum, *datapoint.Timestamp, nil
			}
		case statistic == "Minimum":
			if datapoint.Minimum != nil {
				return datapoint.Minimum, *datapoint.Timestamp, nil
			}
		case statistic == "Sum":
			if datapoint.Sum != nil {
				return datapoint.Sum, *datapoint.Timestamp, nil
			}
		case statistic == "SampleCount":
			if datapoint.SampleCount != nil {
				return datapoint.SampleCount, *datapoint.Timestamp, nil
			}
		case statistic == "Average":
			if datapoint.Average != nil {
				averageDataPoints = append(averageDataPoints, datapoint)
			}
		case percentile.MatchString(statistic):
			if data, ok := datapoint.ExtendedStatistics[statistic]; ok {
				return data, *datapoint.Timestamp, nil
			}
		default:
			return nil, time.Time{}, fmt.Errorf("invalid statistic requested on metric %s: %s", *cwd.Metric, statistic)
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
		return &average, timestamp, nil
	}
	return nil, time.Time{}, nil
}

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData, labelsSnakeCase bool, observedMetricLabels map[string]LabelSet) ([]*PrometheusMetric, map[string]LabelSet, error) {
	output := make([]*PrometheusMetric, 0)

	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			var includeTimestamp bool
			if c.AddCloudwatchTimestamp != nil {
				includeTimestamp = *c.AddCloudwatchTimestamp
			}
			exportedDatapoint, timestamp, err := getDatapoint(c, statistic)
			if err != nil {
				return nil, nil, err
			}
			if exportedDatapoint == nil && (c.AddCloudwatchTimestamp == nil || !*c.AddCloudwatchTimestamp) {
				var nan float64 = math.NaN()
				exportedDatapoint = &nan
				includeTimestamp = false
				if *c.NilToZero {
					var zero float64 = 0
					exportedDatapoint = &zero
				}
			}
			promNs := strings.ToLower(*c.Namespace)
			if !strings.HasPrefix(promNs, "aws") {
				promNs = "aws_" + promNs
			}
			name := promString(promNs) + "_" + strings.ToLower(promString(*c.Metric)) + "_" + strings.ToLower(promString(statistic))
			if exportedDatapoint != nil {
				promLabels := createPrometheusLabels(c, labelsSnakeCase)
				observedMetricLabels = recordLabelsForMetric(name, promLabels, observedMetricLabels)
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

	return output, observedMetricLabels, nil
}
