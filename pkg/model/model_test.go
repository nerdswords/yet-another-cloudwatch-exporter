package model

import (
	"testing"

	"github.com/grafana/regexp"
	"github.com/stretchr/testify/require"
)

func Test_FilterThroughTags(t *testing.T) {
	testCases := []struct {
		testName     string
		resourceTags []Tag
		filterTags   []SearchTag
		result       bool
	}{
		{
			testName: "exactly matching tags",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k1",
					Value: regexp.MustCompile("v1"),
				},
			},
			result: true,
		},
		{
			testName: "unmatching tags",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k2",
					Value: regexp.MustCompile("v2"),
				},
			},
			result: false,
		},
		{
			testName: "resource has more tags",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "v2",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k1",
					Value: regexp.MustCompile("v1"),
				},
			},
			result: true,
		},
		{
			testName: "filter has more tags",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k1",
					Value: regexp.MustCompile("v1"),
				},
				{
					Key:   "k2",
					Value: regexp.MustCompile("v2"),
				},
			},
			result: false,
		},
		{
			testName: "unmatching tag key",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k2",
					Value: regexp.MustCompile("v1"),
				},
			},
			result: false,
		},
		{
			testName: "unmatching tag value",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k1",
					Value: regexp.MustCompile("v2"),
				},
			},
			result: false,
		},
		{
			testName:     "resource without tags",
			resourceTags: []Tag{},
			filterTags: []SearchTag{
				{
					Key:   "k1",
					Value: regexp.MustCompile("v2"),
				},
			},
			result: false,
		},
		{
			testName: "empty filter tags",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{},
			result:     true,
		},
		{
			testName: "filter with value regex",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []SearchTag{
				{
					Key:   "k1",
					Value: regexp.MustCompile("v.*"),
				},
			},
			result: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := TaggedResource{
				ARN:       "aws::arn",
				Namespace: "AWS/Service",
				Region:    "us-east-1",
				Tags:      tc.resourceTags,
			}
			require.Equal(t, tc.result, res.FilterThroughTags(tc.filterTags))
		})
	}
}

func Test_MetricTags(t *testing.T) {
	testCases := []struct {
		testName     string
		resourceTags []Tag
		exportedTags []string
		result       []Tag
	}{
		{
			testName: "empty exported tag",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: []string{},
			result:       []Tag{},
		},
		{
			testName: "single exported tag",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: []string{"k1"},
			result: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
		},
		{
			testName: "multiple exported tags",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: []string{"k1", "k2"},
			result: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "",
				},
			},
		},
		{
			testName:     "resource without tags",
			resourceTags: []Tag{},
			exportedTags: []string{"k1"},
			result: []Tag{
				{
					Key:   "k1",
					Value: "",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := TaggedResource{
				ARN:       "aws::arn",
				Namespace: "AWS/Service",
				Region:    "us-east-1",
				Tags:      tc.resourceTags,
			}

			require.Equal(t, tc.result, res.MetricTags(tc.exportedTags))
		})
	}
}
