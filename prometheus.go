package main

import (
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	cloudwatchAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_requests_total",
		Help: "Help is not implemented yet.",
	})
)

type PrometheusMetric struct {
	name             *string
	labels           map[string]string
	value            *float64
	includeTimestamp bool
	timestamp        time.Time
}

type PrometheusCollector struct {
	metrics []*PrometheusMetric
}

func NewPrometheusCollector(metrics []*PrometheusMetric) *PrometheusCollector {
	return &PrometheusCollector{
		metrics: removeDuplicatedMetrics(metrics),
	}
}

func (p *PrometheusCollector) Describe(descs chan<- *prometheus.Desc) {
	for _, metric := range p.metrics {
		descs <- createDesc(metric)
	}
}

func (p *PrometheusCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, metric := range p.metrics {
		metrics <- createMetric(metric)
	}
}

func createDesc(metric *PrometheusMetric) *prometheus.Desc {
	return prometheus.NewDesc(
		*metric.name,
		"Help is not implemented yet.",
		nil,
		metric.labels,
	)
}

func createMetric(metric *PrometheusMetric) prometheus.Metric {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        *metric.name,
		Help:        "Help is not implemented yet.",
		ConstLabels: metric.labels,
	})

	gauge.Set(*metric.value)

	if !metric.includeTimestamp {
		return gauge
	}

	return prometheus.NewMetricWithTimestamp(metric.timestamp, gauge)
}

func removeDuplicatedMetrics(metrics []*PrometheusMetric) []*PrometheusMetric {
	keys := make(map[string]bool)
	filteredMetrics := []*PrometheusMetric{}
	for _, metric := range metrics {
		check := *metric.name + metric.labels["name"]
		if _, value := keys[check]; !value {
			keys[check] = true
			filteredMetrics = append(filteredMetrics, metric)
		}
	}
	return filteredMetrics
}

func promString(text string) string {
	text = splitString(text)
	return replaceWithUnderscores(text)
}

func promStringTag(text string) string {
	return replaceWithUnderscores(text)
}

func replaceWithUnderscores(text string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_", ":", "_", "=", "_")
	return replacer.Replace(text)
}

func splitString(text string) string {
	splitRegexp := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	return splitRegexp.ReplaceAllString(text, `$1.$2`)
}
