package v1

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

type client struct {
	logger        logging.Logger
	cloudwatchAPI cloudwatchiface.CloudWatchAPI
}

func NewClient(logger logging.Logger, cloudwatchAPI cloudwatchiface.CloudWatchAPI) cloudwatch_client.Client {
	return &client{
		logger:        logger,
		cloudwatchAPI: cloudwatchAPI,
	}
}

func (c client) ListMetrics(ctx context.Context, namespace string, metric *model.MetricConfig, recentlyActiveOnly bool, fn func(page []*model.Metric)) error {
	filter := &cloudwatch.ListMetricsInput{
		MetricName: aws.String(metric.Name),
		Namespace:  aws.String(namespace),
	}
	if recentlyActiveOnly {
		filter.RecentlyActive = aws.String("PT3H")
	}

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("ListMetrics", "input", filter)
	}

	err := c.cloudwatchAPI.ListMetricsPagesWithContext(ctx, filter, func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
		promutil.CloudwatchAPICounter.WithLabelValues("ListMetrics").Inc()

		metricsPage := toModelMetric(page)

		if c.logger.IsDebugEnabled() {
			c.logger.Debug("ListMetrics", "output", metricsPage, "last_page", lastPage)
		}

		fn(metricsPage)
		return !lastPage
	})
	if err != nil {
		promutil.CloudwatchAPIErrorCounter.WithLabelValues("ListMetrics").Inc()
		c.logger.Error(err, "ListMetrics error")
		return err
	}

	return nil
}

func toModelMetric(page *cloudwatch.ListMetricsOutput) []*model.Metric {
	modelMetrics := make([]*model.Metric, 0, len(page.Metrics))
	for _, cloudwatchMetric := range page.Metrics {
		modelMetric := &model.Metric{
			MetricName: *cloudwatchMetric.MetricName,
			Namespace:  *cloudwatchMetric.Namespace,
			Dimensions: toModelDimensions(cloudwatchMetric.Dimensions),
		}
		modelMetrics = append(modelMetrics, modelMetric)
	}
	return modelMetrics
}

func toModelDimensions(dimensions []*cloudwatch.Dimension) []model.Dimension {
	modelDimensions := make([]model.Dimension, 0, len(dimensions))
	for _, dimension := range dimensions {
		modelDimension := model.Dimension{
			Name:  *dimension.Name,
			Value: *dimension.Value,
		}
		modelDimensions = append(modelDimensions, modelDimension)
	}
	return modelDimensions
}

func (c client) GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []cloudwatch_client.MetricDataResult {
	filter := createGetMetricDataInput(getMetricData, &namespace, length, delay, configuredRoundingPeriod, logger)
	promutil.CloudwatchGetMetricDataAPIMetricsCounter.Add(float64(len(filter.MetricDataQueries)))
	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricData", "input", filter)
	}

	var resp cloudwatch.GetMetricDataOutput
	// Using the paged version of the function
	err := c.cloudwatchAPI.GetMetricDataPagesWithContext(ctx, filter,
		func(page *cloudwatch.GetMetricDataOutput, lastPage bool) bool {
			promutil.CloudwatchGetMetricDataAPICounter.Inc()
			promutil.CloudwatchAPICounter.WithLabelValues("GetMetricData").Inc()
			resp.MetricDataResults = append(resp.MetricDataResults, page.MetricDataResults...)
			return !lastPage
		})

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricData", "output", resp)
	}

	if err != nil {
		promutil.CloudwatchAPIErrorCounter.WithLabelValues("GetMetricData").Inc()
		c.logger.Error(err, "GetMetricData error")
		return nil
	}
	return toMetricDataResult(resp)
}

func toMetricDataResult(resp cloudwatch.GetMetricDataOutput) []cloudwatch_client.MetricDataResult {
	output := make([]cloudwatch_client.MetricDataResult, 0, len(resp.MetricDataResults))
	for _, metricDataResult := range resp.MetricDataResults {
		mappedResult := cloudwatch_client.MetricDataResult{ID: *metricDataResult.Id}
		if len(metricDataResult.Values) > 0 {
			mappedResult.Datapoint = metricDataResult.Values[0]
			mappedResult.Timestamp = *metricDataResult.Timestamps[0]
		}
		output = append(output, mappedResult)
	}
	return output
}

func (c client) GetMetricStatistics(ctx context.Context, logger logging.Logger, dimensions []model.Dimension, namespace string, metric *model.MetricConfig) []*model.Datapoint {
	filter := createGetMetricStatisticsInput(dimensions, &namespace, metric, logger)

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricStatistics", "input", filter)
	}

	resp, err := c.cloudwatchAPI.GetMetricStatisticsWithContext(ctx, filter)

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricStatistics", "output", resp)
	}

	promutil.CloudwatchGetMetricStatisticsAPICounter.Inc()
	promutil.CloudwatchAPICounter.WithLabelValues("GetMetricStatistics").Inc()

	if err != nil {
		promutil.CloudwatchAPIErrorCounter.WithLabelValues("GetMetricStatistics").Inc()
		c.logger.Error(err, "Failed to get metric statistics")
		return nil
	}

	return toModelDatapoints(resp.Datapoints)
}

func toModelDatapoints(cwDatapoints []*cloudwatch.Datapoint) []*model.Datapoint {
	modelDataPoints := make([]*model.Datapoint, 0, len(cwDatapoints))

	for _, cwDatapoint := range cwDatapoints {
		modelDataPoints = append(modelDataPoints, &model.Datapoint{
			Average:            cwDatapoint.Average,
			ExtendedStatistics: cwDatapoint.ExtendedStatistics,
			Maximum:            cwDatapoint.Maximum,
			Minimum:            cwDatapoint.Minimum,
			SampleCount:        cwDatapoint.SampleCount,
			Sum:                cwDatapoint.Sum,
			Timestamp:          cwDatapoint.Timestamp,
		})
	}
	return modelDataPoints
}
