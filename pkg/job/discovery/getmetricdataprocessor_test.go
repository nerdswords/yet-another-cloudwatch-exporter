package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

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
						{ID: "metric-3", Datapoint: aws.Float64(15), Timestamp: time.Date(2023, time.June, 7, 3, 9, 8, 0, time.UTC)},
						{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
					},
					{
						{ID: "metric-4", Datapoint: aws.Float64(20), Timestamp: time.Date(2023, time.June, 7, 4, 9, 8, 0, time.UTC)},
					},
					{
						{ID: "metric-2", Datapoint: aws.Float64(12), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne", Namespace: "svc"},
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-2", Statistic: "Max"}, MetricName: "MetricTwo", Namespace: "svc"},
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-3", Statistic: "Sum"}, MetricName: "MetricThree", Namespace: "svc"},
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-4", Statistic: "Count"}, MetricName: "MetricFour", Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricName: "MetricOne",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricTwo",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Max",
						Datapoint: aws.Float64(12),
						Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricThree",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Sum",
						Datapoint: aws.Float64(15),
						Timestamp: time.Date(2023, time.June, 7, 3, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricFour",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Count",
						Datapoint: aws.Float64(20),
						Timestamp: time.Date(2023, time.June, 7, 4, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"duplicate results",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
						{ID: "metric-1", Datapoint: aws.Float64(15), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne", Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricName: "MetricOne",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"unexpected result ID",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
						{ID: "metric-2", Datapoint: aws.Float64(15), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne", Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricName: "MetricOne",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"nil metric data result",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
					},
					nil,
					{
						{ID: "metric-2", Datapoint: aws.Float64(12), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne", Namespace: "svc"},
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-2", Statistic: "Max"}, MetricName: "MetricTwo", Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricName: "MetricOne",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricTwo",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Max",
						Datapoint: aws.Float64(12),
						Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC),
					},
				},
			},
		},
		{
			"missing metric data result",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne", Namespace: "svc"},
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-2", Statistic: "Max"}, MetricName: "MetricTwo", Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricName: "MetricOne",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName:          "MetricTwo",
					Namespace:           "svc",
					GetMetricDataResult: nil,
				},
			},
		},
		{
			"nil metric datapoint",
			args{
				metricDataResults: [][]cloudwatch.MetricDataResult{
					{
						{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
						{ID: "metric-2"},
					},
				},
				cloudwatchDatas: []*model.CloudwatchData{
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne", Namespace: "svc"},
					{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-2", Statistic: "Max"}, MetricName: "MetricTwo", Namespace: "svc"},
				},
			},
			[]*model.CloudwatchData{
				{
					MetricName: "MetricOne",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricTwo",
					Namespace:  "svc",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Max",
						Datapoint: nil,
						Timestamp: time.Time{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapResultsToMetricDatas(tt.args.metricDataResults, tt.args.cloudwatchDatas, logging.NewNopLogger())
			// mapResultsToMetricDatas() modifies its []*model.CloudwatchData parameter in-place, assert that it was updated

			// Ensure processing params were nil'ed when expected to be
			for _, data := range tt.args.cloudwatchDatas {
				if data.GetMetricDataResult != nil {
					require.Nil(t, data.GetMetricDataProcessingParams, "GetMetricDataResult is not nil GetMetricDataProcessingParams should have been nil")
				} else {
					require.NotNil(t, data.GetMetricDataProcessingParams, "GetMetricDataResult is nil GetMetricDataProcessingParams should not have been nil")
				}

				// Drop processing params to simplify further asserts
				data.GetMetricDataProcessingParams = nil
			}
			require.ElementsMatch(t, tt.wantCloudwatchDatas, tt.args.cloudwatchDatas)
		})
	}
}

func getSampleMetricDatas(id string) *model.CloudwatchData {
	return &model.CloudwatchData{
		MetricName: "StorageBytes",
		Dimensions: []model.Dimension{
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
		MetricMigrationParams: model.MetricMigrationParams{
			NilToZero:              false,
			AddCloudwatchTimestamp: false,
		},
		GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{
			QueryID:   id,
			Period:    60,
			Length:    60,
			Delay:     0,
			Statistic: "Average",
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
				Datapoint: aws.Float64(1.4 * float64(batch)),
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
