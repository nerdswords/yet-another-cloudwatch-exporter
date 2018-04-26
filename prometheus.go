package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

func metrics(registry *prometheus.Registry, job job) {
	resources := describeResources(job.Discovery)

	for _, resource := range resources {
		metric := createInfoMetric(resource, job.Name, job.Discovery.ExportedTags)
		registry.MustRegister(metric)

		for _, metric := range job.Metrics {
			fmt.Println(metric)
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
		escapedKey := ConvertTagToLabel(exportedTag)
		promLabels[escapedKey] = ""
		for _, resourceTag := range resource.Tags {
			if exportedTag == resourceTag.Key {
				promLabels[escapedKey] = resourceTag.Value
			}
		}
	}

	promLabels["name"] = jobName
	promLabels["id"] = *resource.Id

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_" + *resource.Service + "_info",
		Help:        "Help is not implemented yet.",
		ConstLabels: promLabels,
	})

	return gauge
}
