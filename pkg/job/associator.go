package job

import (
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// valueToResource contains the mapping of, given a set of dimensions, the values of each to a resource. For example,
// if the dimensions for AWS/ECS `ClusterName` and `ServiceName` are considered, this mapping will contain for each set
// of `some_cluster, some_service` the corresponding ECS service resource.
type valueToResource map[string]*model.TaggedResource

// metricsToResourceAssociator contains for set of dimensions, the matched values and resources. Each set of dimensions
// is expressed as a concatenation of their names, order lexicographically, and using a separator in-between.
type metricsToResourceAssociator map[string]valueToResource

// match represents a dimension name and its value that were extracted from a discovered resource ARN.
type match struct {
	name, value string
}

const separator = byte('#')

// encodeMatches encodes a list of matches in two strings. One describing the dimension name of every match result, and
// one describing the value. The order of these is consistent, in order to be able to pair them with match
// es generated from a cloudwatch.Metric. For example, given the matches for dimensions `ClusterName=cluster` and
// `ServiceName=service`, this will produce the encodings `ClusterName#ServiceName` and `cluster#service`.
func encodeMatches(ms []match) (string, string) {
	var dimensionsBuilder, valuesBuilder strings.Builder
	// first, sort all matches
	sort.Slice(ms, func(i, j int) bool {
		// order lexicographically
		return ms[i].name < ms[j].name
	})
	// encode all dimensions and values, concatenating them with a separator
	for i, m := range ms {
		// write separators only before adding a new name/value to keep size minimal
		if i > 0 {
			dimensionsBuilder.WriteByte(separator)
			valuesBuilder.WriteByte(separator)
		}
		dimensionsBuilder.WriteString(m.name)
		valuesBuilder.WriteString(m.value)
	}
	return dimensionsBuilder.String(), valuesBuilder.String()
}

// newMetricsToResourceAssociator creates a new metricsToResourceAssociator given a set of dimensions regexs that can extract
// dimensions from a resource ARN, and a set of resources from which to extract.
func newMetricsToResourceAssociator(dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource) metricsToResourceAssociator {
	asocciator := make(map[string]valueToResource)
	for _, resource := range resources {
		resourceMatches := []match{}

		for _, dimensionRegexp := range dimensionRegexps {
			names := dimensionRegexp.SubexpNames()
			if dimensionRegexp.Match([]byte(resource.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(resource.ARN)
				for nameIdx, value := range dimensionMatch {
					// avoid using whole match group
					if nameIdx != 0 {
						resourceMatches = append(resourceMatches, match{names[nameIdx], value})
					}
				}
			}
		}

		encodedDimensions, encodedValues := encodeMatches(resourceMatches)
		if _, ok := asocciator[encodedDimensions]; !ok {
			asocciator[encodedDimensions] = make(valueToResource)
		}
		asocciator[encodedDimensions][encodedValues] = resource
	}

	return asocciator
}

// associateMetricsToResources finds for a cloudwatch.Metric, the resource that matches the better. The match is performed
// by taking every dimension in the metric, and producing its encodings as described in encodeMatches. Since the associator
// now for each dimension set, the mapping from values to resources, a match can be found.
func (asoc metricsToResourceAssociator) associateMetricsToResources(cwMetric *cloudwatch.Metric) (*model.TaggedResource, bool) {
	matches := make([]match, len(cwMetric.Dimensions))
	for i, dim := range cwMetric.Dimensions {
		matches[i].name = *dim.Name
		matches[i].value = *dim.Value
	}
	encodedDimensions, encodedValues := encodeMatches(matches)
	// if the dimension set of which we are looking a resource doesn't exist, return nil but avoid skipping the metric we
	// are matching. This is the default logic, and it will associate with a generic resource
	if _, ok := asoc[encodedDimensions]; !ok {
		return nil, false
	}

	// the dimension set exists in the associator, so there needs to be a match in order for the metrics to be used
	if matchedResource, ok := asoc[encodedDimensions][encodedValues]; ok {
		return matchedResource, false
	}
	return nil, true
}

// TODO: remove this, keeping below for documenting every branch

// associateMetricsToResources finds for a cloudwatch.Metrics, the resource that matches the better. If no match is found,
// nil is returned. Also, there's some conditions in which the metric shouldn't be considered, and that is dictated by the
// skip return value.
func (asoc metricsToResourceAssociator) xxxassociateMetricsToResources(cwMetric *cloudwatch.Metric) (r *model.TaggedResource, skip bool) {
	alreadyFound := false
	for _, dimension := range cwMetric.Dimensions {
		if dimensionFilterValues, ok := asoc[*dimension.Name]; ok {
			// If we are here, there is at least one discovered resource that has the dimension we are testing, therefore,
			// there should be a match in order for us to care about this metric
			if d, ok := dimensionFilterValues[*dimension.Value]; !ok {
				// If there was already a resource match, and there's more dimensions that don't match, keep the discovered resource
				if !alreadyFound {
					// If we are here, it means that alreadyFound == false => this is the first dimension we are testing
					// and there's no discovered resource with the dimension value. Avoid scraping this metric, since it
					// doesn't match any discovered resource
					skip = true
				}
				break
			} else { //nolint:revive
				alreadyFound = true
				r = d
			}
		}
	}
	// If there were no dimensions, or none of the dimensions was involved in the discovered resources, don't skip the metrics
	// but return a nil resource
	return r, skip
}
