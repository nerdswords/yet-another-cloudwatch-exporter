package cloudwatchrunner

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/listmetrics"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type CustomNamespaceJob struct {
	Job model.CustomNamespaceJob
}

func (c CustomNamespaceJob) Namespace() string {
	return c.Job.Namespace
}

func (c CustomNamespaceJob) listMetricsParams() listmetrics.ProcessingParams {
	return listmetrics.ProcessingParams{
		Namespace:                 c.Job.Namespace,
		Metrics:                   c.Job.Metrics,
		RecentlyActiveOnly:        c.Job.RecentlyActiveOnly,
		DimensionNameRequirements: c.Job.DimensionNameRequirements,
	}
}

func (c CustomNamespaceJob) CustomTags() []model.Tag {
	return c.Job.CustomTags
}

func (c CustomNamespaceJob) resourceEnrichment() ResourceEnrichment {
	// TODO add implementation in followup
	return nil
}
