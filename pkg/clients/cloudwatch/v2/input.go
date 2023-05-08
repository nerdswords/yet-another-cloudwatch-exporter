package v2

import (
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

const timeFormat = "2006-01-02T15:04:05.999999-07:00"

// Clock small interface which allows for stubbing the time.Now() function for unit testing
type Clock interface {
	Now() time.Time
}

// TimeClock implementation of Clock interface which delegates to Go's Time package
type TimeClock struct{}

func (tc TimeClock) Now() time.Time {
	return time.Now()
}

func createGetMetricDataInput(logger logging.Logger, getMetricData []*model.CloudwatchData, namespace *string, length int64, delay int64, configuredRoundingPeriod *int64) *cloudwatch.GetMetricDataInput {
	metricsDataQuery := make([]types.MetricDataQuery, 0, len(getMetricData))
	roundingPeriod := model.DefaultPeriodSeconds
	for _, data := range getMetricData {
		if data.Period < roundingPeriod {
			roundingPeriod = data.Period
		}
		metricStat := &types.MetricStat{
			Metric: &types.Metric{
				Dimensions: toCloudWatchDimensions(data.Dimensions),
				MetricName: data.Metric,
				Namespace:  namespace,
			},
			Period: aws.Int32(int32(data.Period)),
			Stat:   &data.Statistics[0],
		}
		metricsDataQuery = append(metricsDataQuery, types.MetricDataQuery{
			Id:         data.MetricID,
			MetricStat: metricStat,
			ReturnData: aws.Bool(true),
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

	return &cloudwatch.GetMetricDataInput{
		EndTime:           &endTime,
		StartTime:         &startTime,
		MetricDataQueries: metricsDataQuery,
		ScanBy:            "TimestampDescending",
	}
}

func toCloudWatchDimensions(dimensions []*model.Dimension) []types.Dimension {
	cwDim := make([]types.Dimension, 0, len(dimensions))
	for _, dim := range dimensions {
		cwDim = append(cwDim, types.Dimension{
			Name:  &dim.Name,
			Value: &dim.Value,
		})
	}
	return cwDim
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

func createGetMetricStatisticsInput(logger logging.Logger, dimensions []*model.Dimension, namespace *string, metric *config.Metric) *cloudwatch.GetMetricStatisticsInput {
	period := metric.Period
	length := metric.Length
	delay := metric.Delay
	endTime := time.Now().Add(-time.Duration(delay) * time.Second)
	startTime := time.Now().Add(-(time.Duration(length) + time.Duration(delay)) * time.Second)

	var statistics []types.Statistic
	var extendedStatistics []string
	for _, statistic := range metric.Statistics {
		if promutil.Percentile.MatchString(statistic) {
			extendedStatistics = append(extendedStatistics, statistic)
		} else {
			statistics = append(statistics, types.Statistic(statistic))
		}
	}

	output := &cloudwatch.GetMetricStatisticsInput{
		Dimensions:         toCloudWatchDimensions(dimensions),
		Namespace:          namespace,
		StartTime:          &startTime,
		EndTime:            &endTime,
		Period:             aws.Int32(int32(period)),
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
			" --statistics " + string(statistics[0]) +
			" --period " + strconv.FormatInt(period, 10) +
			" --start-time " + startTime.Format(time.RFC3339) +
			" --end-time " + endTime.Format(time.RFC3339))

		logger.Debug("createGetMetricStatisticsInput", "output", *output)
	}

	return output
}

func dimensionsToCliString(dimensions []*model.Dimension) string {
	out := strings.Builder{}
	for _, dim := range dimensions {
		out.WriteString("Name=")
		out.WriteString(dim.Name)
		out.WriteString(",Value=")
		out.WriteString(dim.Value)
		out.WriteString(" ")
	}
	return out.String()
}
