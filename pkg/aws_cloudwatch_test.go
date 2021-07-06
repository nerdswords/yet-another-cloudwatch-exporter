package exporter

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func TestDimensionsToCliString(t *testing.T) {
	// Setup Test

	// Arrange
	dimensions := []*cloudwatch.Dimension{}
	expected := ""

	// Act
	actual := dimensionsToCliString(dimensions)

	// Assert
	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}

// TestSortyByTimeStamp validates that sortByTimestamp() sorts in descending order.
func TestSortyByTimeStamp(t *testing.T) {
	cloudWatchDataPoints := make([]*cloudwatch.Datapoint, 3)
	maxValue1 := float64(1)
	maxValue2 := float64(2)
	maxValue3 := float64(3)

	dataPointMiddle := &cloudwatch.Datapoint{}
	twoMinutesAgo := time.Now().Add(time.Minute * 2 * -1)
	dataPointMiddle.Timestamp = &twoMinutesAgo
	dataPointMiddle.Maximum = &maxValue2
	cloudWatchDataPoints[0] = dataPointMiddle

	dataPointNewest := &cloudwatch.Datapoint{}
	oneMinutesAgo := time.Now().Add(time.Minute * -1)
	dataPointNewest.Timestamp = &oneMinutesAgo
	dataPointNewest.Maximum = &maxValue1
	cloudWatchDataPoints[1] = dataPointNewest

	dataPointOldest := &cloudwatch.Datapoint{}
	threeMinutesAgo := time.Now().Add(time.Minute * 3 * -1)
	dataPointOldest.Timestamp = &threeMinutesAgo
	dataPointOldest.Maximum = &maxValue3
	cloudWatchDataPoints[2] = dataPointOldest

	sortedDataPoints := sortByTimestamp(cloudWatchDataPoints)

	equals(t, maxValue1, *sortedDataPoints[0].Maximum)
	equals(t, maxValue2, *sortedDataPoints[1].Maximum)
	equals(t, maxValue3, *sortedDataPoints[2].Maximum)
}

func Test_getFilteredMetricDatas(t *testing.T) {
	type args struct {
		region           string
		accountId        *string
		namespace        string
		tagPriorites     []string
		customTags       []Tag
		tagsOnMetrics    exportedTagsOnMetrics
		dimensionRegexps []*string
		resources        []*tagsData
		metricsList      []*cloudwatch.Metric
		m                *Metric
	}
	tests := []struct {
		name               string
		args               args
		wantGetMetricsData []cloudwatchData
	}{
		{
			"ec2",
			args{
				region:     "us-east-1",
				accountId:  aws.String("123123123123"),
				namespace:  "ec2",
				customTags: nil,
				tagsOnMetrics: map[string][]string{
					"ec2": {
						"Value1",
						"Value2",
					},
				},
				dimensionRegexps: SupportedServices.GetService("ec2").DimensionRegexps,
				resources: []*tagsData{
					{
						ID: aws.String("arn:aws:ec2:us-east-1:123123123123:instance/i-12312312312312312"),
						Tags: []*Tag{
							{
								Key:   "Name",
								Value: "some-Node",
							},
						},
						Namespace: aws.String("ec2"),
						Region:    aws.String("us-east-1"),
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
				m: &Metric{
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
			[]cloudwatchData{
				{
					AccountId:              aws.String("123123123123"),
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
					Tags: []Tag{
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
				accountId:  aws.String("123123123123"),
				namespace:  "kafka",
				customTags: nil,
				tagsOnMetrics: map[string][]string{
					"kafka": {
						"Value1",
						"Value2",
					},
				},
				dimensionRegexps: SupportedServices.GetService("kafka").DimensionRegexps,
				resources: []*tagsData{
					{
						ID: aws.String("arn:aws:kafka:us-east-1:123123123123:cluster/demo-cluster-1/12312312-1231-1231-1231-123123123123-12"),
						Tags: []*Tag{
							{
								Key:   "Test",
								Value: "Value",
							},
						},
						Namespace: aws.String("kafka"),
						Region:    aws.String("us-east-1"),
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
				m: &Metric{
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
			[]cloudwatchData{
				{
					AccountId:              aws.String("123123123123"),
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
					Tags: []Tag{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, got := range getFilteredMetricDatas(tt.args.region, tt.args.accountId, tt.args.namespace, tt.args.customTags, tt.args.tagPriorites, tt.args.tagsOnMetrics, tt.args.dimensionRegexps, tt.args.resources, tt.args.metricsList, tt.args.m) {
				if *got.AccountId != *tt.wantGetMetricsData[i].AccountId {
					t.Errorf("getFilteredMetricDatas().AccountId = %v, want %v", *got.AccountId, *tt.wantGetMetricsData[i].AccountId)
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
