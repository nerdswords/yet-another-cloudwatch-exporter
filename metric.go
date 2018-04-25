package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
)

func metrics(registry *prometheus.Registry, job job) {
	resources := describeResources(job.Discovery)

	for _, resource := range resources {
		metric := createInfoMetric(resource, job.Name, job.Discovery.ExportedTags)
		registry.MustRegister(metric)

		for _, metric := range job.Metrics {
			metric := createCloudwatchMetric(resource, metric)
			registry.MustRegister(metric)
		}
	}
}

func createCloudwatchMetric(resource *resourceWrapper, metric metric) prometheus.Gauge {
	value := getCloudwatchMetric(resource, metric)

	labels := prometheus.Labels{
		"id": *resource.Id,
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_" + *resource.Service + "_" + metric.Name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(value)

	return gauge
}

func createInfoMetric(resource *resourceWrapper, jobName string, exportedTags []string) prometheus.Gauge {
	promLabels := make(map[string]string)

	//promLabels := prometheus.Labels{"yace_aws_id": *resource.Id, "yace_aws_service": *resource.Service}

	for _, exportedTag := range exportedTags {
		for _, resourceTag := range resource.Tags {
			escapedKey := ConvertTagToLabel(exportedTag)
			if exportedTag == resourceTag.Key {
				promLabels[escapedKey] = resourceTag.Value
			} else {
				promLabels[escapedKey] = ""
			}
		}
	}

	promLabels["yace_name"] = jobName
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_" + *resource.Service + "_info",
		Help:        "Help is not implemented yet.",
		ConstLabels: promLabels,
	})

	return gauge
}
