package exporter

import (
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

var labelMap = make(map[string][]string)

func createGetMetricStatisticsInput(dimensions []*cloudwatch.Dimension, namespace *string, metric *Metric) (output *cloudwatch.GetMetricStatisticsInput) {
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

	now = time.Now()

	startTime := now.Add(-(time.Duration(length) + time.Duration(delay)) * time.Second)
	endTime := now.Add(-time.Duration(delay) * time.Second)

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
		output = output + "Name=" + *dim.Name + ",Value=" + *dim.Value + " "
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

	if log.IsLevelEnabled(log.DebugLevel) {
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

	if log.IsLevelEnabled(log.DebugLevel) {
		log.Println(resp)
	}

	if err != nil {
		log.Warningf("Unable to get metric data due to %v", err)
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

func getFullMetricsList(namespace string, metric *Metric, clientCloudwatch cloudwatchInterface) (resp *cloudwatch.ListMetricsOutput) {
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

func getFilteredMetricDatas(region string, accountId *string, namespace string, customTags []Tag, tagsOnMetrics exportedTagsOnMetrics, dimensionRegexps []*string, resources []*taggedResource, metricsList []*cloudwatch.Metric, m *Metric) (getMetricsData []cloudwatchData) {
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
		r := &taggedResource{
			ARN:       "global",
			Namespace: namespace,
		}
		for _, dimension := range cwMetric.Dimensions {
			if dimensionFilterValues, ok := dimensionsFilter[*dimension.Name]; ok {
				if d, ok := dimensionFilterValues[*dimension.Value]; !ok {
					skip = true
					break
				} else {
					r = d
				}
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

func recordLabelsForMetric(metricName string, promLabels map[string]string) {
	var workingLabelsCopy []string
	if _, ok := labelMap[metricName]; ok {
		workingLabelsCopy = append(workingLabelsCopy, labelMap[metricName]...)
	}

	for k := range promLabels {
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

func migrateCloudwatchToPrometheus(cwd []*cloudwatchData, labelsSnakeCase bool) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			var includeTimestamp bool
			if c.AddCloudwatchTimestamp != nil {
				includeTimestamp = *c.AddCloudwatchTimestamp
			}
			exportedDatapoint, timestamp := getDatapoint(c, statistic)
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
