package main

import (
	_ "fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
)

func ec2Metrics(registry *prometheus.Registry, job job) {
	instances := describeInstances(job.Discovery.SearchTags)

	for _, instance := range instances {
		metric := createEC2InfoMetric(instance, job.Name, job.Discovery.ExportedTags)
		registry.MustRegister(metric)

		for _, metric := range job.Metrics {
			metric := createEC2CloudwatchMetric(instance.InstanceId, metric)
			registry.MustRegister(metric)
		}
	}
}

func createEC2CloudwatchMetric(instanceId *string, metric metric) prometheus.Gauge {
	value := getCloudwatchMetric("InstanceId", instanceId, "AWS/EC2", metric)

	labels := prometheus.Labels{
		"id": *instanceId,
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_ec2_" + metric.Name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(value)

	return gauge
}

func createEC2InfoMetric(instance *ec2.Instance, jobName string, exportedTags []string) prometheus.Gauge {
	ec2Labels := make(map[string]string)

	for _, tag := range instance.Tags {
		ec2Labels[*tag.Key] = *tag.Value
	}

	labels := prometheus.Labels{"id": *instance.InstanceId}

	for _, label := range exportedTags {
		labelName := ConvertTagToLabel(label)
		labels[labelName] = ec2Labels[label]
	}

	labels["yace_name"] = jobName

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_ec2_info",
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	return gauge
}
