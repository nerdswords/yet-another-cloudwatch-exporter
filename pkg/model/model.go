package model

import (
	"time"

	"github.com/grafana/regexp"
)

const (
	DefaultPeriodSeconds = int64(300)
	DefaultLengthSeconds = int64(300)
	DefaultDelaySeconds  = int64(300)
)

type ExportedTagsOnMetrics map[string][]string

type LabelSet map[string]struct{}

type Tag struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type Dimension struct {
	Name  string
	Value string
}

type Metric struct {
	// The dimensions for the metric.
	Dimensions []*Dimension
	MetricName string
	Namespace  string
}

type Datapoint struct {
	// The average of the metric values that correspond to the data point.
	Average *float64

	// The percentile statistic for the data point.
	ExtendedStatistics map[string]*float64

	// The maximum metric value for the data point.
	Maximum *float64

	// The minimum metric value for the data point.
	Minimum *float64

	// The number of metric values that contributed to the aggregate value of this
	// data point.
	SampleCount *float64

	// The sum of the metric values for the data point.
	Sum *float64

	// The time stamp used for the data point.
	Timestamp *time.Time
}

// CloudwatchData is an internal representation of a CloudWatch
// metric with attached data points, metric and resource information.
type CloudwatchData struct {
	ID                      *string
	MetricID                *string
	Metric                  *string
	Namespace               *string
	Statistics              []string
	Points                  []*Datapoint
	GetMetricDataPoint      *float64
	GetMetricDataTimestamps time.Time
	NilToZero               *bool
	AddCloudwatchTimestamp  *bool
	CustomTags              []Tag
	Tags                    []Tag
	Dimensions              []*Dimension
	Region                  *string
	AccountID               *string
	Period                  int64
}

// TaggedResource is an AWS resource with tags
type TaggedResource struct {
	// ARN is the unique AWS ARN (Amazon Resource Name) of the resource
	ARN string

	// Namespace identifies the resource type (e.g. EC2)
	Namespace string

	// Region is the AWS regions that the resource belongs to
	Region string

	// Tags is a set of tags associated to the resource
	Tags []Tag
}

// filterThroughTags returns true if all filterTags match
// with tags of the TaggedResource, returns false otherwise.
func (r TaggedResource) FilterThroughTags(filterTags []Tag) bool {
	if len(filterTags) == 0 {
		return true
	}

	tagMatches := 0

	for _, resourceTag := range r.Tags {
		for _, filterTag := range filterTags {
			if resourceTag.Key == filterTag.Key {
				r, _ := regexp.Compile(filterTag.Value)
				if r.MatchString(resourceTag.Value) {
					tagMatches++
				}
			}
		}
	}

	return tagMatches == len(filterTags)
}

// MetricTags returns a list of tags built from the tags of
// TaggedResource, if there's a definition for its namespace
// in tagsOnMetrics.
//
// Returned tags have as key the key from tagsOnMetrics, and
// as value the value from the corresponding tag of the resource,
// if it exists (otherwise an empty string).
func (r TaggedResource) MetricTags(tagsOnMetrics ExportedTagsOnMetrics) []Tag {
	wantedTags, ok := tagsOnMetrics[r.Namespace]
	if !ok {
		return []Tag{}
	}

	tags := make([]Tag, 0, len(wantedTags))
	for _, tagName := range wantedTags {
		tag := Tag{
			Key: tagName,
		}
		for _, resourceTag := range r.Tags {
			if resourceTag.Key == tagName {
				tag.Value = resourceTag.Value
				break
			}
		}

		// Always add the tag, even if it's empty, to ensure the same labels are present on all metrics for a single service
		tags = append(tags, tag)
	}
	return tags
}
