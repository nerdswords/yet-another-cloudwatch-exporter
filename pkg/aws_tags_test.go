package exporter

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Test_FilterThroughTags(t *testing.T) {
	testCases := []struct {
		testName     string
		resourceTags []Tag
		filterTags   []Tag
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
			filterTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
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
			filterTags: []Tag{
				{
					Key:   "k2",
					Value: "v2",
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
			filterTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
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
			filterTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "v2",
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
			filterTags: []Tag{
				{
					Key:   "k2",
					Value: "v1",
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
			filterTags: []Tag{
				{
					Key:   "k1",
					Value: "v2",
				},
			},
			result: false,
		},
		{
			testName:     "resource without tags",
			resourceTags: []Tag{},
			filterTags: []Tag{
				{
					Key:   "k1",
					Value: "v2",
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
			filterTags: []Tag{},
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
			filterTags: []Tag{
				{
					Key:   "k1",
					Value: "v.*",
				},
			},
			result: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := taggedResource{
				ARN:       "aws::arn",
				Namespace: "AWS/Service",
				Region:    "us-east-1",
				Tags:      tc.resourceTags,
			}

			require.Equal(t, tc.result, res.filterThroughTags(tc.filterTags))
		})
	}
}

func Test_MetricTags(t *testing.T) {
	testCases := []struct {
		testName     string
		resourceTags []Tag
		exportedTags exportedTagsOnMetrics
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
			exportedTags: exportedTagsOnMetrics{},
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
			exportedTags: exportedTagsOnMetrics{
				"AWS/Service": []string{"k1"},
			},
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
			exportedTags: exportedTagsOnMetrics{
				"AWS/Service": []string{"k1", "k2"},
			},
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
			exportedTags: exportedTagsOnMetrics{
				"AWS/Service": []string{"k1"},
			},
			result: []Tag{
				{
					Key:   "k1",
					Value: "",
				},
			},
		},
		{
			testName: "empty exported tags for service",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: exportedTagsOnMetrics{
				"AWS/Service": []string{},
			},
			result: []Tag{},
		},
		{
			testName: "unmatching service",
			resourceTags: []Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: exportedTagsOnMetrics{
				"AWS/Service_unknown": []string{"k1"},
			},
			result: []Tag{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			res := taggedResource{
				ARN:       "aws::arn",
				Namespace: "AWS/Service",
				Region:    "us-east-1",
				Tags:      tc.resourceTags,
			}

			require.Equal(t, tc.result, res.metricTags(tc.exportedTags))
		})
	}
}

func Test_MigrateTagsToPrometheus(t *testing.T) {
	resources := []*taggedResource{{
		ARN:       "aws::arn",
		Namespace: "AWS/Service",
		Region:    "us-east-1",
		Tags: []Tag{
			{
				Key:   "Name",
				Value: "tag_Value",
			},
		},
	}}

	prometheusMetricName := "aws_service_info"
	var metricValue float64 = 0
	expected := []*PrometheusMetric{{
		name: &prometheusMetricName,
		labels: map[string]string{
			"name":     "aws::arn",
			"tag_Name": "tag_Value",
		},
		value: &metricValue,
	}}

	actual := migrateTagsToPrometheus(resources, false, NewLogrusLogger(log.StandardLogger()))

	require.Equal(t, expected, actual)
}
