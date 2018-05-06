package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

func metrics(registry *prometheus.Registry, job job, exportedTags []string, exportedAttributes []string) {
	resources := describeResources(job.Discovery)

	for _, resource := range resources.Resources {
		metric := createInfoMetric(resource, &job.Name, exportedTags, exportedAttributes)
		registry.MustRegister(metric)

		for _, metric := range job.Metrics {
			metric := createCloudwatchMetric(resource, &job.Name, metric)
			registry.MustRegister(metric)
		}
	}
}

func createCloudwatchMetric(resource *awsResource, jobName *string, metric metric) prometheus.Gauge {
	value := getCloudwatchMetric(resource, metric)

	labels := prometheus.Labels{
		"name": *resource.Id,
		"job":  *jobName,
	}

	name := *resource.Service + "_" + promString(metric.Name)

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(value)

	return gauge
}

func createInfoMetric(resource *awsResource, jobName *string, exportedTags []string, exportedAttributes []string) prometheus.Gauge {
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

	for _, exportedAttribute := range exportedAttributes {
		escapedKey := "attribute_" + promString(exportedAttribute)
		promLabels[escapedKey] = *resource.Attributes[exportedAttribute]
	}

	promLabels["job"] = *jobName
	promLabels["name"] = *resource.Id

	name := *resource.Service + "_info"

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
