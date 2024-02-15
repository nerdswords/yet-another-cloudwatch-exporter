package v2

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/stretchr/testify/require"

	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
)

func Test_toMetricDataResult(t *testing.T) {
	ts := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	type testCase struct {
		name                      string
		getMetricDataOutput       cloudwatch.GetMetricDataOutput
		expectedMetricDataResults []cloudwatch_client.MetricDataResult
	}

	testCases := []testCase{
		{
			name: "all metrics present",
			getMetricDataOutput: cloudwatch.GetMetricDataOutput{
				MetricDataResults: []types.MetricDataResult{
					{
						Id:         aws.String("metric-1"),
						Values:     []float64{1.0, 2.0, 3.0},
						Timestamps: []time.Time{ts.Add(10 * time.Minute), ts.Add(5 * time.Minute), ts},
					},
					{
						Id:         aws.String("metric-2"),
						Values:     []float64{2.0},
						Timestamps: []time.Time{ts},
					},
				},
			},
			expectedMetricDataResults: []cloudwatch_client.MetricDataResult{
				{ID: "metric-1", Datapoint: aws.Float64(1.0), Timestamp: ts.Add(10 * time.Minute)},
				{ID: "metric-2", Datapoint: aws.Float64(2.0), Timestamp: ts},
			},
		},
		{
			name: "metric with no values",
			getMetricDataOutput: cloudwatch.GetMetricDataOutput{
				MetricDataResults: []types.MetricDataResult{
					{
						Id:         aws.String("metric-1"),
						Values:     []float64{1.0, 2.0, 3.0},
						Timestamps: []time.Time{ts.Add(10 * time.Minute), ts.Add(5 * time.Minute), ts},
					},
					{
						Id:         aws.String("metric-2"),
						Values:     []float64{},
						Timestamps: []time.Time{},
					},
				},
			},
			expectedMetricDataResults: []cloudwatch_client.MetricDataResult{
				{ID: "metric-1", Datapoint: aws.Float64(1.0), Timestamp: ts.Add(10 * time.Minute)},
				{ID: "metric-2", Datapoint: nil, Timestamp: time.Time{}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metricDataResults := toMetricDataResult(tc.getMetricDataOutput)
			require.Equal(t, tc.expectedMetricDataResults, metricDataResults)
		})
	}
}
