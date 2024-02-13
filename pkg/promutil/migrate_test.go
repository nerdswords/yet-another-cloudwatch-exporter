package promutil

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestBuildNamespaceInfoMetrics(t *testing.T) {
	type testCase struct {
		name                 string
		resources            []model.TaggedResourceResult
		metrics              []*PrometheusMetric
		observedMetricLabels map[string]model.LabelSet
		labelsSnakeCase      bool
		expectedMetrics      []*PrometheusMetric
		expectedLabels       map[string]model.LabelSet
	}
	testCases := []testCase{
		{
			name: "metric with tag",
			resources: []model.TaggedResourceResult{
				{
					Context: nil,
					Data: []*model.TaggedResource{
						{
							ARN:       "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
							Namespace: "AWS/ElastiCache",
							Region:    "us-east-1",
							Tags: []model.Tag{
								{
									Key:   "CustomTag",
									Value: "tag_Value",
								},
							},
						},
					},
				},
			},
			metrics:              []*PrometheusMetric{},
			observedMetricLabels: map[string]model.LabelSet{},
			labelsSnakeCase:      false,
			expectedMetrics: []*PrometheusMetric{
				{
					Name: aws.String("aws_elasticache_info"),
					Labels: map[string]string{
						"name":          "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"tag_CustomTag": "tag_Value",
					},
					Value: aws.Float64(0),
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_elasticache_info": map[string]struct{}{
					"name":          {},
					"tag_CustomTag": {},
				},
			},
		},
		{
			name: "label snake case",
			resources: []model.TaggedResourceResult{
				{
					Context: nil,
					Data: []*model.TaggedResource{
						{
							ARN:       "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
							Namespace: "AWS/ElastiCache",
							Region:    "us-east-1",
							Tags: []model.Tag{
								{
									Key:   "CustomTag",
									Value: "tag_Value",
								},
							},
						},
					},
				},
			},
			metrics:              []*PrometheusMetric{},
			observedMetricLabels: map[string]model.LabelSet{},
			labelsSnakeCase:      true,
			expectedMetrics: []*PrometheusMetric{
				{
					Name: aws.String("aws_elasticache_info"),
					Labels: map[string]string{
						"name":           "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"tag_custom_tag": "tag_Value",
					},
					Value: aws.Float64(0),
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_elasticache_info": map[string]struct{}{
					"name":           {},
					"tag_custom_tag": {},
				},
			},
		},
		{
			name: "with observed metrics and labels",
			resources: []model.TaggedResourceResult{
				{
					Context: nil,
					Data: []*model.TaggedResource{
						{
							ARN:       "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
							Namespace: "AWS/ElastiCache",
							Region:    "us-east-1",
							Tags: []model.Tag{
								{
									Key:   "CustomTag",
									Value: "tag_Value",
								},
							},
						},
					},
				},
			},
			metrics: []*PrometheusMetric{
				{
					Name: aws.String("aws_ec2_cpuutilization_maximum"),
					Labels: map[string]string{
						"name":                 "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
						"dimension_InstanceId": "i-abc123",
					},
					Value: aws.Float64(0),
				},
			},
			observedMetricLabels: map[string]model.LabelSet{
				"aws_ec2_cpuutilization_maximum": map[string]struct{}{
					"name":                 {},
					"dimension_InstanceId": {},
				},
			},
			labelsSnakeCase: true,
			expectedMetrics: []*PrometheusMetric{
				{
					Name: aws.String("aws_ec2_cpuutilization_maximum"),
					Labels: map[string]string{
						"name":                 "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
						"dimension_InstanceId": "i-abc123",
					},
					Value: aws.Float64(0),
				},
				{
					Name: aws.String("aws_elasticache_info"),
					Labels: map[string]string{
						"name":           "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"tag_custom_tag": "tag_Value",
					},
					Value: aws.Float64(0),
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_ec2_cpuutilization_maximum": map[string]struct{}{
					"name":                 {},
					"dimension_InstanceId": {},
				},
				"aws_elasticache_info": map[string]struct{}{
					"name":           {},
					"tag_custom_tag": {},
				},
			},
		},
		{
			name: "context on info metrics",
			resources: []model.TaggedResourceResult{
				{
					Context: &model.ScrapeContext{
						Region:    "us-east-2",
						AccountID: "12345",
						CustomTags: []model.Tag{{
							Key:   "billable-to",
							Value: "api",
						}},
					},
					Data: []*model.TaggedResource{
						{
							ARN:       "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
							Namespace: "AWS/ElastiCache",
							Region:    "us-east-1",
							Tags: []model.Tag{
								{
									Key:   "cache_name",
									Value: "cache_instance_1",
								},
							},
						},
					},
				},
			},
			metrics:              []*PrometheusMetric{},
			observedMetricLabels: map[string]model.LabelSet{},
			labelsSnakeCase:      true,
			expectedMetrics: []*PrometheusMetric{
				{
					Name: aws.String("aws_elasticache_info"),
					Labels: map[string]string{
						"name":                   "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"tag_cache_name":         "cache_instance_1",
						"account_id":             "12345",
						"region":                 "us-east-2",
						"custom_tag_billable_to": "api",
					},
					Value: aws.Float64(0),
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_elasticache_info": map[string]struct{}{
					"name":                   {},
					"tag_cache_name":         {},
					"account_id":             {},
					"region":                 {},
					"custom_tag_billable_to": {},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics, labels := BuildNamespaceInfoMetrics(tc.resources, tc.metrics, tc.observedMetricLabels, tc.labelsSnakeCase, logging.NewNopLogger())
			require.Equal(t, tc.expectedMetrics, metrics)
			require.Equal(t, tc.expectedLabels, labels)
		})
	}
}

func TestBuildMetrics(t *testing.T) {
	ts := time.Now()

	type testCase struct {
		name            string
		data            []model.CloudwatchMetricResult
		labelsSnakeCase bool
		expectedMetrics []*PrometheusMetric
		expectedLabels  map[string]model.LabelSet
		expectedErr     error
	}

	testCases := []testCase{
		{
			name: "metric with non-nil data point",
			data: []model.CloudwatchMetricResult{{
				Context: &model.ScrapeContext{
					Region:     "us-east-1",
					AccountID:  "123456789012",
					CustomTags: nil,
				},
				Data: []*model.CloudwatchData{
					{
						Metric:     aws.String("CPUUtilization"),
						Namespace:  aws.String("AWS/ElastiCache"),
						Statistics: []string{"Average"},
						Dimensions: []*model.Dimension{
							{
								Name:  "CacheClusterId",
								Value: "redis-cluster",
							},
						},
						NilToZero:               aws.Bool(false),
						GetMetricDataPoint:      aws.Float64(1),
						GetMetricDataTimestamps: ts,
						ID:                      aws.String("arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster"),
					},
				},
			}},
			labelsSnakeCase: false,
			expectedMetrics: []*PrometheusMetric{
				{
					Name:      aws.String("aws_elasticache_cpuutilization_average"),
					Value:     aws.Float64(1),
					Timestamp: ts,
					Labels: map[string]string{
						"account_id":               "123456789012",
						"name":                     "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"region":                   "us-east-1",
						"dimension_CacheClusterId": "redis-cluster",
					},
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_elasticache_cpuutilization_average": {
					"account_id":               {},
					"name":                     {},
					"region":                   {},
					"dimension_CacheClusterId": {},
				},
			},
			expectedErr: nil,
		},
		{
			name: "label snake case",
			data: []model.CloudwatchMetricResult{{
				Context: &model.ScrapeContext{
					Region:     "us-east-1",
					AccountID:  "123456789012",
					CustomTags: nil,
				},
				Data: []*model.CloudwatchData{
					{
						Metric:     aws.String("CPUUtilization"),
						Namespace:  aws.String("AWS/ElastiCache"),
						Statistics: []string{"Average"},
						Dimensions: []*model.Dimension{
							{
								Name:  "CacheClusterId",
								Value: "redis-cluster",
							},
						},
						NilToZero:               aws.Bool(false),
						GetMetricDataPoint:      aws.Float64(1),
						GetMetricDataTimestamps: ts,
						ID:                      aws.String("arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster"),
					},
				},
			}},
			labelsSnakeCase: true,
			expectedMetrics: []*PrometheusMetric{
				{
					Name:      aws.String("aws_elasticache_cpuutilization_average"),
					Value:     aws.Float64(1),
					Timestamp: ts,
					Labels: map[string]string{
						"account_id":                 "123456789012",
						"name":                       "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"region":                     "us-east-1",
						"dimension_cache_cluster_id": "redis-cluster",
					},
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_elasticache_cpuutilization_average": {
					"account_id":                 {},
					"name":                       {},
					"region":                     {},
					"dimension_cache_cluster_id": {},
				},
			},
			expectedErr: nil,
		},
		{
			name: "custom tag",
			data: []model.CloudwatchMetricResult{{
				Context: &model.ScrapeContext{
					Region:    "us-east-1",
					AccountID: "123456789012",
					CustomTags: []model.Tag{{
						Key:   "billable-to",
						Value: "api",
					}},
				},
				Data: []*model.CloudwatchData{
					{
						Metric:     aws.String("CPUUtilization"),
						Namespace:  aws.String("AWS/ElastiCache"),
						Statistics: []string{"Average"},
						Dimensions: []*model.Dimension{
							{
								Name:  "CacheClusterId",
								Value: "redis-cluster",
							},
						},
						NilToZero:               aws.Bool(false),
						GetMetricDataPoint:      aws.Float64(1),
						GetMetricDataTimestamps: ts,
						ID:                      aws.String("arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster"),
					},
				},
			}},
			labelsSnakeCase: true,
			expectedMetrics: []*PrometheusMetric{
				{
					Name:      aws.String("aws_elasticache_cpuutilization_average"),
					Value:     aws.Float64(1),
					Timestamp: ts,
					Labels: map[string]string{
						"account_id":                 "123456789012",
						"name":                       "arn:aws:elasticache:us-east-1:123456789012:cluster:redis-cluster",
						"region":                     "us-east-1",
						"dimension_cache_cluster_id": "redis-cluster",
						"custom_tag_billable_to":     "api",
					},
				},
			},
			expectedLabels: map[string]model.LabelSet{
				"aws_elasticache_cpuutilization_average": {
					"account_id":                 {},
					"name":                       {},
					"region":                     {},
					"dimension_cache_cluster_id": {},
					"custom_tag_billable_to":     {},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, labels, err := BuildMetrics(tc.data, tc.labelsSnakeCase, logging.NewNopLogger())
			if tc.expectedErr != nil {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedMetrics, res)
				require.Equal(t, tc.expectedLabels, labels)
			}
		})
	}
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
			observedLabels: map[string]model.LabelSet{"metric1": {"label1": {}, "label2": {}, "label3": {}}},
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
