package job

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/grafana/regexp"

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
		dimensionRegexps          []*regexp.Regexp
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
				dimensionRegexps: config.SupportedServices.GetService("efs").DimensionRegexps,
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
					AddCloudwatchTimestamp: aws.Bool(false),
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
					ID:        aws.String("arn:aws:elasticfilesystem:us-east-1:123123123123:file-system/fs-abc123"),
					Metric:    aws.String("StorageBytes"),
					Namespace: aws.String("efs"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Statistics: []string{
						"Average",
					},
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
				dimensionRegexps: config.SupportedServices.GetService("ec2").DimensionRegexps,
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
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*model.Dimension{
						{
							Name:  "InstanceId",
							Value: "i-12312312312312312",
						},
					},
					ID:        aws.String("arn:aws:ec2:us-east-1:123123123123:instance/i-12312312312312312"),
					Metric:    aws.String("CPUUtilization"),
					Namespace: aws.String("ec2"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Statistics: []string{
						"Average",
					},
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
				dimensionRegexps: config.SupportedServices.GetService("kafka").DimensionRegexps,
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
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*model.Dimension{
						{
							Name:  "Cluster Name",
							Value: "demo-cluster-1",
						},
					},
					ID:        aws.String("arn:aws:kafka:us-east-1:123123123123:cluster/demo-cluster-1/12312312-1231-1231-1231-123123123123-12"),
					Metric:    aws.String("GlobalTopicCount"),
					Namespace: aws.String("kafka"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Statistics: []string{
						"Average",
					},
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
				dimensionRegexps:          config.SupportedServices.GetService("alb").DimensionRegexps,
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
					AddCloudwatchTimestamp: aws.Bool(false),
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
					ID:        aws.String("arn:aws:elasticloadbalancing:us-east-1:123123123123:loadbalancer/app/some-ALB/0123456789012345"),
					Metric:    aws.String("RequestCount"),
					Namespace: aws.String("alb"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Statistics: []string{
						"Sum",
					},
					Tags: []model.Tag{},
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
				if *got.ID != *tt.wantGetMetricsData[i].ID {
					t.Errorf("getFilteredMetricDatas().ID = %v, want %v", *got.ID, *tt.wantGetMetricsData[i].ID)
				}
				if !reflect.DeepEqual(got.Dimensions, tt.wantGetMetricsData[i].Dimensions) {
					t.Errorf("getFilteredMetricDatas().Dimensions = %+v, want %+v", got.Dimensions, tt.wantGetMetricsData[i].Dimensions)
				}
				if *got.Metric != *tt.wantGetMetricsData[i].Metric {
					t.Errorf("getFilteredMetricDatas().Metric = %v, want %v", *got.Metric, *tt.wantGetMetricsData[i].Metric)
				}
				if *got.Namespace != *tt.wantGetMetricsData[i].Namespace {
					t.Errorf("getFilteredMetricDatas().Namespace = %v, want %v", *got.Namespace, *tt.wantGetMetricsData[i].Namespace)
				}
				if *got.AddCloudwatchTimestamp != *tt.wantGetMetricsData[i].AddCloudwatchTimestamp {
					t.Errorf("getFilteredMetricDatas().AddCloudwatchTimestamp = %v, want %v", *got.AddCloudwatchTimestamp, *tt.wantGetMetricsData[i].AddCloudwatchTimestamp)
				}
				if *got.NilToZero != *tt.wantGetMetricsData[i].NilToZero {
					t.Errorf("getFilteredMetricDatas().NilToZero = %v, want %v", *got.NilToZero, *tt.wantGetMetricsData[i].NilToZero)
				}
				if got.Period != tt.wantGetMetricsData[i].Period {
					t.Errorf("getFilteredMetricDatas().Period = %v, want %v", got.Period, tt.wantGetMetricsData[i].Period)
				}
				if !reflect.DeepEqual(got.Statistics, tt.wantGetMetricsData[i].Statistics) {
					t.Errorf("getFilteredMetricDatas().Statistics = %+v, want %+v", got.Statistics, tt.wantGetMetricsData[i].Statistics)
				}
				if !reflect.DeepEqual(got.Tags, tt.wantGetMetricsData[i].Tags) {
					t.Errorf("getFilteredMetricDatas().Tags = %+v, want %+v", got.Tags, tt.wantGetMetricsData[i].Tags)
				}
			}
		})
	}
}

func getSampleMetricDatas(id string) *model.CloudwatchData {
	return &model.CloudwatchData{
		AddCloudwatchTimestamp: aws.Bool(false),
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
		ID:        aws.String(id),
		MetricID:  aws.String(id),
		Metric:    aws.String("StorageBytes"),
		Namespace: aws.String("efs"),
		NilToZero: aws.Bool(false),
		Period:    60,
		Statistics: []string{
			"Average",
		},
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
