package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func UpdateMetrics(
	config ScrapeConf,
	registry *prometheus.Registry,
	metricsPerQuery int,
	labelsSnakeCase bool,
	cloudwatchSemaphore, tagSemaphore chan struct{},
	cache SessionCache,
) {
	tagsData, cloudwatchData := scrapeAwsData(
		config,
		metricsPerQuery,
		cloudwatchSemaphore,
		tagSemaphore,
		cache,
	)
	var metrics []*PrometheusMetric

	metrics = append(metrics, migrateCloudwatchToPrometheus(cloudwatchData, labelsSnakeCase)...)
	metrics = ensureLabelConsistencyForMetrics(metrics)

	metrics = append(metrics, migrateTagsToPrometheus(tagsData, labelsSnakeCase)...)

	registry.MustRegister(NewPrometheusCollector(metrics))
	for _, counter := range []prometheus.Counter{cloudwatchAPICounter, cloudwatchAPIErrorCounter, cloudwatchGetMetricDataAPICounter, cloudwatchGetMetricStatisticsAPICounter, resourceGroupTaggingAPICounter, autoScalingAPICounter, apiGatewayAPICounter, targetGroupsAPICounter, ec2APICounter, dmsAPICounter} {
		if err := registry.Register(counter); err != nil {
			log.Warning("Could not publish cloudwatch api metric")
		}
	}
}
