package exporter

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var Metrics = []prometheus.Collector{cloudwatchAPICounter, cloudwatchAPIErrorCounter, cloudwatchGetMetricDataAPICounter, cloudwatchGetMetricStatisticsAPICounter, resourceGroupTaggingAPICounter, autoScalingAPICounter, targetGroupsAPICounter, apiGatewayAPICounter, ec2APICounter, dmsAPICounter}

type LabelSet map[string]struct{}

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
