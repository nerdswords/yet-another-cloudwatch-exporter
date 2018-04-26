package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

func metrics(registry *prometheus.Registry, job job, exportedTags []string) {
	resources := describeResources(job.Discovery)

	for _, resource := range resources {
		metric := createInfoMetric(resource, &job.Name, exportedTags)
		registry.MustRegister(metric)

		for _, metric := range job.Metrics {
			metric := createCloudwatchMetric(resource, &job.Name, metric)
			registry.MustRegister(metric)
		}
	}
}

func createCloudwatchMetric(resource *resourceWrapper, jobName *string, metric metric) prometheus.Gauge {
	value := getCloudwatchMetric(resource, metric)

	labels := prometheus.Labels{
		"yace_name": *resource.Id,
		"yace_job":  *jobName,
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_" + *resource.Service + "_" + metric.Name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(value)

	return gauge
}

func createInfoMetric(resource *resourceWrapper, jobName *string, exportedTags []string) prometheus.Gauge {
	promLabels := make(map[string]string)

	for _, exportedTag := range exportedTags {
		escapedKey := convertTagToLabel(exportedTag)
		promLabels[escapedKey] = ""
		for _, resourceTag := range resource.Tags {
			if exportedTag == resourceTag.Key {
				promLabels[escapedKey] = resourceTag.Value
			}
		}
	}

	promLabels["yace_name"] = *jobName
	promLabels["id"] = *resource.Id

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_" + *resource.Service + "_info",
		Help:        "Help is not implemented yet.",
		ConstLabels: promLabels,
	})

	return gauge
}

func convertTagToLabel(label string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_")
	saveLabel := replacer.Replace(label)
	return "tag_" + saveLabel
}
