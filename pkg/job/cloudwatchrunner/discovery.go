package cloudwatchrunner

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/listmetrics"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type DiscoveryJob struct {
	Job       model.DiscoveryJob
	Resources []*model.TaggedResource
}

func (d DiscoveryJob) Namespace() string {
	return d.Job.Type
}

func (d DiscoveryJob) CustomTags() []model.Tag {
	return d.Job.CustomTags
}

func (d DiscoveryJob) listMetricsParams() listmetrics.ProcessingParams {
	return listmetrics.ProcessingParams{
		Namespace:                 d.Job.Type,
		Metrics:                   d.Job.Metrics,
		RecentlyActiveOnly:        d.Job.RecentlyActiveOnly,
		DimensionNameRequirements: d.Job.DimensionNameRequirements,
	}
}

func (d DiscoveryJob) resourceEnrichment() ResourceEnrichment {
	// TODO add implementation in followup
	return nil
}
