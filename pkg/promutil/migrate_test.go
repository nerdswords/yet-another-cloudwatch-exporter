package promutil

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func Test_MigrateTagsToPrometheus(t *testing.T) {
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

	prometheusMetricName := "aws_service_info"
	var metricValue float64
	expected := []*PrometheusMetric{{
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

// TestSortyByTimeStamp validates that sortByTimestamp() sorts in descending order.
func TestSortyByTimeStamp(t *testing.T) {
	dataPointMiddle := &cloudwatch.Datapoint{
		Timestamp: aws.Time(time.Now().Add(time.Minute * 2 * -1)),
		Maximum:   aws.Float64(2),
	}

	dataPointNewest := &cloudwatch.Datapoint{
		Timestamp: aws.Time(time.Now().Add(time.Minute * -1)),
		Maximum:   aws.Float64(1),
	}

	dataPointOldest := &cloudwatch.Datapoint{
		Timestamp: aws.Time(time.Now().Add(time.Minute * 3 * -1)),
		Maximum:   aws.Float64(3),
	}

	cloudWatchDataPoints := []*cloudwatch.Datapoint{
		dataPointMiddle,
		dataPointNewest,
		dataPointOldest,
	}

	sortedDataPoints := sortByTimestamp(cloudWatchDataPoints)

	expectedDataPoints := []*cloudwatch.Datapoint{
		dataPointNewest,
		dataPointMiddle,
		dataPointOldest,
	}

	require.Equal(t, expectedDataPoints, sortedDataPoints)
}

func Test_ensureLabelConsistencyForMetrics(t *testing.T) {
	value1 := 1.0
	metric1 := PrometheusMetric{
		Name:   aws.String("metric1"),
		Labels: map[string]string{"label1": "value1"},
		Value:  &value1,
	}

	value2 := 2.0
	metric2 := PrometheusMetric{
		Name:   aws.String("metric1"),
		Labels: map[string]string{"label2": "value2"},
		Value:  &value2,
	}

	value3 := 2.0
	metric3 := PrometheusMetric{
		Name:   aws.String("metric1"),
		Labels: map[string]string{},
		Value:  &value3,
	}

	metrics := []*PrometheusMetric{&metric1, &metric2, &metric3}
	result := EnsureLabelConsistencyForMetrics(metrics, map[string]model.LabelSet{"metric1": {"label1": struct{}{}, "label2": struct{}{}, "label3": struct{}{}}})

	expected := []string{"label1", "label2", "label3"}
	for _, metric := range result {
		assert.Equal(t, len(expected), len(metric.Labels))
		labels := []string{}
		for labelName := range metric.Labels {
			labels = append(labels, labelName)
		}

		assert.ElementsMatch(t, expected, labels)
	}
}
