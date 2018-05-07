package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

func preparePrometheusData(registry *prometheus.Registry) {
}

func createCloudwatchMetric(data cloudwatchData) prometheus.Gauge {
	labels := prometheus.Labels{
		"name": *data.Id,
	}

	name := "aws_" + *data.Service + "_" + promString(*data.Metric) + "_" + promString(*data.Statistics)

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(*data.Value)

	return gauge
}

func createInfoMetric(resource *awsResource) prometheus.Gauge {
	promLabels := make(map[string]string)

	promLabels["name"] = *resource.Id

	name := "aws_" + *resource.Service + "_info"

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        "Help is not implemented yet.",
		ConstLabels: promLabels,
	})

	return gauge
}

func promString(text string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_")
	return replacer.Replace(text)
}
