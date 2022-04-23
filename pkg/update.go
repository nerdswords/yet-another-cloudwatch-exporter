package exporter

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Metrics is a slice of prometheus metrics specific to the scraping process such API call counters
var Metrics = []prometheus.Collector{cloudwatchAPICounter, cloudwatchAPIErrorCounter, cloudwatchGetMetricDataAPICounter, cloudwatchGetMetricStatisticsAPICounter, resourceGroupTaggingAPICounter, autoScalingAPICounter, targetGroupsAPICounter, apiGatewayAPICounter, ec2APICounter, dmsAPICounter}

type LabelSet map[string]struct{}

// UpdateMetrics can be used to scrape metrics from AWS on demand using the provided parameters. Scraped metrics will be added to the provided registry and
// any labels discovered during the scrape will be added to observedMetricLabels with their metric name as the key. Any errors encountered are not returned but
// will be logged and will either fail the scrape or a partial metric result will be added to the registry.
func UpdateMetrics(
	ctx context.Context,
	config ScrapeConf,
	registry *prometheus.Registry,
	metricsPerQuery int,
	labelsSnakeCase bool,
	cloudwatchSemaphore, tagSemaphore chan struct{},
	cache SessionCache,
	observedMetricLabels map[string]LabelSet,
) {
	tagsData, cloudwatchData := scrapeAwsData(
		ctx,
		config,
		metricsPerQuery,
		cloudwatchSemaphore,
		tagSemaphore,
		cache,
	)

	metrics, observedMetricLabels, err := migrateCloudwatchToPrometheus(cloudwatchData, labelsSnakeCase, observedMetricLabels)
	if err != nil {
		log.Printf("Error migrating cloudwatch metrics to prometheus metrics: %s\n", err.Error())
		return
	}
	metrics = ensureLabelConsistencyForMetrics(metrics, observedMetricLabels)

	metrics = append(metrics, migrateTagsToPrometheus(tagsData, labelsSnakeCase)...)

	registry.MustRegister(NewPrometheusCollector(metrics))
}
