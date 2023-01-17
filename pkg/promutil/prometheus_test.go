package promutil

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
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "duplicate metric",
			input: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "duplicate metric, multiple labels",
			input: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1", "label2": "value2"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label2": "value2", "label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1", "label2": "value2"},
				},
			},
		},
		{
			name: "metric with different labels",
			input: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label2": "value2"},
				},
			},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label2": "value2"},
				},
			},
		},
		{
			name: "two metrics",
			input: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "two metrics with different labels",
			input: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label2": "value2"},
				},
			},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label2": "value2"},
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

func TestPromStringTag(t *testing.T) {
	testCases := []struct {
		name        string
		label       string
		toSnakeCase bool
		ok          bool
		out         string
	}{
		{
			name:        "valid",
			label:       "labelName",
			toSnakeCase: false,
			ok:          true,
			out:         "labelName",
		},
		{
			name:        "valid, convert to snake case",
			label:       "labelName",
			toSnakeCase: true,
			ok:          true,
			out:         "label_name",
		},
		{
			name:        "valid (snake case)",
			label:       "label_name",
			toSnakeCase: false,
			ok:          true,
			out:         "label_name",
		},
		{
			name:        "valid (snake case) unchanged",
			label:       "label_name",
			toSnakeCase: true,
			ok:          true,
			out:         "label_name",
		},
		{
			name:        "invalid chars",
			label:       "invalidChars@$",
			toSnakeCase: false,
			ok:          false,
			out:         "",
		},
		{
			name:        "invalid chars, convert to snake case",
			label:       "invalidChars@$",
			toSnakeCase: true,
			ok:          false,
			out:         "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok, out := PromStringTag(tc.label, tc.toSnakeCase)
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.out, out)
			}
		})
	}
}
