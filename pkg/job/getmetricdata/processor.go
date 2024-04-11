package getmetricdata

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type Client interface {
	GetMetricData(ctx context.Context, logger logging.Logger, data []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64, addHistoricalMetrics bool) []cloudwatch.MetricDataResult
}

type Processor struct {
	metricsPerQuery int
	client          Client
	concurrency     int
}

func NewProcessor(client Client, metricsPerQuery int, concurrency int) Processor {
	return Processor{
		metricsPerQuery: metricsPerQuery,
		client:          client,
		concurrency:     concurrency,
	}
}

func (p Processor) Run(ctx context.Context, logger logging.Logger, namespace string, jobMetricLength int64, jobMetricDelay int64, jobRoundingPeriod *int64, requests []*model.CloudwatchData, addHistoricalMetrics bool) ([]*model.CloudwatchData, error) {
	metricDataLength := len(requests)
	partitionSize := int(math.Ceil(float64(metricDataLength) / float64(p.metricsPerQuery)))
	logger.Debug("GetMetricData partitions", "size", partitionSize)

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
			data := p.client.GetMetricData(gCtx, logger, input, namespace, jobMetricLength, jobMetricDelay, jobRoundingPeriod, addHistoricalMetrics)
			if data != nil {
				mapResultsToBatch(logger, data, input, addHistoricalMetrics)
			} else {
				logger.Warn("GetMetricData partition empty result", "start", start, "end", end, "partitionNum", partitionNum)
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

func mapResultsToBatch(logger logging.Logger, results []cloudwatch.MetricDataResult, batch []*model.CloudwatchData, addHistoricalMetrics bool) {
	previousIdx := -1
	previousID := ""
	for _, entry := range results {
		//id, err := queryIDToIndex(entry.ID)
		//if err != nil {
		//	logger.Warn("GetMetricData returned unknown Query ID", "err", err, "query_id", id)
		//	continue
		//}
		idx := findGetMetricDataByID(batch, entry.ID)
		//if batch[id].GetMetricDataResult == nil || addHistoricalMetrics {
		if idx == -1 {
			if addHistoricalMetrics {
				if previousIdx != -1 && previousID == entry.ID {
					cloudwatchData := *batch[previousIdx]
					cloudwatchData.GetMetricDataResult = &model.GetMetricDataResult{
						Statistic: cloudwatchData.GetMetricDataProcessingParams.Statistic,
						Datapoint: entry.Datapoint,
						Timestamp: entry.Timestamp,
					}
					batch = append(batch, &cloudwatchData)
				} else {
					logger.Warn("GetMetricData returned unknown metric ID", "metric_id", entry.ID)
				}
			} else {
				logger.Warn("GetMetricData returned unknown metric ID", "metric_id", entry.ID)
			}
		}
		batch[idx].GetMetricDataResult = &model.GetMetricDataResult{
			Statistic: batch[idx].GetMetricDataProcessingParams.Statistic,
			Datapoint: entry.Datapoint,
			Timestamp: entry.Timestamp,
		}
		// All GetMetricData processing is done clear the params
		batch[idx].GetMetricDataProcessingParams = nil
		previousIdx = idx
		previousID = entry.ID

		//}

	}
}

func findGetMetricDataByID(getMetricDatas []*model.CloudwatchData, value string) int {
	for i := 0; i < len(getMetricDatas); i++ {
		if getMetricDatas[i].GetMetricDataProcessingParams == nil {
			continue // skip elements that have been already marked
		}
		if getMetricDatas[i].GetMetricDataProcessingParams.QueryID == value {
			return i
		}
	}
	return -1
}

func indexToQueryID(i int) string {
	return fmt.Sprintf("id_%d", i)
}

func queryIDToIndex(queryID string) (int, error) {
	noID := strings.TrimPrefix(queryID, "id_")
	id, err := strconv.Atoi(noID)
	return id, err
}
