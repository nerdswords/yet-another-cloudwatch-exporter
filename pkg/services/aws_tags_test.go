package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

func Test_FilterThroughTags(t *testing.T) {
	testCases := []struct {
		testName     string
		resourceTags []model.Tag
		filterTags   []model.Tag
		result       bool
	}{
		{
			testName: "exactly matching tags",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			result: true,
		},
		{
			testName: "unmatching tags",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{
				{
					Key:   "k2",
					Value: "v2",
				},
			},
			result: false,
		},
		{
			testName: "resource has more tags",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "v2",
				},
			},
			filterTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			result: true,
		},
		{
			testName: "filter has more tags",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{
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
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{
				{
					Key:   "k2",
					Value: "v1",
				},
			},
			result: false,
		},
		{
			testName: "unmatching tag value",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v2",
				},
			},
			result: false,
		},
		{
			testName:     "resource without tags",
			resourceTags: []model.Tag{},
			filterTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v2",
				},
			},
			result: false,
		},
		{
			testName: "empty filter tags",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{},
			result:     true,
		},
		{
			testName: "filter with value regex",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			filterTags: []model.Tag{
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
		resourceTags []model.Tag
		exportedTags config.ExportedTagsOnMetrics
		result       []model.Tag
	}{
		{
			testName: "empty exported tag",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: config.ExportedTagsOnMetrics{},
			result:       []model.Tag{},
		},
		{
			testName: "single exported tag",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: config.ExportedTagsOnMetrics{
				"AWS/Service": []string{"k1"},
			},
			result: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
		},
		{
			testName: "multiple exported tags",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: config.ExportedTagsOnMetrics{
				"AWS/Service": []string{"k1", "k2"},
			},
			result: []model.Tag{
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
			resourceTags: []model.Tag{},
			exportedTags: config.ExportedTagsOnMetrics{
				"AWS/Service": []string{"k1"},
			},
			result: []model.Tag{
				{
					Key:   "k1",
					Value: "",
				},
			},
		},
		{
			testName: "empty exported tags for service",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: config.ExportedTagsOnMetrics{
				"AWS/Service": []string{},
			},
			result: []model.Tag{},
		},
		{
			testName: "unmatching service",
			resourceTags: []model.Tag{
				{
					Key:   "k1",
					Value: "v1",
				},
			},
			exportedTags: config.ExportedTagsOnMetrics{
				"AWS/Service_unknown": []string{"k1"},
			},
			result: []model.Tag{},
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

func Test_MigrateTagsToPrometheus(t *testing.T) {
	resources := []*TaggedResource{{
		ARN:       "aws::arn",
		Namespace: "AWS/Service",
		Region:    "us-east-1",
		Tags: []model.Tag{
			{
				Key:   "Name",
				Value: "tag_Value",
			},
		},
	}}

	prometheusMetricName := "aws_service_info"
	var metricValue float64
	expected := []*promutil.PrometheusMetric{{
		Name: &prometheusMetricName,
		Labels: map[string]string{
			"name":     "aws::arn",
			"tag_Name": "tag_Value",
		},
		Value: &metricValue,
	}}

	actual := MigrateTagsToPrometheus(resources, false, logging.NewNopLogger())

	require.Equal(t, expected, actual)
}
