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

type JobsConfig struct {
	StsRegion           string
	DiscoveryJobs       []DiscoveryJob
	StaticJobs          []StaticJob
	CustomNamespaceJobs []CustomNamespaceJob
}

type DiscoveryJob struct {
	Regions                   []string
	Type                      string
	Roles                     []Role
	SearchTags                []Tag
	CustomTags                []Tag
	DimensionNameRequirements []string
	DimensionValueFilter      []*DimensionFilter
	Metrics                   []*MetricConfig
	RoundingPeriod            *int64
	RecentlyActiveOnly        bool
	ExportedTagsOnMetrics     []string
	JobLevelMetricFields
}

type StaticJob struct {
	Name       string
	Regions    []string
	Roles      []Role
	Namespace  string
	CustomTags []Tag
	Dimensions []Dimension
	Metrics    []*MetricConfig
}

type CustomNamespaceJob struct {
	Regions                   []string
	Name                      string
	Namespace                 string
	RecentlyActiveOnly        bool
	Roles                     []Role
	Metrics                   []*MetricConfig
	CustomTags                []Tag
	DimensionNameRequirements []string
	DimensionValueFilter      []*DimensionFilter
	RoundingPeriod            *int64
	JobLevelMetricFields
}

type JobLevelMetricFields struct {
	Statistics             []string
	Period                 int64
	Length                 int64
	Delay                  int64
	NilToZero              *bool
	AddCloudwatchTimestamp *bool
}

type Role struct {
	RoleArn    string
	ExternalID string
}

type MetricConfig struct {
	Name                   string
	Statistics             []string
	Period                 int64
	Length                 int64
	Delay                  int64
	NilToZero              *bool
	AddCloudwatchTimestamp *bool
}

type LabelSet map[string]struct{}

type Tag struct {
	Key   string
	Value string
}

type Dimension struct {
	Name  string
	Value string
}
type DimensionFilter struct {
	Name  string
	Value *regexp.Regexp
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

type CloudwatchMetricResult struct {
	Context *JobContext
	Data    []*CloudwatchData
}

type JobContext struct {
	Region     string
	AccountID  string
	CustomTags []Tag
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
	Tags                    []Tag
	Dimensions              []*Dimension
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
// TaggedResource, if exportedTags is not empty.
//
// Returned tags have as key the key from exportedTags, and
// as value the value from the corresponding tag of the resource,
// if it exists (otherwise an empty string).
func (r TaggedResource) MetricTags(exportedTags []string) []Tag {
	if len(exportedTags) == 0 {
		return []Tag{}
	}

	tags := make([]Tag, 0, len(exportedTags))
	for _, tagName := range exportedTags {
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
