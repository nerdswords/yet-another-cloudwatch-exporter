package job

import (
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// valueToResource contains the mapping of, given a set of dimensions, the values of each to a resource. For example,
// if the dimensions for AWS/ECS `ClusterName` and `ServiceName` are considered, this mapping will contain for each set
// of `some_cluster, some_service` the corresponding ECS service resource.
type valueToResource map[string]*model.TaggedResource

// metricsToResourceAssociator contains for each dimension, the matched values and resources.
type metricsToResourceAssociator map[string]valueToResource

// newMetricsToResourceAssociator creates a new metricsToResourceAssociator given a set of dimensions regexs that can extract
// dimensions from a resource ARN, and a set of resources from which to extract.
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

// associateMetricsToResources finds for a cloudwatch.Metrics, the resource that matches the better. If no match is found,
// nil is returned. Also, there's some conditions in which the metric shouldn't be considered, and that is dictated by the
// skip return value.
func (asoc metricsToResourceAssociator) associateMetricsToResources(cwMetric *cloudwatch.Metric) (r *model.TaggedResource, skip bool) {
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
