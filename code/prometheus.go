package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

func createPrometheusMetrics(resources []*awsResource, cloudwatch []*cloudwatchData) *prometheus.Registry {
	registry := prometheus.NewRegistry()

	exportedTags := findExportedTags(resources)

	pushCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_requests_total",
		Help: "Help is not implemented yet.",
	})
	registry.MustRegister(pushCounter)

	for _, r := range resources {
		metric := createInfoMetric(r, exportedTags[*r.Service])
		registry.MustRegister(metric)
	}

	for _, c := range cloudwatch {
		pushCounter.Inc()
		if c.Value != nil {
			metric := createCloudwatchMetric(*c)
			registry.MustRegister(metric)
		}
	}

	return registry
}

func createCloudwatchMetric(data cloudwatchData) prometheus.Gauge {
	labels := prometheus.Labels{
		"name": *data.Id,
	}

	name := "aws_" + strings.ToLower(*data.Service) + "_" + strings.ToLower(promString(*data.Metric)) + "_" + strings.ToLower(promString(*data.Statistics))

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(*data.Value)

	return gauge
}

func createInfoMetric(resource *awsResource, exportedTags []string) prometheus.Gauge {
	promLabels := make(map[string]string)

	promLabels["name"] = *resource.Id

	name := "aws_" + *resource.Service + "_info"

	for _, exportedTag := range exportedTags {
		escapedKey := "tag_" + promString(exportedTag)
		promLabels[escapedKey] = ""
		for _, resourceTag := range resource.Tags {
			if exportedTag == resourceTag.Key {
				promLabels[escapedKey] = resourceTag.Value
			}
		}
	}

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
