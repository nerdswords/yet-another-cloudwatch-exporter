package getmetricdata

import (
	"context"
	"fmt"
	"math"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type Client interface {
	GetMetricData(ctx context.Context, logger logging.Logger, getMetricData []*model.CloudwatchData, namespace string, length int64, delay int64, configuredRoundingPeriod *int64) []cloudwatch.MetricDataResult
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

func (p Processor) Run(ctx context.Context, logger logging.Logger, namespace string, jobMetricLength int64, jobMetricDelay int64, jobRoundingPeriod *int64, requests []*model.CloudwatchData) ([]*model.CloudwatchData, error) {
	metricDataLength := len(requests)
	partitionSize := int(math.Ceil(float64(metricDataLength) / float64(p.metricsPerQuery)))
	logger.Debug("GetMetricData partitions", "size", partitionSize)
	getMetricDataOutput := make([][]cloudwatch.MetricDataResult, 0, partitionSize)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(p.concurrency)
	mu := sync.Mutex{}
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
			input := requests[start:end]
			data := p.client.GetMetricData(gCtx, logger, input, namespace, jobMetricLength, jobMetricDelay, jobRoundingPeriod)
			if data != nil {
				mu.Lock()
				getMetricDataOutput = append(getMetricDataOutput, data)
				mu.Unlock()
			} else {
				logger.Warn("GetMetricData partition empty result", "start", start, "end", end, "partitionNum", partitionNum)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("GetMetricData work group error: %w", err)
	}

	mapResultsToMetricDatas(getMetricDataOutput, requests, logger)

	// Remove unprocessed/unknown elements in place, if any. Since getMetricDatas
	// is a slice of pointers, the compaction can be easily done in-place.
	requests = compact(requests, func(m *model.CloudwatchData) bool {
		return m.GetMetricDataResult != nil
	})

	return requests, nil
}

func mapResultsToMetricDatas(output [][]cloudwatch.MetricDataResult, datas []*model.CloudwatchData, logger logging.Logger) {
	// queryIDToData is a support structure used to easily find via a QueryID, the corresponding
	// model.CloudatchData.
	queryIDToData := make(map[string]*model.CloudwatchData, len(datas))

	// load the index
	for _, data := range datas {
		queryIDToData[data.GetMetricDataProcessingParams.QueryID] = data
	}

	// Update getMetricDatas slice with values and timestamps from API response.
	// We iterate through the response MetricDataResults and match the result ID
	// with what was sent in the API request.
	// In the event that the API response contains any ID we don't know about
	// (shouldn't really happen) we log a warning and move on. On the other hand,
	// in case the API response does not contain results for all the IDs we've
	// requested, unprocessed elements will be removed later on.
	for _, data := range output {
		if data == nil {
			continue
		}
		for _, metricDataResult := range data {
			// find into index
			metricData, ok := queryIDToData[metricDataResult.ID]
			if !ok {
				logger.Warn("GetMetricData returned unknown metric ID", "metric_id", metricDataResult.ID)
				continue
			}
			// skip elements that have been already mapped but still exist in queryIDToData
			if metricData.GetMetricDataResult != nil {
				continue
			}
			metricData.GetMetricDataResult = &model.GetMetricDataResult{
				Statistic: metricData.GetMetricDataProcessingParams.Statistic,
				Datapoint: metricDataResult.Datapoint,
				Timestamp: metricDataResult.Timestamp,
			}
			// All GetMetricData processing is done clear the params
			metricData.GetMetricDataProcessingParams = nil
		}
	}
}
