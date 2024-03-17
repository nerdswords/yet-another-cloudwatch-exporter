package getmetricdata

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type cloudwatchDataInput struct {
	MetricName                    string
	GetMetricDataProcessingParams *model.GetMetricDataProcessingParams
}
type cloudwatchDataOutput struct {
	MetricName string
	*model.GetMetricDataResult
}

func TestProcessor_Run(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name                             string
		requests                         []*cloudwatchDataInput
		getMetricDataResponsesForQueries []cloudwatch.MetricDataResult
		want                             []cloudwatchDataOutput
		metricsPerBatch                  int
	}{
		{
			name: "successfully maps input to output when GetMetricData returns data",
			requests: []*cloudwatchDataInput{
				{MetricName: "metric-1", GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "1234", Statistic: "Average"}},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "1234", Datapoint: aws.Float64(1000), Timestamp: now}},
			want: []cloudwatchDataOutput{
				{MetricName: "metric-1", GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Average", Datapoint: aws.Float64(1000), Timestamp: now}},
			},
		},
		{
			name: "handles duplicate results",
			requests: []*cloudwatchDataInput{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne"},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
				{ID: "metric-1", Datapoint: aws.Float64(15), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
			},
			want: []cloudwatchDataOutput{
				{MetricName: "MetricOne", GetMetricDataResult: &model.GetMetricDataResult{
					Statistic: "Min",
					Datapoint: aws.Float64(5),
					Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
				}},
			},
		},
		{
			name: "handles duplicate results",
			requests: []*cloudwatchDataInput{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne"},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
				{ID: "metric-1", Datapoint: aws.Float64(15), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
			},
			want: []cloudwatchDataOutput{
				{MetricName: "MetricOne", GetMetricDataResult: &model.GetMetricDataResult{
					Statistic: "Min",
					Datapoint: aws.Float64(5),
					Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
				}},
			},
		},
		{
			name: "does not return a request when QueryID is not in MetricDataResult",
			requests: []*cloudwatchDataInput{
				{MetricName: "metric-1", GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "1234", Statistic: "Average"}},
				{MetricName: "metric-2", GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "make-me-disappear", Statistic: "Average"}},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "1234", Datapoint: aws.Float64(1000), Timestamp: now}},
			want: []cloudwatchDataOutput{
				{MetricName: "metric-1", GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Average", Datapoint: aws.Float64(1000), Timestamp: now}},
			},
		},
		{
			name: "does not return a request when QueryID is not in MetricDataResult",
			requests: []*cloudwatchDataInput{
				{MetricName: "metric-1", GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "1234", Statistic: "Average"}},
				{MetricName: "metric-2", GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "make-me-disappear", Statistic: "Average"}},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "1234", Datapoint: aws.Float64(1000), Timestamp: now}},
			want: []cloudwatchDataOutput{
				{MetricName: "metric-1", GetMetricDataResult: &model.GetMetricDataResult{Statistic: "Average", Datapoint: aws.Float64(1000), Timestamp: now}},
			},
		},
		{
			name: "maps nil metric datapoints",
			requests: []*cloudwatchDataInput{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne"},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-2", Statistic: "Max"}, MetricName: "MetricTwo"},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
				{ID: "metric-2"},
			},
			want: []cloudwatchDataOutput{
				{
					MetricName: "MetricOne",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricTwo",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Max",
						Datapoint: nil,
						Timestamp: time.Time{},
					},
				},
			},
		},
		{
			name:            "successfully maps input to output when multiple batches are involved",
			metricsPerBatch: 1,
			requests: []*cloudwatchDataInput{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-1", Statistic: "Min"}, MetricName: "MetricOne"},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-2", Statistic: "Max"}, MetricName: "MetricTwo"},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-3", Statistic: "Sum"}, MetricName: "MetricThree"},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{QueryID: "metric-4", Statistic: "Count"}, MetricName: "MetricFour"},
			},
			getMetricDataResponsesForQueries: []cloudwatch.MetricDataResult{
				{ID: "metric-1", Datapoint: aws.Float64(5), Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC)},
				{ID: "metric-2", Datapoint: aws.Float64(12), Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC)},
				{ID: "metric-3", Datapoint: aws.Float64(15), Timestamp: time.Date(2023, time.June, 7, 3, 9, 8, 0, time.UTC)},
				{ID: "metric-4", Datapoint: aws.Float64(20), Timestamp: time.Date(2023, time.June, 7, 4, 9, 8, 0, time.UTC)},
			},
			want: []cloudwatchDataOutput{
				{
					MetricName: "MetricOne",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Min",
						Datapoint: aws.Float64(5),
						Timestamp: time.Date(2023, time.June, 7, 1, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricTwo",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Max",
						Datapoint: aws.Float64(12),
						Timestamp: time.Date(2023, time.June, 7, 2, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricThree",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Sum",
						Datapoint: aws.Float64(15),
						Timestamp: time.Date(2023, time.June, 7, 3, 9, 8, 0, time.UTC),
					},
				},
				{
					MetricName: "MetricFour",
					GetMetricDataResult: &model.GetMetricDataResult{
						Statistic: "Count",
						Datapoint: aws.Float64(20),
						Timestamp: time.Date(2023, time.June, 7, 4, 9, 8, 0, time.UTC),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsPerQuery := 500
			if tt.metricsPerBatch != 0 {
				metricsPerQuery = tt.metricsPerBatch
			}
			r := NewProcessor(testClient{GetMetricDataResponsesForQueries: tt.getMetricDataResponsesForQueries}, metricsPerQuery, 1)
			cloudwatchData, err := r.Run(context.Background(), logging.NewNopLogger(), "anything_is_fine", 100, 100, aws.Int64(100), ToCloudwatchData(tt.requests))
			require.NoError(t, err)
			require.Len(t, cloudwatchData, len(tt.want))
			got := make([]cloudwatchDataOutput, 0, len(cloudwatchData))
			for _, data := range cloudwatchData {
				assert.Nil(t, data.GetMetricStatisticsResult)
				assert.Nil(t, data.GetMetricDataProcessingParams)
				assert.NotNil(t, data.GetMetricDataResult)
				got = append(got, cloudwatchDataOutput{
					MetricName:          data.MetricName,
					GetMetricDataResult: data.GetMetricDataResult,
				})
			}

			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestProcessor_Run_BatchesByMetricsPerQuery(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name                                 string
		metricsPerQuery                      int
		numberOfRequests                     int
		expectedNumberOfCallsToGetMetricData int32
	}{
		{name: "1 per batch", metricsPerQuery: 1, numberOfRequests: 10, expectedNumberOfCallsToGetMetricData: 10},
		{name: "divisible batches and requests", metricsPerQuery: 5, numberOfRequests: 100, expectedNumberOfCallsToGetMetricData: 20},
		{name: "indivisible batches and requests", metricsPerQuery: 5, numberOfRequests: 94, expectedNumberOfCallsToGetMetricData: 19},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCounter atomic.Int32
			getMetricDataFunc := func(_ context.Context, _ logging.Logger, requests []*model.CloudwatchData, _ string, _ int64, _ int64, _ *int64) []cloudwatch.MetricDataResult {
				callCounter.Add(1)
				response := make([]cloudwatch.MetricDataResult, 0, len(requests))
				for _, gmd := range requests {
					response = append(response, cloudwatch.MetricDataResult{
						ID:        gmd.GetMetricDataProcessingParams.QueryID,
						Datapoint: aws.Float64(1000),
						Timestamp: now,
					})
				}
				return response
			}

			requests := make([]*model.CloudwatchData, 0, tt.numberOfRequests)
			for i := 0; i < tt.numberOfRequests; i++ {
				requests = append(requests, getSampleMetricDatas(strconv.Itoa(i)))
			}
			r := Processor{
				metricsPerQuery: tt.metricsPerQuery,
				client:          testClient{GetMetricDataFunc: getMetricDataFunc},
				concurrency:     1,
			}
			cloudwatchData, err := r.Run(context.Background(), logging.NewNopLogger(), "anything_is_fine", 1, 1, aws.Int64(1), requests)
			require.NoError(t, err)
			assert.Len(t, cloudwatchData, tt.numberOfRequests)
			assert.Equal(t, tt.expectedNumberOfCallsToGetMetricData, callCounter.Load())
		})
	}
}

func ToCloudwatchData(input []*cloudwatchDataInput) []*model.CloudwatchData {
	output := make([]*model.CloudwatchData, 0, len(input))
	for _, i := range input {
		cloudwatchData := &model.CloudwatchData{
			MetricName:                    i.MetricName,
			ResourceName:                  "test",
			Namespace:                     "test",
			Tags:                          []model.Tag{{Key: "tag", Value: "value"}},
			Dimensions:                    []model.Dimension{{Name: "dimension", Value: "value"}},
			GetMetricDataProcessingParams: i.GetMetricDataProcessingParams,
			GetMetricDataResult:           nil,
			GetMetricStatisticsResult:     nil,
		}
		output = append(output, cloudwatchData)
	}
	return output
}

type testClient struct {
	GetMetricDataFunc                func(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []cloudwatch.MetricDataResult
	GetMetricDataResponsesForQueries []cloudwatch.MetricDataResult
}

func (t testClient) GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []cloudwatch.MetricDataResult {
	if t.GetMetricDataResponsesForQueries != nil {
		var result []cloudwatch.MetricDataResult
		for _, datum := range getMetricData {
			for _, response := range t.GetMetricDataResponsesForQueries {
				if datum.GetMetricDataProcessingParams.QueryID == response.ID {
					result = append(result, response)
				}
			}
		}
		return result
	}
	return t.GetMetricDataFunc(ctx, logger, getMetricData, namespace, length, delay, configuredRoundingPeriod)
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

func BenchmarkProcessorRun(b *testing.B) {
	type testcase struct {
		concurrency        int
		metricsPerQuery    int
		testResourcesCount int
		metricsPerResource int
	}

	for name, tc := range map[string]testcase{
		"small case": {
			concurrency:        10,
			metricsPerQuery:    500,
			testResourcesCount: 10,
		},
		"medium case": {
			concurrency:        10,
			metricsPerQuery:    500,
			testResourcesCount: 1000,
		},
		"big case": {
			concurrency:        10,
			metricsPerQuery:    500,
			testResourcesCount: 2000,
		},
	} {
		b.Run(name, func(b *testing.B) {
			doBench(b, tc.metricsPerQuery, tc.testResourcesCount, tc.concurrency)
		})
	}
}

func doBench(b *testing.B, metricsPerQuery, testResourcesCount int, concurrency int) {
	testResourceIDs := make([]string, testResourcesCount)
	for i := 0; i < testResourcesCount; i++ {
		testResourceIDs[i] = fmt.Sprintf("test-resource-%d", i)
	}

	client := testClient{GetMetricDataFunc: func(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []cloudwatch.MetricDataResult {
		b.StopTimer()
		results := make([]cloudwatch.MetricDataResult, 0, len(getMetricData))
		for _, entry := range getMetricData {
			results = append(results, cloudwatch.MetricDataResult{
				ID:        entry.GetMetricDataProcessingParams.QueryID,
				Datapoint: aws.Float64(1),
				Timestamp: time.Now(),
			})
		}
		b.StartTimer()
		return results
	}}

	for i := 0; i < b.N; i++ {
		// stop timer to not affect benchmark run
		// this has to do in every run, since running the processor mutates the metric datas slice
		b.StopTimer()
		datas := []*model.CloudwatchData{}
		for i := 0; i < testResourcesCount; i++ {
			datas = append(datas, getSampleMetricDatas(testResourceIDs[i]))
		}

		r := NewProcessor(client, metricsPerQuery, concurrency)
		// re-start timer
		b.ReportAllocs()
		b.StartTimer()

		//nolint:errcheck
		r.Run(context.Background(), logging.NewNopLogger(), "anything_is_fine", 100, 100, aws.Int64(100), datas)
	}
}
