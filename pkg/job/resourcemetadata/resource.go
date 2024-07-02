package resourcemetadata

import (
	"context"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type Resource struct {
	// Name is an identifiable value for the resource and is variable dependent on the match made
	//	It will be the AWS ARN (Amazon Resource Name) if a unique resource was found
	//  It will be "global" if a unique resource was not found
	//  CustomNamespaces will have the custom namespace Name
	Name string
	// Tags is a set of tags associated to the resource
	Tags []model.Tag
}

type Resources struct {
	StaticResource      *Resource
	AssociatedResources []*Resource
}

type MetricResourceEnricher interface {
	Enrich(ctx context.Context, metrics []*model.Metric) ([]*model.Metric, Resources)
}
