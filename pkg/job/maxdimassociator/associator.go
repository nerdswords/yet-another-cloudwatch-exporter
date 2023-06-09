package maxdimassociator

import (
	"strings"

	"github.com/grafana/regexp"
	prom_model "github.com/prometheus/common/model"
	"golang.org/x/exp/slices"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

// Associator implements a "best effort" algorithm to automatically map the output
// of the ListMetrics API to the list of resources retrieved from the Tagging API.
// The core logic is based on a manually maintained list of regexes that extract
// dimensions names from ARNs (see services.go). YACE supports auto-discovery for
// those AWS namespaces where the ARN regexes are correctly defined.
type Associator struct {
	// mappings is a slice of dimensions-based mappings, one for each regex of a given namespace
	mappings []*dimensionsRegexpMapping
}

type dimensionsRegexpMapping struct {
	// dimensions is a slice of dimensions names in a regex (normally 1 name is enough
	// to identify the resource type by its ARN, sometimes 2 or 3 dimensions names are
	// needed to identify sub-resources)
	dimensions []string

	// dimensionsMapping maps the set of dimensions (names and values) to a resource.
	// Dimensions names and values are encoded as a uint64 fingerprint.
	dimensionsMapping map[uint64]*model.TaggedResource
}

// NewAssociator builds all mappings for the given dimensions regexps and list of resources.
func NewAssociator(dimensionRegexps []*regexp.Regexp, resources []*model.TaggedResource) Associator {
	assoc := Associator{mappings: []*dimensionsRegexpMapping{}}

	// Keep track of resources that have already been mapped.
	// Each resource will be matched against at most one regex.
	// TODO(cristian): use a more memory-efficient data structure
	mappedResources := make([]bool, len(resources))

	for _, regex := range dimensionRegexps {
		m := &dimensionsRegexpMapping{dimensionsMapping: map[uint64]*model.TaggedResource{}}

		names := regex.SubexpNames()
		dimensionNames := make([]string, 0, len(names)-1)
		for i := 1; i < len(names); i++ { // skip first name, it's always empty string
			// in the regex names we use underscores where AWS dimensions have spaces
			names[i] = strings.ReplaceAll(names[i], "_", " ")
			dimensionNames = append(dimensionNames, names[i])
		}
		m.dimensions = dimensionNames

		for idx, r := range resources {
			if mappedResources[idx] {
				continue
			}

			match := regex.FindStringSubmatch(r.ARN)
			if match == nil {
				continue
			}

			labels := make(map[string]string, len(match))
			for i := 1; i < len(match); i++ {
				labels[names[i]] = match[i]
			}
			signature := prom_model.LabelsToSignature(labels)
			m.dimensionsMapping[signature] = r
			mappedResources[idx] = true
		}

		assoc.mappings = append(assoc.mappings, m)
	}

	// sort all mappings by decreasing number of dimensions names
	// (this is essential so that during matching we try to find the metric
	// with the most specific set of dimensions)
	slices.SortFunc(assoc.mappings, func(a, b *dimensionsRegexpMapping) bool {
		return len(a.dimensions) >= len(b.dimensions)
	})

	return assoc
}

// AssociateMetricToResource finds the resource that corresponds to the given set of dimensions
// names and values of a metric. The guess is based on the mapping built from dimensions regexps.
// In case a map can't be found, the second return parameter indicates whether the metric should be
// ignored or not.
func (assoc Associator) AssociateMetricToResource(cwMetric *model.Metric) (*model.TaggedResource, bool) {
	if len(cwMetric.Dimensions) == 0 {
		// Do not skip the metric (create a "global" metric)
		return nil, false
	}

	dimensions := make([]string, 0, len(cwMetric.Dimensions))
	for _, dimension := range cwMetric.Dimensions {
		dimensions = append(dimensions, dimension.Name)
	}

	// Find the mapping which contains the most
	// (but not necessarily all) the metric's dimensions names.
	// Mappings are sorted by decreasing number of dimensions names.
	var regexpMapping *dimensionsRegexpMapping
	for _, m := range assoc.mappings {
		// if no dimensions are mapped for a mapping attempt the next mapping
		if len(m.dimensionsMapping) == 0 {
			continue
		}
		if containsAll(dimensions, m.dimensions) {
			regexpMapping = m
			break
		}
	}

	if regexpMapping == nil {
		// if no mapping is found, it means the ListMetrics API response
		// did not contain any subset of the metric's dimensions names.
		// Do not skip the metric though (create a "global" metric).
		return nil, false
	}

	// A mapping has been found. The metric has all (and possibly more)
	// the dimensions computed for the mapping. Pick only exactly
	// the dimensions of the mapping to build a labels signature.
	labels := buildLabelsMap(cwMetric, regexpMapping)
	signature := prom_model.LabelsToSignature(labels)

	if resource, ok := regexpMapping.dimensionsMapping[signature]; ok {
		return resource, false
	}

	// if there's no mapping entry for this resource, skip it
	return nil, true
}

// buildLabelsMap returns a map of labels names and values.
// For some namespaces, values might need to be modified in order
// to match the dimension value extracted from ARN.
func buildLabelsMap(cwMetric *model.Metric, regexpMapping *dimensionsRegexpMapping) map[string]string {
	labels := make(map[string]string, len(cwMetric.Dimensions))
	for _, rDimension := range regexpMapping.dimensions {
		for _, mDimension := range cwMetric.Dimensions {
			name := mDimension.Name
			value := mDimension.Value

			// AmazonMQ is special - for active/standby ActiveMQ brokers,
			// the value of the "Broker" dimension contains a number suffix
			// that is not part of the resource ARN
			if cwMetric.Namespace == "AWS/AmazonMQ" && name == "Broker" {
				brokerSuffix := regexp.MustCompile("-[0-9]+$")
				if brokerSuffix.MatchString(value) {
					value = brokerSuffix.ReplaceAllString(value, "")
				}
			}

			if rDimension == mDimension.Name {
				labels[name] = value
			}
		}
	}
	return labels
}

// containsAll returns true if a contains all elements of b
func containsAll(a, b []string) bool {
	for _, e := range b {
		if slices.Contains(a, e) {
			continue
		}
		return false
	}
	return true
}
