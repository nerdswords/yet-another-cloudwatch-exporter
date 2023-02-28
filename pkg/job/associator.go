package job

import (
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// valueToResource contains the mapping of, given a dimension, values of it to a resource. For example, if the  dimension
// for which this valueToResource has been creates is InstanceId, it will contain for a given EC2 instance ID the resource
// that matches it.
type valueToResource map[string]*model.TaggedResource

// metricsToResourceAssociator contains for each dimension, the matched values and resources.
type metricsToResourceAssociator map[string]valueToResource

type match struct {
	name, value string
}

const separator = byte('#')

func encodeMatches(ms []match) (string, string) {
	var dimensionsBuilder, valuesBuilder strings.Builder
	// first, sort all matches
	sort.Slice(ms, func(i, j int) bool {
		// check if i < j
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
	dimensionsFilter := make(map[string]valueToResource)
	for _, r := range resources {
		matches := []match{}

		for _, dimensionRegexp := range dimensionRegexps {
			names := dimensionRegexp.SubexpNames()
			if dimensionRegexp.Match([]byte(r.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(r.ARN)
				for i, value := range dimensionMatch {
					// avoid using whole match group
					if i != 0 {
						matches = append(matches, match{names[i], value})
					}
				}
			}
		}

		dims, vals := encodeMatches(matches)

		if _, ok := dimensionsFilter[dims]; !ok {
			dimensionsFilter[dims] = make(valueToResource)
		}

		dimensionsFilter[dims][vals] = r
	}

	return dimensionsFilter
}

func (asoc metricsToResourceAssociator) associateMetricsToResources(cwMetric *cloudwatch.Metric) (*model.TaggedResource, bool) {
	matches := make([]match, len(cwMetric.Dimensions))
	for i, dim := range cwMetric.Dimensions {
		matches[i].name = *dim.Name
		matches[i].value = *dim.Value
	}
	dims, vals := encodeMatches(matches)
	// if the dimension set of which we are looking a resource doesn't exists, return nil but avoid skipping
	// this is the default logic
	if _, ok := asoc[dims]; !ok {
		return nil, false
	}

	// the dimension set exists in the associator, so there needs to be a match in order for the metrics to be used
	if res, ok := asoc[dims][vals]; ok {
		return res, false
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
