package cloudwatch

import (
	"context"
	"time"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type Client interface {
	// ListMetrics returns the list of metrics and dimensions for a given namespace
	// and metric name. Results pagination is handled automatically: the caller can
	// optionally pass a non-nil func in order to handle results pages.
	ListMetrics(ctx context.Context, namespace string, metric *config.Metric, recentlyActiveOnly bool, fn func(page []*model.Metric)) ([]*model.Metric, error)

	// GetMetricData returns the output of the GetMetricData CloudWatch API.
	// Results pagination is handled automatically.
	GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64, addHistoricalMetrics bool) []*MetricDataResult

	// GetMetricStatistics returns the output of the GetMetricStatistics CloudWatch API.
	GetMetricStatistics(ctx context.Context, logger logging.Logger, dimensions []*model.Dimension, namespace string, metric *config.Metric) []*model.Datapoint
}

type MetricDataResult struct {
	ID        *string
	Datapoint *float64
	Timestamp *time.Time
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

func (c limitedConcurrencyClient) GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64, addHistoricalMetrics bool) []*MetricDataResult {
	c.sem <- struct{}{}
	res := c.client.GetMetricData(ctx, logger, getMetricData, namespace, length, delay, configuredRoundingPeriod, addHistoricalMetrics)
	<-c.sem
	return res
}

func (c limitedConcurrencyClient) ListMetrics(ctx context.Context, namespace string, metric *config.Metric, recentlyActiveOnly bool, fn func(page []*model.Metric)) ([]*model.Metric, error) {
	c.sem <- struct{}{}
	res, err := c.client.ListMetrics(ctx, namespace, metric, recentlyActiveOnly, fn)
	<-c.sem
	return res, err
}
