package getmetricdata

import (
	"math"
	"time"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type WindowCalculator interface {
	Calculate(period time.Duration, length time.Duration, delay time.Duration) (time.Time, time.Time)
}

type iteratorFactory struct {
	metricsPerQuery  int
	windowCalculator WindowCalculator
}

func (b iteratorFactory) Build(data []*model.CloudwatchData, jobMetricLength, jobMetricDelay int64) BatchIterator {
	if len(data) == 0 {
		return nothingToIterate{}
	}

	params := data[0].GetMetricDataProcessingParams
	params.Length = jobMetricLength
	params.Delay = jobMetricDelay

	return NewSimpleBatchIterator(b.windowCalculator, b.metricsPerQuery, data, params)
}

type nothingToIterate struct{}

func (n nothingToIterate) Size() int {
	return 0
}

func (n nothingToIterate) Next() ([]*model.CloudwatchData, time.Time, time.Time) {
	return nil, time.Time{}, time.Time{}
}

func (n nothingToIterate) HasMore() bool {
	return false
}

type simpleBatchingIterator struct {
	size            int
	currentBatch    int
	startTime       time.Time
	endTime         time.Time
	data            []*model.CloudwatchData
	entriesPerBatch int
}

func (s *simpleBatchingIterator) Size() int {
	return s.size
}

func (s *simpleBatchingIterator) Next() ([]*model.CloudwatchData, time.Time, time.Time) {
	// We are out of data return defaults
	if s.currentBatch >= s.size {
		return nil, time.Time{}, time.Time{}
	}

	startingIndex := s.currentBatch * s.entriesPerBatch
	endingIndex := startingIndex + s.entriesPerBatch
	if endingIndex > len(s.data) {
		endingIndex = len(s.data)
	}

	// TODO are we technically doing this https://go.dev/wiki/SliceTricks#batching-with-minimal-allocation and if not
	// would it change allocations to do this ahead of time?
	result := s.data[startingIndex:endingIndex]
	s.currentBatch++
	return result, s.startTime, s.endTime
}

func (s *simpleBatchingIterator) HasMore() bool {
	return s.currentBatch < s.size
}

// NewSimpleBatchIterator returns an iterator which slices the data in place based on the metricsPerQuery.
func NewSimpleBatchIterator(windowCalculator WindowCalculator, metricsPerQuery int, data []*model.CloudwatchData, params *model.GetMetricDataProcessingParams) BatchIterator {
	startTime, endTime := windowCalculator.Calculate(
		time.Duration(params.Period)*time.Second,
		time.Duration(params.Length)*time.Second,
		time.Duration(params.Delay)*time.Second)

	size := int(math.Ceil(float64(len(data)) / float64(metricsPerQuery)))

	return &simpleBatchingIterator{
		size:            size,
		startTime:       startTime,
		endTime:         endTime,
		data:            data,
		entriesPerBatch: metricsPerQuery,
	}
}
