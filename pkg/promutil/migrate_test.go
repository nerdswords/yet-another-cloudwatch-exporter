package promutil

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestBuildNamespaceInfoMetrics(t *testing.T) {
	resources := []*model.TaggedResource{{
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

	expectedMetrics := []*PrometheusMetric{{
		Name: aws.String("aws_service_info"),
		Labels: map[string]string{
			"name":     "aws::arn",
			"tag_Name": "tag_Value",
		},
		Value: aws.Float64(0),
	}}

	expectedLabels := map[string]model.LabelSet{
		"aws_service_info": map[string]struct{}{
			"name":     {},
			"tag_Name": {},
		},
	}

	actualMetrics, actualLabels := BuildNamespaceInfoMetrics(resources, []*PrometheusMetric{}, map[string]model.LabelSet{}, false, logging.NewNopLogger())

	assert.Equal(t, expectedMetrics, actualMetrics)
	assert.Equal(t, expectedLabels, actualLabels)
}

// TestSortByTimeStamp validates that sortByTimestamp() sorts in descending order.
func TestSortByTimeStamp(t *testing.T) {
	dataPointMiddle := &model.Datapoint{
		Timestamp: aws.Time(time.Now().Add(time.Minute * 2 * -1)),
		Maximum:   aws.Float64(2),
	}

	dataPointNewest := &model.Datapoint{
		Timestamp: aws.Time(time.Now().Add(time.Minute * -1)),
		Maximum:   aws.Float64(1),
	}

	dataPointOldest := &model.Datapoint{
		Timestamp: aws.Time(time.Now().Add(time.Minute * 3 * -1)),
		Maximum:   aws.Float64(3),
	}

	cloudWatchDataPoints := []*model.Datapoint{
		dataPointMiddle,
		dataPointNewest,
		dataPointOldest,
	}

	sortedDataPoints := sortByTimestamp(cloudWatchDataPoints)

	expectedDataPoints := []*model.Datapoint{
		dataPointNewest,
		dataPointMiddle,
		dataPointOldest,
	}

	require.Equal(t, expectedDataPoints, sortedDataPoints)
}

func Test_EnsureLabelConsistencyAndRemoveDuplicates(t *testing.T) {
	testCases := []struct {
		name           string
		metrics        []*PrometheusMetric
		observedLabels map[string]model.LabelSet
		output         []*PrometheusMetric
	}{
		{
			name: "adds missing labels",
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
					Value:  aws.Float64(1.0),
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label2": "value2"},
					Value:  aws.Float64(2.0),
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{},
					Value:  aws.Float64(3.0),
				},
			},
			observedLabels: map[string]model.LabelSet{"metric1": {"label1": struct{}{}, "label2": struct{}{}, "label3": struct{}{}}},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1", "label2": "", "label3": ""},
					Value:  aws.Float64(1.0),
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "", "label3": "", "label2": "value2"},
					Value:  aws.Float64(2.0),
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "", "label2": "", "label3": ""},
					Value:  aws.Float64(3.0),
				},
			},
		},
		{
			name: "duplicate metric",
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
			observedLabels: map[string]model.LabelSet{},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
		},
		{
			name: "duplicate metric, multiple labels",
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1", "label2": "value2"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label2": "value2", "label1": "value1"},
				},
			},
			observedLabels: map[string]model.LabelSet{},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1", "label2": "value2"},
				},
			},
		},
		{
			name: "metric with different labels",
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label2": "value2"},
				},
			},
			observedLabels: map[string]model.LabelSet{},
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
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
			observedLabels: map[string]model.LabelSet{},
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
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label2": "value2"},
				},
			},
			observedLabels: map[string]model.LabelSet{},
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
		{
			name: "multiple duplicates and non-duplicates",
			metrics: []*PrometheusMetric{
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label2": "value2"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
			observedLabels: map[string]model.LabelSet{},
			output: []*PrometheusMetric{
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label2": "value2"},
				},
				{
					Name:   aws.String("metric2"),
					Labels: map[string]string{"label1": "value1"},
				},
				{
					Name:   aws.String("metric1"),
					Labels: map[string]string{"label1": "value1"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := EnsureLabelConsistencyAndRemoveDuplicates(tc.metrics, tc.observedLabels)
			require.ElementsMatch(t, tc.output, actual)
		})
	}
}
