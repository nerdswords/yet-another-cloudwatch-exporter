package getmetricdata

import (
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestIteratorFactory_Build(t *testing.T) {
	tests := []struct {
		name             string
		input            []*model.CloudwatchData
		expectedIterator Iterator
	}{
		{
			name:             "empty returns nothing to iterator",
			input:            []*model.CloudwatchData{},
			expectedIterator: nothingToIterate{},
		},
		{
			name: "input with data returns simple batching",
			input: []*model.CloudwatchData{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
			},
			expectedIterator: &simpleBatchingIterator{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			factory := iteratorFactory{100}
			iterator := factory.Build(tc.input, 10, 100, aws.Int64(100))
			assert.IsType(t, tc.expectedIterator, iterator)
		})
	}
}

func TestSimpleBatchingIterator_SetsLengthAndDelay(t *testing.T) {
	data := []*model.CloudwatchData{
		{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 101, Delay: 100}},
	}
	jobLength := int64(101)
	jobDelay := int64(100)
	iterator := NewSimpleBatchIterator(1, data, jobLength, jobDelay, nil)
	_, params := iterator.Next()
	assert.Equal(t, jobLength, params.Length)
	assert.Equal(t, jobDelay, params.Delay)
}

func TestSimpleBatchingIterator_CalculatesPeriod(t *testing.T) {
	tests := []struct {
		name                        string
		data                        []*model.CloudwatchData
		metricsPerQuery             int
		roundingPeriod              *int64
		iterateCallToExpectedPeriod map[int]int64
	}{
		{
			name: "rounding period overrides all",
			data: []*model.CloudwatchData{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 200}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 200}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 200}},
			},
			metricsPerQuery:             1,
			roundingPeriod:              aws.Int64(100),
			iterateCallToExpectedPeriod: map[int]int64{1: 100, 2: 100, 3: 100},
		},
		{
			name: "uses metric period when no rounding period is set",
			data: []*model.CloudwatchData{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 200}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 200}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 200}},
			},
			metricsPerQuery:             1,
			roundingPeriod:              nil,
			iterateCallToExpectedPeriod: map[int]int64{1: 200, 2: 200, 3: 200},
		},
		{
			name: "smallest period wins",
			data: []*model.CloudwatchData{
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 1000}},
			},
			metricsPerQuery:             3,
			roundingPeriod:              nil,
			iterateCallToExpectedPeriod: map[int]int64{1: 10},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iterator := NewSimpleBatchIterator(tc.metricsPerQuery, tc.data, 10, 100, tc.roundingPeriod)

			numberOfCallsToNext := 0
			for iterator.HasMore() {
				numberOfCallsToNext++
				_, params := iterator.Next()
				assert.Equal(t, tc.iterateCallToExpectedPeriod[numberOfCallsToNext], params.Period)
			}
		})
	}
}

func TestSimpleBatchingIterator_IterateFlow(t *testing.T) {
	tests := []struct {
		name                               string
		metricsPerQuery                    int
		lengthOfCloudwatchData             int
		expectedSizeAndNumberOfCallsToNext int
	}{
		{
			name:                               "1 per batch",
			metricsPerQuery:                    1,
			lengthOfCloudwatchData:             10,
			expectedSizeAndNumberOfCallsToNext: 10,
		},
		{
			name:                               "divisible batches and requests",
			metricsPerQuery:                    5,
			lengthOfCloudwatchData:             100,
			expectedSizeAndNumberOfCallsToNext: 20,
		},
		{
			name:                               "indivisible batches and requests",
			metricsPerQuery:                    5,
			lengthOfCloudwatchData:             94,
			expectedSizeAndNumberOfCallsToNext: 19,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]*model.CloudwatchData, 0, tc.lengthOfCloudwatchData)
			for i := 0; i < tc.lengthOfCloudwatchData; i++ {
				data = append(data, getSampleMetricDatas(strconv.Itoa(i)))
			}
			iterator := NewSimpleBatchIterator(tc.metricsPerQuery, data, data[0].GetMetricDataProcessingParams.Length, data[0].GetMetricDataProcessingParams.Delay, nil)

			numberOfCallsToNext := 0
			for iterator.HasMore() {
				numberOfCallsToNext++
				iterator.Next()
			}

			assert.Equal(t, tc.expectedSizeAndNumberOfCallsToNext, numberOfCallsToNext)
		})
	}
}
