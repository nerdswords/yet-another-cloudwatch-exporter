package getmetricdata

import (
	"math"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type iteratorFactory struct {
	metricsPerQuery int
}

func (b iteratorFactory) Build(data []*model.CloudwatchData, jobMetricLength, jobMetricDelay int64, jobRoundingPeriod *int64) Iterator {
	if len(data) == 0 {
		return nothingToIterate{}
	}

	return NewSimpleBatchIterator(b.metricsPerQuery, data, jobMetricLength, jobMetricDelay, jobRoundingPeriod)
}

type nothingToIterate struct{}

func (n nothingToIterate) Next() ([]*model.CloudwatchData, *model.GetMetricDataProcessingParams) {
	return nil, nil
}

func (n nothingToIterate) HasMore() bool {
	return false
}

type simpleBatchingIterator struct {
	size            int
	currentBatch    int
	data            []*model.CloudwatchData
	entriesPerBatch int
	roundingPeriod  *int64
	delay           int64
	length          int64
}

func (s *simpleBatchingIterator) Next() ([]*model.CloudwatchData, *model.GetMetricDataProcessingParams) {
	// We are out of data return defaults
	if s.currentBatch >= s.size {
		return nil, nil
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

	batchPeriod := model.DefaultPeriodSeconds
	if s.roundingPeriod == nil {
		for _, data := range result {
			if data.GetMetricDataProcessingParams.Period < batchPeriod {
				batchPeriod = data.GetMetricDataProcessingParams.Period
			}
		}
	} else {
		batchPeriod = *s.roundingPeriod
	}

	params := &model.GetMetricDataProcessingParams{
		Length: s.length,
		Delay:  s.delay,
		Period: batchPeriod,
	}

	return result, params
}

func (s *simpleBatchingIterator) HasMore() bool {
	return s.currentBatch < s.size
}

// NewSimpleBatchIterator returns an iterator which slices the data in place based on the metricsPerQuery.
func NewSimpleBatchIterator(metricsPerQuery int, data []*model.CloudwatchData, jobMetricLength, jobMetricDelay int64, jobRoundingPeriod *int64) Iterator {
	return &simpleBatchingIterator{
		size:            int(math.Ceil(float64(len(data)) / float64(metricsPerQuery))),
		length:          jobMetricLength,
		delay:           jobMetricDelay,
		roundingPeriod:  jobRoundingPeriod,
		data:            data,
		entriesPerBatch: metricsPerQuery,
	}
}
