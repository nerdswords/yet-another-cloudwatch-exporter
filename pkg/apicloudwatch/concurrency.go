package apicloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
)

type MaxConcurrencyClient struct {
	client *Client
	sem    chan struct{}
}

func NewWithMaxConcurrency(client *Client, maxConcurrency int) *MaxConcurrencyClient {
	return &MaxConcurrencyClient{
		client: client,
		sem:    make(chan struct{}, maxConcurrency),
	}
}

func (c MaxConcurrencyClient) GetMetricStatistics(ctx context.Context, filter *cloudwatch.GetMetricStatisticsInput) []*cloudwatch.Datapoint {
	c.sem <- struct{}{}
	res := c.client.GetMetricStatistics(ctx, filter)
	<-c.sem
	return res
}

func (c MaxConcurrencyClient) GetMetricData(ctx context.Context, filter *cloudwatch.GetMetricDataInput) *cloudwatch.GetMetricDataOutput {
	c.sem <- struct{}{}
	res := c.client.GetMetricData(ctx, filter)
	<-c.sem
	return res
}

func (c MaxConcurrencyClient) ListMetrics(ctx context.Context, namespace string, metric *config.Metric) (*cloudwatch.ListMetricsOutput, error) {
	c.sem <- struct{}{}
	res, err := c.client.ListMetrics(ctx, namespace, metric)
	<-c.sem
	return res, err
}
