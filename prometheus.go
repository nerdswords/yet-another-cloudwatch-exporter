package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

func metrics(registry *prometheus.Registry, job job, exportedTags []string) {
	resources := describeResources(job.Discovery)

	for _, resource := range resources.Resources {
		metric := createInfoMetric(resource, &job.Name, exportedTags)
		registry.MustRegister(metric)

		for _, metric := range job.Metrics {
			metric := createCloudwatchMetric(resource, resources.CloudwatchInfo, &job.Name, metric)
			registry.MustRegister(metric)
		}
	}
}

func createCloudwatchMetric(resource *awsResource, cloudwatchInfo *cloudwatchInfo, jobName *string, metric metric) prometheus.Gauge {
	value := getCloudwatchMetric(resource, cloudwatchInfo, metric)

	labels := prometheus.Labels{
		"yace_name": *resource.Id,
		"yace_job":  *jobName,
	}

	name := "yace_" + *resource.Service + "_" + promString(metric.Name)

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(value)

	return gauge
}

func createInfoMetric(resource *awsResource, jobName *string, exportedTags []string) prometheus.Gauge {
	promLabels := make(map[string]string)

	for _, exportedTag := range exportedTags {
		escapedKey := "tag_" + promString(exportedTag)
		promLabels[escapedKey] = ""
		for _, resourceTag := range resource.Tags {
			if exportedTag == resourceTag.Key {
				promLabels[escapedKey] = resourceTag.Value
			}
		}
	}

	promLabels["yace_name"] = *jobName
	promLabels["id"] = *resource.Id

	name := "yace_" + *resource.Service + "_info"

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
