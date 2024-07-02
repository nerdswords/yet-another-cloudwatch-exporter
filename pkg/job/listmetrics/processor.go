package listmetrics

import "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"

type ProcessingParams struct {
	Namespace                 string
	Metrics                   []*model.MetricConfig
	RecentlyActiveOnly        bool
	DimensionNameRequirements []string
}
