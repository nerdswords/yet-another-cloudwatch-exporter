package getmetricdata

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestIteratorFactory_Build(t *testing.T) {
	tests := []struct {
		name             string
		input            []*model.CloudwatchData
		expectedIterator BatchIterator
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
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
				{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 10, Delay: 100}},
			},
			expectedIterator: &simpleBatchingIterator{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			factory := iteratorFactory{100, MetricWindowCalculator{clock: TimeClock{}}}
			iterator := factory.Build(tc.input, 10, 100)
			assert.IsType(t, tc.expectedIterator, iterator)
		})
	}
}

type testWindowCalculator struct {
	startTime time.Time
	endTime   time.Time
}

func (t testWindowCalculator) Calculate(time.Duration, time.Duration, time.Duration) (time.Time, time.Time) {
	return t.startTime, t.endTime
}

func TestSimpleBatchingIterator_SetsStartAndEndTime(t *testing.T) {
	calc := testWindowCalculator{
		startTime: time.Now().Truncate(time.Second).Add(-time.Second * 5),
		endTime:   time.Now().Truncate(time.Second),
	}
	data := []*model.CloudwatchData{
		{GetMetricDataProcessingParams: &model.GetMetricDataProcessingParams{Period: 101, Delay: 100}},
	}
	iterator := NewSimpleBatchIterator(calc, 1, data, data[0].GetMetricDataProcessingParams)
	_, startTime, endTime := iterator.Next()
	assert.Equal(t, calc.startTime, startTime)
	assert.Equal(t, calc.endTime, endTime)
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
			iterator := NewSimpleBatchIterator(testWindowCalculator{}, tc.metricsPerQuery, data, data[0].GetMetricDataProcessingParams)
			assert.Equal(t, tc.expectedSizeAndNumberOfCallsToNext, iterator.Size())

			numberOfCallsToNext := 0
			for iterator.HasMore() {
				numberOfCallsToNext++
				iterator.Next()
			}

			assert.Equal(t, tc.expectedSizeAndNumberOfCallsToNext, numberOfCallsToNext)
		})
	}
}
