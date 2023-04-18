package cloudwatch

import (
	"context"
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

type Client interface {
	// ListMetrics returns the list of metrics and dimensions for a given namespace
	// and metric name. Results pagination is handled automatically: the caller can
	// optionally pass a non-nil func in order to handle results pages.
	ListMetrics(ctx context.Context, namespace string, recentlyActiveOnly bool, metric *config.Metric, fn func(page []*model.Metric)) ([]*model.Metric, error)

	// GetMetricData returns the output of the GetMetricData CloudWatch API.
	// Results pagination is handled automatically.
	GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []*MetricDataResult

	// GetMetricStatistics returns the output of the GetMetricStatistics CloudWatch API.
	GetMetricStatistics(ctx context.Context, logger logging.Logger, dimensions []*model.Dimension, namespace string, metric *config.Metric) []*model.Datapoint
}

const timeFormat = "2006-01-02T15:04:05.999999-07:00"

type client struct {
	logger        logging.Logger
	cloudwatchAPI cloudwatchiface.CloudWatchAPI
}

func NewClient(logger logging.Logger, cloudwatchAPI cloudwatchiface.CloudWatchAPI) Client {
	return &client{
		logger:        logger,
		cloudwatchAPI: cloudwatchAPI,
	}
}

func GetListMetricsInput(metricName string, namespace string, recentlyActiveOnly bool) *cloudwatch.ListMetricsInput {
	if !recentlyActiveOnly {
		return &cloudwatch.ListMetricsInput{
			MetricName: aws.String(metricName),
			Namespace:  aws.String(namespace),
		}
	}
	recentActiveString := "PT3H"
	return &cloudwatch.ListMetricsInput{
		MetricName:     aws.String(metricName),
		Namespace:      aws.String(namespace),
		RecentlyActive: &recentActiveString,
	}
}

func (c client) ListMetrics(ctx context.Context, namespace string, recentlyActiveOnly bool, metric *config.Metric, fn func(page []*model.Metric)) ([]*model.Metric, error) {
	filter := GetListMetricsInput(metric.Name, namespace, recentlyActiveOnly)

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("ListMetrics", "input", input)
	}

	var metrics []*model.Metric
	err := c.cloudwatchAPI.ListMetricsPagesWithContext(ctx, filter,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			promutil.CloudwatchAPICounter.Inc()
			metricsPage := toModelMetric(page)
			if fn != nil {
				fn(metricsPage)
			} else {
				metrics = append(metrics, metricsPage...)
			}
			return !lastPage
		})
	if err != nil {
		promutil.CloudwatchAPIErrorCounter.Inc()
		c.logger.Error(err, "ListMetrics error")
		return nil, err
	}

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("ListMetrics", "output", metrics)
	}

	return metrics, nil
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

func toModelDimensions(dimensions []*cloudwatch.Dimension) []*model.Dimension {
	modelDimensions := make([]*model.Dimension, 0, len(dimensions))
	for _, dimension := range dimensions {
		modelDimension := &model.Dimension{
			Name:  *dimension.Name,
			Value: *dimension.Value,
		}
		modelDimensions = append(modelDimensions, modelDimension)
	}
	return modelDimensions
}

type MetricDataResult struct {
	ID        *string
	Datapoint *float64
	Timestamp *time.Time
}

func (c client) GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []*MetricDataResult {
	var resp cloudwatch.GetMetricDataOutput
	filter := createGetMetricDataInput(getMetricData, &namespace, length, delay, configuredRoundingPeriod, logger)
	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricData", "input", filter)
	}

	// Using the paged version of the function
	err := c.cloudwatchAPI.GetMetricDataPagesWithContext(ctx, filter,
		func(page *cloudwatch.GetMetricDataOutput, lastPage bool) bool {
			promutil.CloudwatchAPICounter.Inc()
			promutil.CloudwatchGetMetricDataAPICounter.Inc()
			resp.MetricDataResults = append(resp.MetricDataResults, page.MetricDataResults...)
			return !lastPage
		})

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricData", "output", resp)
	}

	if err != nil {
		c.logger.Error(err, "GetMetricData error")
		return nil
	}
	return toMetricDataResult(resp)
}

func toMetricDataResult(resp cloudwatch.GetMetricDataOutput) []*MetricDataResult {
	output := make([]*MetricDataResult, 0, len(resp.MetricDataResults))
	for _, metricDataResult := range resp.MetricDataResults {
		mappedResult := MetricDataResult{ID: metricDataResult.Id}
		if len(metricDataResult.Values) > 0 {
			mappedResult.Datapoint = metricDataResult.Values[0]
			mappedResult.Timestamp = metricDataResult.Timestamps[0]
		}
		output = append(output, &mappedResult)
	}
	return output
}

func (c client) GetMetricStatistics(ctx context.Context, logger logging.Logger, dimensions []*model.Dimension, namespace string, metric *config.Metric) []*model.Datapoint {
	filter := createGetMetricStatisticsInput(dimensions, &namespace, metric, logger)

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricStatistics", "input", filter)
	}

	resp, err := c.cloudwatchAPI.GetMetricStatisticsWithContext(ctx, filter)

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("GetMetricStatistics", "output", resp)
	}

	promutil.CloudwatchAPICounter.Inc()
	promutil.CloudwatchGetMetricStatisticsAPICounter.Inc()

	if err != nil {
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

type limitedConcurrencyClient struct {
	client Client
	sem    chan struct{}
}

func NewLimitedConcurrencyClient(client Client, maxConcurrency int) Client {
	return &limitedConcurrencyClient{
		client: client,
		sem:    make(chan struct{}, maxConcurrency),
	}
}

func (c limitedConcurrencyClient) GetMetricStatistics(ctx context.Context, logger logging.Logger, dimensions []*model.Dimension, namespace string, metric *config.Metric) []*model.Datapoint {
	c.sem <- struct{}{}
	res := c.client.GetMetricStatistics(ctx, logger, dimensions, namespace, metric)
	<-c.sem
	return res
}

func (c limitedConcurrencyClient) GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []*MetricDataResult {
	c.sem <- struct{}{}
	res := c.client.GetMetricData(ctx, logger, getMetricData, namespace, length, delay, configuredRoundingPeriod)
	<-c.sem
	return res
}

func (c limitedConcurrencyClient) ListMetrics(ctx context.Context, namespace string, recentlyActiveOnly bool, metric *config.Metric, fn func(page []*model.Metric)) ([]*model.Metric, error) {
	c.sem <- struct{}{}
	res, err := c.client.ListMetrics(ctx, input, fn)
	<-c.sem
	return res, err
}
