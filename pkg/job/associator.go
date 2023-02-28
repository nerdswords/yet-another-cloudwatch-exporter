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

// associator contains for set of dimensions, the matched values and resources. Each set of dimensions
// is expressed as a concatenation of their names, order lexicographically, and using a separator in-between.
type associator struct {
	associations   map[string]valueToResource
	seenDimensions stringSet
}

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
func newMetricsToResourceAssociator(dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource) *associator {
	assoc := make(map[string]valueToResource)
	seenDimensions := make(stringSet)
	for _, resource := range resources {
		resourceMatches := []match{}

		for _, dimensionRegexp := range dimensionRegexps {
			names := dimensionRegexp.SubexpNames()
			if dimensionRegexp.Match([]byte(resource.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(resource.ARN)
				for nameIdx, value := range dimensionMatch {
					// avoid using whole match group
					if nameIdx != 0 {
						dimensionName := names[nameIdx]
						resourceMatches = append(resourceMatches, match{dimensionName, value})
						seenDimensions.add(dimensionName)
					}
				}
			}
		}

		encodedDimensions, encodedValues := encodeMatches(resourceMatches)
		if _, ok := assoc[encodedDimensions]; !ok {
			assoc[encodedDimensions] = make(valueToResource)
		}
		assoc[encodedDimensions][encodedValues] = resource
	}

	return &associator{
		associations:   assoc,
		seenDimensions: seenDimensions,
	}
}

// associateMetricsToResources finds for a cloudwatch.Metric, the resource that matches the better. The match is performed
// by taking every dimension in the metric, and producing its encodings as described in encodeMatches. Since the associator
// now for each dimension set, the mapping from values to resources, a match can be found.
func (a *associator) associateMetricsToResources(cwMetric *cloudwatch.Metric) (*model.TaggedResource, bool) {
	matches := []match{}
	metricDimensions := make(stringSet)
	for _, dim := range cwMetric.Dimensions {
		metricDimensions.add(*dim.Name)
	}
	intersect := a.seenDimensions.intersect(metricDimensions)
	for _, dim := range cwMetric.Dimensions {
		// only consider dimensions that are both in the ones seen by the associator, and the current metric. This gives
		// the algorithm the best effort matching
		if intersect.contains(*dim.Name) {
			matches = append(matches, match{
				name:  *dim.Name,
				value: *dim.Value,
			})
		}
	}
	encodedDimensions, encodedValues := encodeMatches(matches)
	// if the dimension set of which we are looking a resource doesn't exist, return nil but avoid skipping the metric we
	// are matching. This is the default logic, and it will associate with a generic resource
	if _, ok := a.associations[encodedDimensions]; !ok {
		return nil, false
	}

	// the dimension set exists in the associator, so there needs to be a match in order for the metrics to be used
	if matchedResource, ok := a.associations[encodedDimensions][encodedValues]; ok {
		return matchedResource, false
	}
	return nil, true
}

const presentByte = byte('#')

// stringSet is a simple set implementation, with string values, that allows intersection operations.
type stringSet map[string]byte

// add adds an element to the string set. It mutates the base object.
func (ss stringSet) add(key string) {
	ss[key] = presentByte
}

func (ss stringSet) contains(key string) bool {
	if _, ok := ss[key]; ok {
		return true
	}
	return false
}

// intersect creates a new stringSet that contains the set intersection between the base object, and the argument stringSet.
func (ss stringSet) intersect(other stringSet) stringSet {
	intersection := make(stringSet)
	for k, _ := range ss {
		if _, ok := other[k]; ok {
			intersection[k] = presentByte
		}
	}
	return intersection
}
