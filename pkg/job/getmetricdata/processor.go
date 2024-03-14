package getmetricdata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type Client interface {
	GetMetricData(ctx context.Context, getMetricData []*model.CloudwatchData, namespace string, startTime time.Time, endTime time.Time) []cloudwatch.MetricDataResult
}

type IteratorFactory interface {
	// Build returns an ideal batch iterator based on the provided CloudwatchData
	Build(requests []*model.CloudwatchData, jobMetricLength, jobMetricDelay int64) BatchIterator
}

type BatchIterator interface {
	// Size returns the number of batches in the iterator
	Size() int

	// Next returns the next batch of CloudWatch data be used when calling GetMetricData and the start + end time for
	// the GetMetricData call
	// If called when there are no more batches default values will be returned
	Next() ([]*model.CloudwatchData, time.Time, time.Time)

	// HasMore returns true if there are more batches to iterate otherwise false. Should be used in a loop
	// to govern calls to Next()
	HasMore() bool
}

type Processor struct {
	client      Client
	concurrency int
	logger      logging.Logger
	factory     IteratorFactory
}

func NewDefaultProcessor(logger logging.Logger, client Client, metricsPerQuery int, concurrency int) Processor {
	factory := iteratorFactory{
		metricsPerQuery:  metricsPerQuery,
		windowCalculator: MetricWindowCalculator{clock: TimeClock{}},
	}

	return NewProcessor(logger, client, factory, concurrency)
}

func NewProcessor(logger logging.Logger, client Client, factory IteratorFactory, concurrency int) Processor {
	return Processor{
		logger:      logger,
		factory:     factory,
		client:      client,
		concurrency: concurrency,
	}
}

func (b Processor) Run(ctx context.Context, namespace string, jobMetricLength, jobMetricDelay int64, requests []*model.CloudwatchData) ([]*model.CloudwatchData, error) {
	if len(requests) == 0 {
		return requests, nil
	}

	iterator := b.factory.Build(requests, jobMetricLength, jobMetricDelay)
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(b.concurrency)

	for iterator.HasMore() {
		batch, startTime, endTime := iterator.Next()
		g.Go(func() error {
			batch = addQueryIDsToBatch(batch)
			data := b.client.GetMetricData(gCtx, batch, namespace, startTime, endTime)
			if data != nil {
				mapResultsToBatch(b.logger, data, batch)
			} else {
				b.logger.Warn("GetMetricData empty result", "start", startTime, "end", endTime)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("GetMetricData work group error: %w", err)
	}

	// Remove unprocessed/unknown elements in place, if any. Since getMetricDatas
	// is a slice of pointers, the compaction can be easily done in-place.
	requests = compact(requests, func(m *model.CloudwatchData) bool {
		return m.GetMetricDataResult != nil
	})

	return requests, nil
}

func addQueryIDsToBatch(batch []*model.CloudwatchData) []*model.CloudwatchData {
	for i, entry := range batch {
		entry.GetMetricDataProcessingParams.QueryID = indexToQueryID(i)
	}

	return batch
}

func mapResultsToBatch(logger logging.Logger, results []cloudwatch.MetricDataResult, batch []*model.CloudwatchData) {
	for _, entry := range results {
		id, err := queryIDToIndex(entry.ID)
		if err != nil {
			logger.Warn("GetMetricData returned unknown Query ID", "err", err, "query_id", id)
			continue
		}
		if batch[id].GetMetricDataResult == nil {
			cloudwatchData := batch[id]
			cloudwatchData.GetMetricDataResult = &model.GetMetricDataResult{
				Statistic: cloudwatchData.GetMetricDataProcessingParams.Statistic,
				Datapoint: entry.Datapoint,
				Timestamp: entry.Timestamp,
			}

			// All GetMetricData processing is done clear the params
			cloudwatchData.GetMetricDataProcessingParams = nil
		}
	}
}

func indexToQueryID(i int) string {
	return fmt.Sprintf("id_%d", i)
}

func queryIDToIndex(queryID string) (int, error) {
	noID := strings.TrimPrefix(queryID, "id_")
	id, err := strconv.Atoi(noID)
	return id, err
}
