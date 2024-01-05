package tagging

import (
	"context"
	"errors"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type Client interface {
	GetResources(ctx context.Context, job model.DiscoveryJob, region string) ([]*model.TaggedResource, error)
}

var ErrExpectedToFindResources = errors.New("expected to discover resources but none were found")

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

func (c limitedConcurrencyClient) GetResources(ctx context.Context, job model.DiscoveryJob, region string) ([]*model.TaggedResource, error) {
	c.sem <- struct{}{}
	res, err := c.client.GetResources(ctx, job, region)
	<-c.sem
	return res, err
}
