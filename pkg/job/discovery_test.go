package job

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job/maxdimassociator"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func Test_getFilteredMetricDatas(t *testing.T) {
	type args struct {
		region                    string
		accountID                 string
		namespace                 string
		customTags                []model.Tag
		tagsOnMetrics             []string
		dimensionRegexps          []model.DimensionsRegexp
		dimensionNameRequirements []string
		resources                 []*model.TaggedResource
		metricsList               []*model.Metric
		m                         *model.MetricConfig
	}
	tests := []struct {
		name               string
		args               args
		wantGetMetricsData []model.CloudwatchData
	}{
		{
			"additional dimension",
			args{
				region:     "us-east-1",
				accountID:  "123123123123",
				namespace:  "efs",
				customTags: nil,
				tagsOnMetrics: []string{
					"Value1",
					"Value2",
				},
				dimensionRegexps: config.SupportedServices.GetService("AWS/EFS").ToModelDimensionsRegexp(),
				resources: []*model.TaggedResource{
					{
						ARN: "arn:aws:elasticfilesystem:us-east-1:123123123123:file-system/fs-abc123",
						Tags: []model.Tag{
							{
								Key:   "Tag",
								Value: "some-Tag",
							},
						},
						Namespace: "efs",
						Region:    "us-east-1",
					},
				},
				metricsList: []*model.Metric{
					{
						MetricName: "StorageBytes",
						Dimensions: []*model.Dimension{
							{
								Name:  "FileSystemId",
								Value: "fs-abc123",
							},
							{
								Name:  "StorageClass",
								Value: "Standard",
							},
						},
						Namespace: "AWS/EFS",
					},
				},
				m: &model.MetricConfig{
					Name: "StorageBytes",
					Statistics: []string{
						"Average",
					},
					Period:                 60,
					Length:                 600,
					Delay:                  120,
					NilToZero:              aws.Bool(false),
					AddCloudwatchTimestamp: aws.Bool(false),
				},
			},
			[]model.CloudwatchData{
				{
					Dimensions: []*model.Dimension{
						{
							Name:  "FileSystemId",
							Value: "fs-abc123",
						},
						{
							Name:  "StorageClass",
							Value: "Standard",
						},
					},
					ResourceName:        "arn:aws:elasticfilesystem:us-east-1:123123123123:file-system/fs-abc123",
					Namespace:           "efs",
					GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Average"},
					Tags: []model.Tag{
						{
							Key:   "Value1",
							Value: "",
						},
						{
							Key:   "Value2",
							Value: "",
						},
					},
					MetricConfig: &model.MetricConfig{
						Name: "StorageBytes",
						Statistics: []string{
							"Average",
						},
						Period:                 60,
						Length:                 600,
						Delay:                  120,
						NilToZero:              aws.Bool(false),
						AddCloudwatchTimestamp: aws.Bool(false),
					},
				},
			},
		},
		{
			"ec2",
			args{
				region:     "us-east-1",
				accountID:  "123123123123",
				namespace:  "ec2",
				customTags: nil,
				tagsOnMetrics: []string{
					"Value1",
					"Value2",
				},
				dimensionRegexps: config.SupportedServices.GetService("AWS/EC2").ToModelDimensionsRegexp(),
				resources: []*model.TaggedResource{
					{
						ARN: "arn:aws:ec2:us-east-1:123123123123:instance/i-12312312312312312",
						Tags: []model.Tag{
							{
								Key:   "Name",
								Value: "some-Node",
							},
						},
						Namespace: "ec2",
						Region:    "us-east-1",
					},
				},
				metricsList: []*model.Metric{
					{
						MetricName: "CPUUtilization",
						Dimensions: []*model.Dimension{
							{
								Name:  "InstanceId",
								Value: "i-12312312312312312",
							},
						},
						Namespace: "AWS/EC2",
					},
				},
				m: &model.MetricConfig{
					Name: "CPUUtilization",
					Statistics: []string{
						"Average",
					},
					Period:                 60,
					Length:                 600,
					Delay:                  120,
					NilToZero:              aws.Bool(false),
					AddCloudwatchTimestamp: aws.Bool(false),
				},
			},
			[]model.CloudwatchData{
				{
					Dimensions: []*model.Dimension{
						{
							Name:  "InstanceId",
							Value: "i-12312312312312312",
						},
					},
					ResourceName:        "arn:aws:ec2:us-east-1:123123123123:instance/i-12312312312312312",
					Namespace:           "ec2",
					GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Average"},
					Tags: []model.Tag{
						{
							Key:   "Value1",
							Value: "",
						},
						{
							Key:   "Value2",
							Value: "",
						},
					},
					MetricConfig: &model.MetricConfig{
						Name: "CPUUtilization",
						Statistics: []string{
							"Average",
						},
						Period:                 60,
						Length:                 600,
						Delay:                  120,
						NilToZero:              aws.Bool(false),
						AddCloudwatchTimestamp: aws.Bool(false),
					},
				},
			},
		},
		{
			"kafka",
			args{
				region:     "us-east-1",
				accountID:  "123123123123",
				namespace:  "kafka",
				customTags: nil,
				tagsOnMetrics: []string{
					"Value1",
					"Value2",
				},
				dimensionRegexps: config.SupportedServices.GetService("AWS/Kafka").ToModelDimensionsRegexp(),
				resources: []*model.TaggedResource{
					{
						ARN: "arn:aws:kafka:us-east-1:123123123123:cluster/demo-cluster-1/12312312-1231-1231-1231-123123123123-12",
						Tags: []model.Tag{
							{
								Key:   "Test",
								Value: "Value",
							},
						},
						Namespace: "kafka",
						Region:    "us-east-1",
					},
				},
				metricsList: []*model.Metric{
					{
						MetricName: "GlobalTopicCount",
						Dimensions: []*model.Dimension{
							{
								Name:  "Cluster Name",
								Value: "demo-cluster-1",
							},
						},
						Namespace: "AWS/Kafka",
					},
				},
				m: &model.MetricConfig{
					Name: "GlobalTopicCount",
					Statistics: []string{
						"Average",
					},
					Period:                 60,
					Length:                 600,
					Delay:                  120,
					NilToZero:              aws.Bool(false),
					AddCloudwatchTimestamp: aws.Bool(false),
				},
			},
			[]model.CloudwatchData{
				{
					Dimensions: []*model.Dimension{
						{
							Name:  "Cluster Name",
							Value: "demo-cluster-1",
						},
					},
					ResourceName:        "arn:aws:kafka:us-east-1:123123123123:cluster/demo-cluster-1/12312312-1231-1231-1231-123123123123-12",
					Namespace:           "kafka",
					GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Average"},
					Tags: []model.Tag{
						{
							Key:   "Value1",
							Value: "",
						},
						{
							Key:   "Value2",
							Value: "",
						},
					},
					MetricConfig: &model.MetricConfig{
						Name: "GlobalTopicCount",
						Statistics: []string{
							"Average",
						},
						Period:                 60,
						Length:                 600,
						Delay:                  120,
						NilToZero:              aws.Bool(false),
						AddCloudwatchTimestamp: aws.Bool(false),
					},
				},
			},
		},
		{
			"alb",
			args{
				region:                    "us-east-1",
				accountID:                 "123123123123",
				namespace:                 "alb",
				customTags:                nil,
				tagsOnMetrics:             nil,
				dimensionRegexps:          config.SupportedServices.GetService("AWS/ApplicationELB").ToModelDimensionsRegexp(),
				dimensionNameRequirements: []string{"LoadBalancer", "TargetGroup"},
				resources: []*model.TaggedResource{
					{
						ARN: "arn:aws:elasticloadbalancing:us-east-1:123123123123:loadbalancer/app/some-ALB/0123456789012345",
						Tags: []model.Tag{
							{
								Key:   "Name",
								Value: "some-ALB",
							},
						},
						Namespace: "alb",
						Region:    "us-east-1",
					},
				},
				metricsList: []*model.Metric{
					{
						MetricName: "RequestCount",
						Dimensions: []*model.Dimension{
							{
								Name:  "LoadBalancer",
								Value: "app/some-ALB/0123456789012345",
							},
							{
								Name:  "TargetGroup",
								Value: "targetgroup/some-ALB/9999666677773333",
							},
							{
								Name:  "AvailabilityZone",
								Value: "us-east-1",
							},
						},
						Namespace: "AWS/ApplicationELB",
					},
					{
						MetricName: "RequestCount",
						Dimensions: []*model.Dimension{
							{
								Name:  "LoadBalancer",
								Value: "app/some-ALB/0123456789012345",
							},
							{
								Name:  "TargetGroup",
								Value: "targetgroup/some-ALB/9999666677773333",
							},
						},
						Namespace: "AWS/ApplicationELB",
					},
					{
						MetricName: "RequestCount",
						Dimensions: []*model.Dimension{
							{
								Name:  "LoadBalancer",
								Value: "app/some-ALB/0123456789012345",
							},
							{
								Name:  "AvailabilityZone",
								Value: "us-east-1",
							},
						},
						Namespace: "AWS/ApplicationELB",
					},
					{
						MetricName: "RequestCount",
						Dimensions: []*model.Dimension{
							{
								Name:  "LoadBalancer",
								Value: "app/some-ALB/0123456789012345",
							},
						},
						Namespace: "AWS/ApplicationELB",
					},
				},
				m: &model.MetricConfig{
					Name: "RequestCount",
					Statistics: []string{
						"Sum",
					},
					Period:                 60,
					Length:                 600,
					Delay:                  120,
					NilToZero:              aws.Bool(false),
					AddCloudwatchTimestamp: aws.Bool(false),
				},
			},
			[]model.CloudwatchData{
				{
					Dimensions: []*model.Dimension{
						{
							Name:  "LoadBalancer",
							Value: "app/some-ALB/0123456789012345",
						},
						{
							Name:  "TargetGroup",
							Value: "targetgroup/some-ALB/9999666677773333",
						},
					},
					ResourceName:        "arn:aws:elasticloadbalancing:us-east-1:123123123123:loadbalancer/app/some-ALB/0123456789012345",
					Namespace:           "alb",
					GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Sum"},
					Tags:                []model.Tag{},
					MetricConfig: &model.MetricConfig{
						Name: "RequestCount",
						Statistics: []string{
							"Sum",
						},
						Period:                 60,
						Length:                 600,
						Delay:                  120,
						NilToZero:              aws.Bool(false),
						AddCloudwatchTimestamp: aws.Bool(false),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assoc := maxdimassociator.NewAssociator(logging.NewNopLogger(), tt.args.dimensionRegexps, tt.args.resources)
			metricDatas := getFilteredMetricDatas(logging.NewNopLogger(), tt.args.namespace, tt.args.tagsOnMetrics, tt.args.metricsList, tt.args.dimensionNameRequirements, tt.args.m, assoc)
			if len(metricDatas) != len(tt.wantGetMetricsData) {
				t.Errorf("len(getFilteredMetricDatas()) = %v, want %v", len(metricDatas), len(tt.wantGetMetricsData))
			}
			for i, got := range metricDatas {
				want := tt.wantGetMetricsData[i]
				assert.Equal(t, want.ResourceName, got.ResourceName)
				assert.Equal(t, want.Namespace, got.Namespace)
				assert.ElementsMatch(t, want.Dimensions, got.Dimensions)
				assert.Equal(t, want.GetMetricDataResult.Statistic, got.GetMetricDataResult.Statistic)
				assert.ElementsMatch(t, want.Tags, got.Tags)
				assert.Equal(t, *want.MetricConfig, *got.MetricConfig)
			}
		})
	}
}

func Test_mapResultsToMetricDatas(t *testing.T) {
	type args struct {
		metricDataResults [][]cloudwatch.MetricDataResult
		cloudwatchDatas   []*model.CloudwatchData
	}
	tests := []struct {
		name                string
		args                args
		wantCloudwatchDatas []*model.CloudwatchData
	}{
		{
			"all datapoints present",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-3", Datapoint: 15, Timestamp: time.Date(2023, time.June, 7, 3, 9, 8, 0, time.UTC)},
						{ID: "metric-1", Datapoint: 5, Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
					},
					{
						{ID: "metric-4", Datapoint: 20, Timestamp: time.Date(2023, time.June, 7, 4, 9, 8, 0, time.UTC)},
					},
					{
						{ID: "metric-2", Datapoint: 12, Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-1"}, MetricConfig: &model.MetricConfig{Name: "MetricOne"}, Namespace: "svc"},
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-2"}, MetricConfig: &model.MetricConfig{Name: "MetricTwo"}, Namespace: "svc"},
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-3"}, MetricConfig: &model.MetricConfig{Name: "MetricThree"}, Namespace: "svc"},
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-4"}, MetricConfig: &model.MetricConfig{Name: "MetricFour"}, Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricConfig: &model.MetricConfig{Name: "MetricOne"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-1",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            5,
						Timestamp:            time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricConfig: &model.MetricConfig{Name: "MetricTwo"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-2",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            12,
						Timestamp:            time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricConfig: &model.MetricConfig{Name: "MetricThree"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-3",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            15,
						Timestamp:            time.Date(2023, time.June, 7, 3, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricConfig: &model.MetricConfig{Name: "MetricFour"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-4",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            20,
						Timestamp:            time.Date(2023, time.June, 7, 4, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"duplicate results",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: 5, Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
						{ID: "metric-1", Datapoint: 15, Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-1"}, MetricConfig: &model.MetricConfig{Name: "MetricOne"}, Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricConfig: &model.MetricConfig{Name: "MetricOne"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-1",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            5,
						Timestamp:            time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"unexpected result ID",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: 5, Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
						{ID: "metric-2", Datapoint: 15, Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-1"}, MetricConfig: &model.MetricConfig{Name: "MetricOne"}, Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricConfig: &model.MetricConfig{Name: "MetricOne"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-1",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            5,
						Timestamp:            time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"nil metric data result",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: 5, Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
					},
					nil,
					{
						{ID: "metric-2", Datapoint: 12, Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-1"}, MetricConfig: &model.MetricConfig{Name: "MetricOne"}, Namespace: "svc"},
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-2"}, MetricConfig: &model.MetricConfig{Name: "MetricTwo"}, Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricConfig: &model.MetricConfig{Name: "MetricOne"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-1",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            5,
						Timestamp:            time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricConfig: &model.MetricConfig{Name: "MetricTwo"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-2",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            12,
						Timestamp:            time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"missing metric data result",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: 5, Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-1"}, MetricConfig: &model.MetricConfig{Name: "MetricOne"}, Namespace: "svc"},
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-2"}, MetricConfig: &model.MetricConfig{Name: "MetricTwo"}, Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricConfig: &model.MetricConfig{Name: "MetricOne"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-1",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            5,
						Timestamp:            time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricConfig: &model.MetricConfig{Name: "MetricTwo"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-2",
						MappedToAQueryResult: false,
						Statistic:            "",
						Datapoint:            0,
						Timestamp:            time.Time{},
					},
				},
			},
		},
		{
			"missing metric datapoint",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: 5, Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
						{ID: "metric-2"},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-1"}, MetricConfig: &model.MetricConfig{Name: "MetricOne"}, Namespace: "svc"},
					{GetMetricDataResult: &model.GetMetricDataResult{ID: "metric-2"}, MetricConfig: &model.MetricConfig{Name: "MetricTwo"}, Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricConfig: &model.MetricConfig{Name: "MetricOne"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-1",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            5,
						Timestamp:            time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricConfig: &model.MetricConfig{Name: "MetricTwo"},
					Namespace:    "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						ID:                   "metric-2",
						MappedToAQueryResult: true,
						Statistic:            "",
						Datapoint:            0,
						Timestamp:            time.Time{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapResultsToMetricDatas(tt.args.metricDataResults, tt.args.cloudwatchDatas, logging.NewNopLogger())
			// mapResultsToMetricDatas() modifies its []*model.CloudwatchData parameter in-place, assert that it was updated
			require.Equal(t, tt.wantCloudwatchDatas, tt.args.cloudwatchDatas)
		})
	}
}

func getSampleMetricDatas(id string) *model.CloudwatchData {
	return &model.CloudwatchData{
		Dimensions: []*model.Dimension{
			{
				Name:  "FileSystemId",
				Value: "fs-abc123",
			},
			{
				Name:  "StorageClass",
				Value: "Standard",
			},
		},
		ResourceName: id,
		Namespace:    "efs",
		Tags: []model.Tag{
			{
				Key:   "Value1",
				Value: "",
			},
			{
				Key:   "Value2",
				Value: "",
			},
		},
		MetricConfig: &model.MetricConfig{
			Name:      "StorageBytes",
			NilToZero: aws.Bool(false),
			Period:    60,
			Statistics: []string{
				"Average",
			},
		},
		GetMetricDataResult: &model.GetMetricDataResult{
			ID: id,
		},
	}
}

func BenchmarkMapResultsToMetricDatas(b *testing.B) {
	type testcase struct {
		metricsPerQuery    int
		testResourcesCount int
		metricsPerResource int
	}

	for name, tc := range map[string]testcase{
		"small case": {
			metricsPerQuery:    500,
			testResourcesCount: 10,
			metricsPerResource: 10,
		},
		"medium case": {
			metricsPerQuery:    500,
			testResourcesCount: 1000,
			metricsPerResource: 50,
		},
		"big case": {
			metricsPerQuery:    500,
			testResourcesCount: 2000,
			metricsPerResource: 50,
		},
	} {
		b.Run(name, func(b *testing.B) {
			doBench(b, tc.metricsPerQuery, tc.testResourcesCount, tc.metricsPerResource)
		})
	}
}

func doBench(b *testing.B, metricsPerQuery, testResourcesCount, metricsPerResource int) {
	outputs := [][]cloudwatch.MetricDataResult{}
	now := time.Now()
	testResourceIDs := make([]string, testResourcesCount)

	for i := 0; i < testResourcesCount; i++ {
		testResourceIDs[i] = fmt.Sprintf("test-resource-%d", i)
	}

	totalMetricsDatapoints := metricsPerResource * testResourcesCount
	batchesCount := totalMetricsDatapoints / metricsPerQuery

	if batchesCount == 0 {
		batchesCount = 1
	}

	for batch := 0; batch < batchesCount; batch++ {
		newBatchOutputs := make([]cloudwatch.MetricDataResult, 0)
		for i := 0; i < metricsPerQuery; i++ {
			id := testResourceIDs[(batch*metricsPerQuery+i)%testResourcesCount]
			newBatchOutputs = append(newBatchOutputs, cloudwatch.MetricDataResult{
				ID:        id,
				Datapoint: 1.4 * float64(batch),
				Timestamp: now,
			})
		}
		outputs = append(outputs, newBatchOutputs)
	}

	for i := 0; i < b.N; i++ {
		// stop timer to not affect benchmark run
		// this has to do in every run, since mapResultsToMetricDatas mutates the metric datas slice
		b.StopTimer()
		datas := []*model.CloudwatchData{}
		for i := 0; i < testResourcesCount; i++ {
			datas = append(datas, getSampleMetricDatas(testResourceIDs[i]))
		}
		// re-start timer
		b.StartTimer()
		mapResultsToMetricDatas(outputs, datas, logging.NewNopLogger())
	}
}
