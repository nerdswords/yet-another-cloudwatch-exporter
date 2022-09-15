package exporter

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestSplitString(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  "GlobalTopicCount",
			output: "Global.Topic.Count",
		},
		{
			input:  "CPUUtilization",
			output: "CPUUtilization",
		},
		{
			input:  "StatusCheckFailed_Instance",
			output: "Status.Check.Failed_Instance",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.output, splitString(tc.input))
	}
}

func TestSanitize(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  "Global.Topic.Count",
			output: "Global_Topic_Count",
		},
		{
			input:  "Status.Check.Failed_Instance",
			output: "Status_Check_Failed_Instance",
		},
		{
			input:  "IHaveA%Sign",
			output: "IHaveA_percentSign",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.output, sanitize(tc.input))
	}
}

func TestRemoveDuplicateMetrics(t *testing.T) {
	testCases := []struct {
		name   string
		input  []*PrometheusMetric
		output []*PrometheusMetric
	}{
		{
			name:   "empty",
			input:  []*PrometheusMetric{},
			output: []*PrometheusMetric{},
		},
		{
			name: "one metric",
			input: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "duplicate metric",
			input: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "duplicate metric, multiple labels",
			input: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1", "label2": "value2"},
				},
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label2": "value2", "label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1", "label2": "value2"},
				},
			},
		},
		{
			name: "metric with different labels",
			input: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label2": "value2"},
				},
			},
			output: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label2": "value2"},
				},
			},
		},
		{
			name: "two metrics",
			input: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric2"),
					labels: map[string]string{"label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric2"),
					labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "two metrics with different labels",
			input: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric2"),
					labels: map[string]string{"label2": "value2"},
				},
			},
			output: []*PrometheusMetric{
				{
					name:   aws.String("metric1"),
					labels: map[string]string{"label1": "value1"},
				},
				{
					name:   aws.String("metric2"),
					labels: map[string]string{"label2": "value2"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.output, removeDuplicatedMetrics(tc.input))
		})
	}
}
