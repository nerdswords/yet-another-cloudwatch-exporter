package model

import (
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
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

type CloudwatchData struct {
	ID                      *string
	MetricID                *string
	Metric                  *string
	Namespace               *string
	Statistics              []string
	Points                  []*cloudwatch.Datapoint
	GetMetricDataPoint      *float64
	GetMetricDataTimestamps *time.Time
	NilToZero               *bool
	AddCloudwatchTimestamp  *bool
	CustomTags              []Tag
	Tags                    []Tag
	Dimensions              []*cloudwatch.Dimension
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
	tags := make([]Tag, 0)
	for _, tagName := range tagsOnMetrics[r.Namespace] {
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
