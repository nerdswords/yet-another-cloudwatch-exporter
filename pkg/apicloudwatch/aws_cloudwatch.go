package apicloudwatch

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

const timeFormat = "2006-01-02T15:04:05.999999-07:00"

type CloudwatchInterface struct {
	client cloudwatchiface.CloudWatchAPI
	logger logging.Logger
}

func NewCloudWatchInterface(client cloudwatchiface.CloudWatchAPI, logger logging.Logger) *CloudwatchInterface {
	return &CloudwatchInterface{
		client: client,
		logger: logger,
	}
}

func CreateGetMetricStatisticsInput(dimensions []*cloudwatch.Dimension, namespace *string, metric *config.Metric, logger logging.Logger) (output *cloudwatch.GetMetricStatisticsInput) {
	period := metric.Period
	length := metric.Length
	delay := metric.Delay
	endTime := time.Now().Add(-time.Duration(delay) * time.Second)
	startTime := time.Now().Add(-(time.Duration(length) + time.Duration(delay)) * time.Second)

	var statistics []*string
	var extendedStatistics []*string
	for _, statistic := range metric.Statistics {
		if promutil.Percentile.MatchString(statistic) {
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

	if logger.IsDebugEnabled() {
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
	}

	return output
}

func CreateGetMetricDataInput(getMetricData []model.CloudwatchData, namespace *string, length int64, delay int64, configuredRoundingPeriod *int64, logger logging.Logger) (output *cloudwatch.GetMetricDataInput) {
	var metricsDataQuery []*cloudwatch.MetricDataQuery
	roundingPeriod := model.DefaultPeriodSeconds
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

	if logger.IsDebugEnabled() {
		logger.Debug("GetMetricData Window", "start_time", startTime.Format(timeFormat), "end_time", endTime.Format(timeFormat))
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

func (iface CloudwatchInterface) GetMetricStatistics(ctx context.Context, filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint {
	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("GetMetricStatistics", "input", filter)
	}

	resp, err := iface.client.GetMetricStatisticsWithContext(ctx, filter)

	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("GetMetricStatistics", "output", resp)
	}

	promutil.CloudwatchAPICounter.Inc()
	promutil.CloudwatchGetMetricStatisticsAPICounter.Inc()

	if err != nil {
		iface.logger.Error(err, "Failed to get metric statistics")
		return nil
	}

	return resp.Datapoints
}

func (iface CloudwatchInterface) GetMetricData(ctx context.Context, filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput {
	var resp cloudwatch.GetMetricDataOutput

	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("GetMetricData", "input", filter)
	}

	// Using the paged version of the function
	err := iface.client.GetMetricDataPagesWithContext(ctx, filter,
		func(page *cloudwatch.GetMetricDataOutput, lastPage bool) bool {
			promutil.CloudwatchAPICounter.Inc()
			promutil.CloudwatchGetMetricDataAPICounter.Inc()
			resp.MetricDataResults = append(resp.MetricDataResults, page.MetricDataResults...)
			return !lastPage
		})

	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("GetMetricData", "output", resp)
	}

	if err != nil {
		iface.logger.Error(err, "GetMetricData error")
		return nil
	}
	return &resp
}

func (iface CloudwatchInterface) ListMetrics(ctx context.Context, namespace string, metric *config.Metric) (*cloudwatch.ListMetricsOutput, error) {
	filter := createListMetricsInput(nil, &namespace, &metric.Name)
	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("ListMetrics", "input", filter)
	}

	var res cloudwatch.ListMetricsOutput
	err := iface.client.ListMetricsPagesWithContext(ctx, filter,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			res.Metrics = append(res.Metrics, page.Metrics...)
			return !lastPage
		})
	if err != nil {
		promutil.CloudwatchAPIErrorCounter.Inc()
		iface.logger.Error(err, "ListMetrics error")
		return nil, err
	}

	if iface.logger.IsDebugEnabled() {
		iface.logger.Debug("ListMetrics", "output", res)
	}

	promutil.CloudwatchAPICounter.Inc()
	return &res, nil
}
