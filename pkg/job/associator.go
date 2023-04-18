package job

import (
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/grafana/regexp"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// valueToResource contains the mapping of, given a dimension, values of it to a resource. For example, if the  dimension
// for which this valueToResource has been creates is InstanceId, it will contain for a given EC2 instance ID the resource
// that matches it.
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
			if dimensionRegexp.MatchString(r.ARN) {
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

// AssociateMetricsToResources finds, for a given cloudwatch.Metrics, the resource that matches the better.
// If no match is found, nil is returned. Also, there are some conditions where the metric shouldn't be
// considered, and that is dictated by the skip return value.
func (asoc metricsToResourceAssociator) AssociateMetricsToResources(cwMetric *cloudwatch.Metric) (*model.TaggedResource, bool) {
	var r *model.TaggedResource
	skip := false
	alreadyFound := false
	for _, dimension := range cwMetric.Dimensions {
		if dimensionFilterValues, ok := asoc[*dimension.Name]; ok {
			if d, ok := dimensionFilterValues[*dimension.Value]; !ok {
				if !alreadyFound {
					skip = true
				}
				break
			} else { //nolint:revive
				alreadyFound = true
				r = d
			}
		}
	}
	return r, skip
}
