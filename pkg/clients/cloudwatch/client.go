package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

type Client interface {
	// ListMetrics returns the list of metrics and dimensions for a given namespace
	// and metric name. Results pagination is handled automatically: the caller can
	// optionally pass a non-nil func in order to handle results pages.
	ListMetrics(ctx context.Context, namespace string, metric *config.Metric, fn func(page *cloudwatch.ListMetricsOutput)) (*cloudwatch.ListMetricsOutput, error)

	// GetMetricData returns the output of the GetMetricData CloudWatch API.
	// Results pagination is handled automatically.
	GetMetricData(ctx context.Context, filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput

	// GetMetricStatistics returns the output of the GetMetricStatistics CloudWatch API.
	GetMetricStatistics(ctx context.Context, filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint
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

func (c client) ListMetrics(ctx context.Context, namespace string, metric *config.Metric, fn func(page *cloudwatch.ListMetricsOutput)) (*cloudwatch.ListMetricsOutput, error) {
	filter := &cloudwatch.ListMetricsInput{
		MetricName: aws.String(metric.Name),
		Namespace:  aws.String(namespace),
	}

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("ListMetrics", "input", filter)
	}

	var res cloudwatch.ListMetricsOutput
	err := c.cloudwatchAPI.ListMetricsPagesWithContext(ctx, filter,
		func(page *cloudwatch.ListMetricsOutput, lastPage bool) bool {
			if fn != nil {
				fn(page)
			} else {
				res.Metrics = append(res.Metrics, page.Metrics...)
			}
			return !lastPage
		})
	if err != nil {
		promutil.CloudwatchAPIErrorCounter.Inc()
		c.logger.Error(err, "ListMetrics error")
		return nil, err
	}

	if c.logger.IsDebugEnabled() {
		c.logger.Debug("ListMetrics", "output", res)
	}

	promutil.CloudwatchAPICounter.Inc()
	return &res, nil
}

func (c client) GetMetricData(ctx context.Context, filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput {
	var resp cloudwatch.GetMetricDataOutput

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
	return &resp
}

func (c client) GetMetricStatistics(ctx context.Context, filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint {
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

	return resp.Datapoints
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

func (c limitedConcurrencyClient) GetMetricStatistics(ctx context.Context, filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint {
	c.sem <- struct{}{}
	res := c.client.GetMetricStatistics(ctx, filter)
	<-c.sem
	return res
}

func (c limitedConcurrencyClient) GetMetricData(ctx context.Context, filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput {
	c.sem <- struct{}{}
	res := c.client.GetMetricData(ctx, filter)
	<-c.sem
	return res
}

func (c limitedConcurrencyClient) ListMetrics(ctx context.Context, namespace string, metric *config.Metric, fn func(page *cloudwatch.ListMetricsOutput)) (*cloudwatch.ListMetricsOutput, error) {
	c.sem <- struct{}{}
	res, err := c.client.ListMetrics(ctx, namespace, metric, fn)
	<-c.sem
	return res, err
}
