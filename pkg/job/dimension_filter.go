package job

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"regexp"
	"strings"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// dimensionsFilter keeps an inverted index from dimension pairs, that is a name and value, to a resource
// that matches that pair.
type dimensionsFilter map[string]valuesToResource

// valuesToResource keeps a mapping scoped under one dimension name, and tracks which value of that dimension
// maps to a specific AWS resource.
type valuesToResource map[string]*model.TaggedResource

// matches finds if there is a resource that matches the given dimension.
func (fv dimensionsFilter) matches(name, value string) (res *model.TaggedResource, ok bool) {
	if dimensionFilterValues, ok := fv[name]; ok {
		if matchingResource, ok := dimensionFilterValues[value]; ok {
			return matchingResource, true
		}
	}
	return
}

// filterAndFindMatchingResource finds, if there is, a resource that matches the dimension that the given metric posses.
func (fv dimensionsFilter) filterAndFindMatchingResource(metric *cloudwatch.Metric) (res *model.TaggedResource, skip bool) {
	alreadyFound := false
	for _, dimension := range metric.Dimensions {
		if matchingResource, ok := fv.matches(*dimension.Name, *dimension.Value); !ok {
			if !alreadyFound {
				skip = true
			}
			break
		} else {
			alreadyFound = true
			res = matchingResource
		}
	}
	return
}

// buildDimensionsFilter creates a dimensionFilter from a given set of regexs that look for dimensions inside resource
// ARNs, and a set of resource whose values to extract from.
func buildDimensionsFilter(dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource) dimensionsFilter {
	filter := make(map[string]valuesToResource)
	// what does this thing do?
	for _, dimensionRegexp := range dimensionRegexps {
		names := dimensionRegexp.SubexpNames()
		// First, initialize dimensionsFilter with each named capture group in each DimensionFilter
		for i, dimensionName := range names {
			if i != 0 {
				names[i] = strings.ReplaceAll(dimensionName, "_", " ")
				if _, ok := filter[names[i]]; !ok {
					filter[names[i]] = make(valuesToResource)
				}
			}
		}
		// For each resource, extract from the ARN the matching DimensionFilter, and keep a mapping
		// between:
		//
		//   DimensionFilter.CaptureGroup.Name -> CaptureGroup.Value -> Resource
		//
		// For example, for "AWS/AppSync", the mapping will contain:
		//
		//   GraphQLAPIId -> some_id -> resource that has "some_id" assigned in the GraphQLAPIId ARN pattern
		for _, r := range resources {
			if dimensionRegexp.Match([]byte(r.ARN)) {
				dimensionMatch := dimensionRegexp.FindStringSubmatch(r.ARN)
				for i, value := range dimensionMatch {
					if i != 0 {
						filter[names[i]][value] = r
					}
				}
			}
		}
	}
	return filter
}
