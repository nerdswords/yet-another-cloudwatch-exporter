package getmetricdata

import (
	"context"
	"fmt"
	"math"
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

type Processor struct {
	metricsPerQuery  int
	client           Client
	concurrency      int
	windowCalculator MetricWindowCalculator
	logger           logging.Logger
}

func NewDefaultProcessor(logger logging.Logger, client Client, metricsPerQuery int, concurrency int) Processor {
	return NewProcessor(logger, client, metricsPerQuery, concurrency, MetricWindowCalculator{clock: TimeClock{}})
}

func NewProcessor(logger logging.Logger, client Client, metricsPerQuery int, concurrency int, windowCalculator MetricWindowCalculator) Processor {
	return Processor{
		logger:           logger,
		metricsPerQuery:  metricsPerQuery,
		client:           client,
		concurrency:      concurrency,
		windowCalculator: windowCalculator,
	}
}

func (p Processor) Run(ctx context.Context, namespace string, jobMetricLength, jobMetricDelay int64, jobRoundingPeriod *int64, requests []*model.CloudwatchData) ([]*model.CloudwatchData, error) {
	metricDataLength := len(requests)
	partitionSize := int(math.Ceil(float64(metricDataLength) / float64(p.metricsPerQuery)))
	p.logger.Debug("GetMetricData partitions", "size", partitionSize)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(p.concurrency)

	count := 0
	for i := 0; i < metricDataLength; i += p.metricsPerQuery {
		start := i
		end := i + p.metricsPerQuery
		if end > metricDataLength {
			end = metricDataLength
		}
		partitionNum := count
		count++

		g.Go(func() error {
			input := addQueryIDsToBatch(requests[start:end])

			var batchPeriod int64
			if jobRoundingPeriod == nil {
				for _, data := range input {
					if data.GetMetricDataProcessingParams.Period < batchPeriod {
						batchPeriod = data.GetMetricDataProcessingParams.Period
					}
				}
			} else {
				batchPeriod = *jobRoundingPeriod
			}

			startTime, endTime := p.windowCalculator.Calculate(toSecondDuration(batchPeriod), toSecondDuration(jobMetricLength), toSecondDuration(jobMetricDelay))
			if p.logger.IsDebugEnabled() {
				p.logger.Debug("GetMetricData Window", "start_time", startTime.Format(TimeFormat), "end_time", endTime.Format(TimeFormat))
			}

			data := p.client.GetMetricData(gCtx, input, namespace, startTime, endTime)
			if data != nil {
				mapResultsToBatch(p.logger, data, input)
			} else {
				p.logger.Warn("GetMetricData partition empty result", "start", start, "end", end, "partitionNum", partitionNum)
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

func toSecondDuration(i int64) time.Duration {
	return time.Duration(i) * time.Second
}
