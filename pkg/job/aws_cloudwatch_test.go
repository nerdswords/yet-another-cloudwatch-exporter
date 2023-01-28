package job

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/services"
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

func Test_getFilteredMetricDatas(t *testing.T) {
	type args struct {
		region                    string
		accountID                 *string
		namespace                 string
		customTags                []model.Tag
		tagsOnMetrics             config.ExportedTagsOnMetrics
		dimensionRegexps          []*regexp.Regexp
		dimensionNameRequirements []string
		resources                 []*services.TaggedResource
		metricsList               []*cloudwatch.Metric
		m                         *config.Metric
	}
	tests := []struct {
		name               string
		args               args
		wantGetMetricsData []cloudwatchData
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
				resources: []*services.TaggedResource{
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
			[]cloudwatchData{
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
				resources: []*services.TaggedResource{
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
			[]cloudwatchData{
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
				resources: []*services.TaggedResource{
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
			[]cloudwatchData{
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
				resources: []*services.TaggedResource{
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
			[]cloudwatchData{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricDatas := getFilteredMetricDatas(tt.args.region, tt.args.accountID, tt.args.namespace, tt.args.customTags, tt.args.tagsOnMetrics, tt.args.dimensionRegexps, tt.args.resources, tt.args.metricsList, tt.args.dimensionNameRequirements, tt.args.m)
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

func Test_ensureLabelConsistencyForMetrics(t *testing.T) {
	value1 := 1.0
	metric1 := promutil.PrometheusMetric{
		Name:   aws.String("metric1"),
		Labels: map[string]string{"label1": "value1"},
		Value:  &value1,
	}

	value2 := 2.0
	metric2 := promutil.PrometheusMetric{
		Name:   aws.String("metric1"),
		Labels: map[string]string{"label2": "value2"},
		Value:  &value2,
	}

	value3 := 2.0
	metric3 := promutil.PrometheusMetric{
		Name:   aws.String("metric1"),
		Labels: map[string]string{},
		Value:  &value3,
	}

	metrics := []*promutil.PrometheusMetric{&metric1, &metric2, &metric3}
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

// StubClock stub implementation of Clock interface that allows tests
// to control time.Now()
type StubClock struct {
	currentTime time.Time
}

func (mt StubClock) Now() time.Time {
	return mt.currentTime
}

func Test_MetricWindow(t *testing.T) {
	type data struct {
		roundingPeriod    time.Duration
		length            time.Duration
		delay             time.Duration
		clock             StubClock
		expectedStartTime time.Time
		expectedEndTime   time.Time
	}

	testCases := []struct {
		testName string
		data     data
	}{
		{
			testName: "Go back four minutes and round to the nearest two minutes with two minute delay",
			data: data{
				roundingPeriod: 120 * time.Second,
				length:         120 * time.Second,
				delay:          120 * time.Second,
				clock: StubClock{
					currentTime: time.Date(2021, 11, 20, 0, 0, 0, 0, time.UTC),
				},
				expectedStartTime: time.Date(2021, 11, 19, 23, 56, 0, 0, time.UTC),
				expectedEndTime:   time.Date(2021, 11, 19, 23, 58, 0, 0, time.UTC),
			},
		},
		{
			testName: "Go back four minutes with two minute delay nad no rounding",
			data: data{
				roundingPeriod: 0,
				length:         120 * time.Second,
				delay:          120 * time.Second,
				clock: StubClock{
					currentTime: time.Date(2021, 1, 1, 0, 0o2, 22, 33, time.UTC),
				},
				expectedStartTime: time.Date(2020, 12, 31, 23, 58, 22, 33, time.UTC),
				expectedEndTime:   time.Date(2021, 1, 1, 0, 0, 22, 33, time.UTC),
			},
		},
		{
			testName: "Go back two days and round to the nearest day (midnight) with zero delay",
			data: data{
				roundingPeriod: 86400 * time.Second,  // 1 day
				length:         172800 * time.Second, // 2 days
				delay:          0,
				clock: StubClock{
					currentTime: time.Date(2021, 11, 20, 8, 33, 44, 0, time.UTC),
				},
				expectedStartTime: time.Date(2021, 11, 18, 0, 0, 0, 0, time.UTC),
				expectedEndTime:   time.Date(2021, 11, 20, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			testName: "Go back two days and round to the nearest 5 minutes with zero delay",
			data: data{
				roundingPeriod: 300 * time.Second,    // 5 min
				length:         172800 * time.Second, // 2 days
				delay:          0,
				clock: StubClock{
					currentTime: time.Date(2021, 11, 20, 8, 33, 44, 0, time.UTC),
				},
				expectedStartTime: time.Date(2021, 11, 18, 8, 30, 0, 0, time.UTC),
				expectedEndTime:   time.Date(2021, 11, 20, 8, 30, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			startTime, endTime := determineGetMetricDataWindow(tc.data.clock, tc.data.roundingPeriod, tc.data.length, tc.data.delay)
			if !startTime.Equal(tc.data.expectedStartTime) {
				t.Errorf("start time incorrect. Expected: %s, Actual: %s", tc.data.expectedStartTime.Format(timeFormat), startTime.Format(timeFormat))
				t.Errorf("end time incorrect. Expected: %s, Actual: %s", tc.data.expectedEndTime.Format(timeFormat), endTime.Format(timeFormat))
			}
		})
	}
}
