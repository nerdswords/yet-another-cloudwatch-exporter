package apitagging

import (
	"context"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
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

func (c MaxConcurrencyClient) GetResources(ctx context.Context, job *config.Job, region string) ([]*model.TaggedResource, error) {
	c.sem <- struct{}{}
	res, err := c.client.GetResources(ctx, job, region)
	<-c.sem
	return res, err
}
