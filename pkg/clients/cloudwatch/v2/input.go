package v2

import (
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

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

	startTime, endTime := cloudwatch_client.DetermineGetMetricDataWindow(
		cloudwatch_client.TimeClock{},
		time.Duration(roundingPeriod)*time.Second,
		time.Duration(length)*time.Second,
		time.Duration(delay)*time.Second)

	if logger.IsDebugEnabled() {
		logger.Debug("GetMetricData Window", "start_time", startTime.Format(cloudwatch_client.TimeFormat), "end_time", endTime.Format(cloudwatch_client.TimeFormat))
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
