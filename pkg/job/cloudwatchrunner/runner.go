package cloudwatchrunner

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/listmetrics"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/resourcemetadata"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type ResourceEnrichment interface {
	Create(logger logging.Logger) resourcemetadata.MetricResourceEnricher
}

type Job interface {
	Namespace() string
	CustomTags() []model.Tag
	listMetricsParams() listmetrics.ProcessingParams
	resourceEnrichment() ResourceEnrichment
}
