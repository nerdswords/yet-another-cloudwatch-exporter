package exporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func UpdateMetrics(
	config ScrapeConf,
	registry *prometheus.Registry,
	now time.Time,
	metricsPerQuery int,
	fips, floatingTimeWindow, labelsSnakeCase bool,
	cloudwatchSemaphore, tagSemaphore chan struct{},
	cache SessionCache,
) time.Time {
	tagsData, cloudwatchData, endtime := scrapeAwsData(
		config,
		now,
		metricsPerQuery,
		fips, floatingTimeWindow,
		cloudwatchSemaphore, tagSemaphore,
		cache,
	)
	var metrics []*PrometheusMetric

	metrics = append(metrics, migrateCloudwatchToPrometheus(cloudwatchData, labelsSnakeCase)...)
	metrics = ensureLabelConsistencyForMetrics(metrics)

	metrics = append(metrics, migrateTagsToPrometheus(tagsData, labelsSnakeCase)...)

	registry.MustRegister(NewPrometheusCollector(metrics))
	for _, counter := range []prometheus.Counter{cloudwatchAPICounter, cloudwatchGetMetricDataAPICounter, cloudwatchGetMetricStatisticsAPICounter, resourceGroupTaggingAPICounter, autoScalingAPICounter, apiGatewayAPICounter, targetGroupsAPICounter} {
		if err := registry.Register(counter); err != nil {
			log.Warning("Could not publish cloudwatch api metric")
		}
	}
	return *endtime
}
