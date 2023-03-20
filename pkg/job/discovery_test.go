package job

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func Test_getFilteredMetricDatas(t *testing.T) {
	type args struct {
		region                    string
		accountID                 *string
		namespace                 string
		customTags                []model.Tag
		tagsOnMetrics             model.ExportedTagsOnMetrics
		dimensionRegexps          []*regexp.Regexp
		dimensionNameRequirements []string
		resources                 []*model.TaggedResource
		metricsList               []*cloudwatch.Metric
		m                         *config.Metric
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
				accountID:  aws.String("123123123123"),
				namespace:  "efs",
				customTags: nil,
				tagsOnMetrics: map[string][]string{
					"efs": {
						"Value1",
						"Value2",
					},
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
				metricsList: []*cloudwatch.Metric{
					{
						MetricName: aws.String("StorageBytes"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("FileSystemId"),
								Value: aws.String("fs-abc123"),
							},
							{
								Name:  aws.String("StorageClass"),
								Value: aws.String("Standard"),
							},
						},
						Namespace: aws.String("AWS/EFS"),
					},
				},
				m: &config.Metric{
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
					AccountID:              aws.String("123123123123"),
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("FileSystemId"),
							Value: aws.String("fs-abc123"),
						},
						{
							Name:  aws.String("StorageClass"),
							Value: aws.String("Standard"),
						},
					},
					ID:        aws.String("arn:aws:elasticfilesystem:us-east-1:123123123123:file-system/fs-abc123"),
					Metric:    aws.String("StorageBytes"),
					Namespace: aws.String("efs"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Region:    aws.String("us-east-1"),
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
				accountID:  aws.String("123123123123"),
				namespace:  "ec2",
				customTags: nil,
				tagsOnMetrics: map[string][]string{
					"ec2": {
						"Value1",
						"Value2",
					},
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
				metricsList: []*cloudwatch.Metric{
					{
						MetricName: aws.String("CPUUtilization"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String("i-12312312312312312"),
							},
						},
						Namespace: aws.String("AWS/EC2"),
					},
				},
				m: &config.Metric{
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
					AccountID:              aws.String("123123123123"),
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("InstanceId"),
							Value: aws.String("i-12312312312312312"),
						},
					},
					ID:        aws.String("arn:aws:ec2:us-east-1:123123123123:instance/i-12312312312312312"),
					Metric:    aws.String("CPUUtilization"),
					Namespace: aws.String("ec2"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Region:    aws.String("us-east-1"),
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
				accountID:  aws.String("123123123123"),
				namespace:  "kafka",
				customTags: nil,
				tagsOnMetrics: map[string][]string{
					"kafka": {
						"Value1",
						"Value2",
					},
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
				metricsList: []*cloudwatch.Metric{
					{
						MetricName: aws.String("GlobalTopicCount"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("Cluster Name"),
								Value: aws.String("demo-cluster-1"),
							},
						},
						Namespace: aws.String("AWS/Kafka"),
					},
				},
				m: &config.Metric{
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
					AccountID:              aws.String("123123123123"),
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("Cluster Name"),
							Value: aws.String("demo-cluster-1"),
						},
					},
					ID:        aws.String("arn:aws:kafka:us-east-1:123123123123:cluster/demo-cluster-1/12312312-1231-1231-1231-123123123123-12"),
					Metric:    aws.String("GlobalTopicCount"),
					Namespace: aws.String("kafka"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Region:    aws.String("us-east-1"),
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
				accountID:                 aws.String("123123123123"),
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
				metricsList: []*cloudwatch.Metric{
					{
						MetricName: aws.String("RequestCount"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("LoadBalancer"),
								Value: aws.String("app/some-ALB/0123456789012345"),
							},
							{
								Name:  aws.String("TargetGroup"),
								Value: aws.String("targetgroup/some-ALB/9999666677773333"),
							},
							{
								Name:  aws.String("AvailabilityZone"),
								Value: aws.String("us-east-1"),
							},
						},
						Namespace: aws.String("AWS/ApplicationELB"),
					},
					{
						MetricName: aws.String("RequestCount"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("LoadBalancer"),
								Value: aws.String("app/some-ALB/0123456789012345"),
							},
							{
								Name:  aws.String("TargetGroup"),
								Value: aws.String("targetgroup/some-ALB/9999666677773333"),
							},
						},
						Namespace: aws.String("AWS/ApplicationELB"),
					},
					{
						MetricName: aws.String("RequestCount"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("LoadBalancer"),
								Value: aws.String("app/some-ALB/0123456789012345"),
							},
							{
								Name:  aws.String("AvailabilityZone"),
								Value: aws.String("us-east-1"),
							},
						},
						Namespace: aws.String("AWS/ApplicationELB"),
					},
					{
						MetricName: aws.String("RequestCount"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("LoadBalancer"),
								Value: aws.String("app/some-ALB/0123456789012345"),
							},
						},
						Namespace: aws.String("AWS/ApplicationELB"),
					},
				},
				m: &config.Metric{
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
					AccountID:              aws.String("123123123123"),
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("LoadBalancer"),
							Value: aws.String("app/some-ALB/0123456789012345"),
						},
						{
							Name:  aws.String("TargetGroup"),
							Value: aws.String("targetgroup/some-ALB/9999666677773333"),
						},
					},
					ID:        aws.String("arn:aws:elasticloadbalancing:us-east-1:123123123123:loadbalancer/app/some-ALB/0123456789012345"),
					Metric:    aws.String("RequestCount"),
					Namespace: aws.String("alb"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Region:    aws.String("us-east-1"),
					Statistics: []string{
						"Sum",
					},
					Tags: []model.Tag{},
				},
			},
		},
		{
			"best effort matching in GlobalAccelerator metric",
			args{
				region:           "us-east-1",
				accountID:        aws.String("123123123123"),
				namespace:        "AWS/GlobalAccelerator",
				customTags:       nil,
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").DimensionRegexps,
				resources: []*model.TaggedResource{
					{
						ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/5555abcd-abcd-5555-abcd-5555EXAMPLE1",
						Namespace: "AWS/GlobalAccelerator",
						Region:    "us-east-1",
						Tags: []model.Tag{
							{Key: "Name", Value: "SomeAccelerator"},
						},
					},
				},
				tagsOnMetrics: map[string][]string{
					"AWS/GlobalAccelerator": {"Name"},
				},
				metricsList: []*cloudwatch.Metric{
					{
						MetricName: aws.String("NewFlowCount"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("Accelerator"),
								Value: aws.String("5555abcd-abcd-5555-abcd-5555EXAMPLE1"),
							},
							{
								Name:  aws.String("TransportProtocol"),
								Value: aws.String("tcp"),
							},
						},
						Namespace: aws.String("AWS/GlobalAccelerator"),
					},
				},
				m: &config.Metric{
					Name: "NewFlowCount",
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
					AccountID:              aws.String("123123123123"),
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("Accelerator"),
							Value: aws.String("5555abcd-abcd-5555-abcd-5555EXAMPLE1"),
						},
						{
							Name:  aws.String("TransportProtocol"),
							Value: aws.String("tcp"),
						},
					},
					ID:        aws.String("arn:aws:globalaccelerator::012345678901:accelerator/5555abcd-abcd-5555-abcd-5555EXAMPLE1"),
					Metric:    aws.String("NewFlowCount"),
					Namespace: aws.String("AWS/GlobalAccelerator"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Region:    aws.String("us-east-1"),
					Statistics: []string{
						"Average",
					},
					Tags: []model.Tag{
						{Key: "Name", Value: "SomeAccelerator"},
					},
				},
			},
		},
		{
			"ECS",
			args{
				region:           "us-east-1",
				accountID:        aws.String("123123123123"),
				namespace:        "AWS/ECS",
				customTags:       nil,
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").DimensionRegexps,
				resources: []*model.TaggedResource{
					{
						ARN:       "arn:aws:ecs:us-east-1:366620023056:cluster/scorekeep-cluster",
						Namespace: "AWS/ECS",
						Region:    "us-east-1",
						Tags: []model.Tag{
							{Key: "Name", Value: "scorekeep-cluster"},
						},
					},
					{
						ARN:       "arn:aws:ecs:us-east-1:366620023056:service/scorekeep-cluster/scorekeep-service",
						Namespace: "AWS/ECS",
						Region:    "us-east-1",
						Tags: []model.Tag{
							{Key: "Name", Value: "scorekeep-service"},
						},
					},
				},
				metricsList: []*cloudwatch.Metric{
					{
						MetricName: aws.String("CPUUtilization"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String("scorekeep-cluster"),
							},
							{
								Name:  aws.String("ServiceName"),
								Value: aws.String("scorekeep-service"),
							},
						},
						Namespace: aws.String("AWS/ECS"),
					},
				},
				m: &config.Metric{
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
					AccountID:              aws.String("123123123123"),
					AddCloudwatchTimestamp: aws.Bool(false),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("ClusterName"),
							Value: aws.String("scorekeep-cluster"),
						},
						{
							Name:  aws.String("ServiceName"),
							Value: aws.String("scorekeep-service"),
						},
					},
					ID:        aws.String("arn:aws:ecs:us-east-1:366620023056:service/scorekeep-cluster/scorekeep-service"),
					Metric:    aws.String("CPUUtilization"),
					Namespace: aws.String("AWS/ECS"),
					NilToZero: aws.Bool(false),
					Period:    60,
					Region:    aws.String("us-east-1"),
					Statistics: []string{
						"Average",
					},
					Tags: []model.Tag{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricDatas := getFilteredMetricDatas(logging.NewNopLogger(), tt.args.region, tt.args.accountID, tt.args.namespace, tt.args.customTags, tt.args.tagsOnMetrics, tt.args.dimensionRegexps, tt.args.resources, tt.args.metricsList, tt.args.dimensionNameRequirements, tt.args.m)
			if len(metricDatas) != len(tt.wantGetMetricsData) {
				t.Errorf("len(getFilteredMetricDatas()) = %v, want %v", len(metricDatas), len(tt.wantGetMetricsData))
			}
			for i, got := range metricDatas {
				if *got.AccountID != *tt.wantGetMetricsData[i].AccountID {
					t.Errorf("getFilteredMetricDatas().AccountId = %v, want %v", *got.AccountID, *tt.wantGetMetricsData[i].AccountID)
				}
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
				if *got.Region != *tt.wantGetMetricsData[i].Region {
					t.Errorf("getFilteredMetricDatas().Region = %v, want %v", *got.Region, *tt.wantGetMetricsData[i].Region)
				}
				if !reflect.DeepEqual(got.Tags, tt.wantGetMetricsData[i].Tags) {
					t.Errorf("getFilteredMetricDatas().Tags = %+v, want %+v", got.Tags, tt.wantGetMetricsData[i].Tags)
				}
			}
		})
	}
}
