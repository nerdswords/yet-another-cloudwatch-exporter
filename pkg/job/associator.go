package job

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"regexp"
	"strings"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type valueToResource map[string]*model.TaggedResource
type metricsToResourceAssociator map[string]valueToResource

func newMetricsToResourceAssociator(dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource) metricsToResourceAssociator {
	dimensionsFilter := make(map[string]valueToResource)
	for _, dimensionRegexp := range dimensionRegexps {
		names := dimensionRegexp.SubexpNames()
		for i, dimensionName := range names {
			if i != 0 {
				names[i] = strings.ReplaceAll(dimensionName, "_", " ")
				if _, ok := dimensionsFilter[names[i]]; !ok {
					dimensionsFilter[names[i]] = make(valueToResource)
				}
			}
		}
		for _, r := range resources {
			if dimensionRegexp.Match([]byte(r.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(r.ARN)
				for i, value := range dimensionMatch {
					if i != 0 {
						dimensionsFilter[names[i]][value] = r
					}
				}
			}
		}
	}
	return dimensionsFilter
}

func (asoc metricsToResourceAssociator) associateMetricsToResources(namespace string, cwMetric *cloudwatch.Metric) (r *model.TaggedResource, skip bool) {
	alreadyFound := false
	for _, dimension := range cwMetric.Dimensions {
		if dimensionFilterValues, ok := asoc[*dimension.Name]; ok {
			if d, ok := dimensionFilterValues[*dimension.Value]; !ok {
				if !alreadyFound {
					skip = true
				}
				break
			} else {
				alreadyFound = true
				r = d
			}
		}
	}
	return r, skip
}
