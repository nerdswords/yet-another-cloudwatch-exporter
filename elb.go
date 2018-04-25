package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
)

func elbMetrics(registry *prometheus.Registry, job job) {
	fmt.Println("Started an elb job")
	elbs := describeLoadBalancers() //tags)

	for _, elb := range elbs {
		if FilterELBThroughTags(elb.Tags, job.Discovery.SearchTags) {
			metric := createELBInfoMetric(elb, job.Name, job.Discovery.ExportedTags)
			registry.MustRegister(metric)

			for _, metric := range job.Metrics {
				metric := createELBCloudwatchMetric(elb.Elb.LoadBalancerName, metric)
				registry.MustRegister(metric)
			}
		}
	}
}

func createELBCloudwatchMetric(loadBalancerName *string, metric metric) prometheus.Gauge {
	value := getCloudwatchMetric("LoadBalancerName", loadBalancerName, "AWS/ELB", metric)

	labels := prometheus.Labels{
		"id": *loadBalancerName,
	}

	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_elb_" + metric.Name,
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	gauge.Set(value)

	return gauge
}

func createELBInfoMetric(e ElbWrapper, jobName string, exportedTags []string) prometheus.Gauge {
	elbLabels := make(map[string]string)

	for _, tag := range e.Tags {
		elbLabels[*tag.Key] = *tag.Value
	}

	labels := prometheus.Labels{"LoadBalancerName": *e.Elb.LoadBalancerName}

	for _, label := range exportedTags {
		labelName := ConvertTagToLabel(label)
		labels[labelName] = elbLabels[label]
	}

	labels["yace_name"] = jobName
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "yace_elb_info",
		Help:        "Help is not implemented yet.",
		ConstLabels: labels,
	})

	return gauge
}
