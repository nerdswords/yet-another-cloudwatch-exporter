package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

func ec2Metrics(registry *prometheus.Registry, checks []Check) {
	for _, check := range checks {
		tags := check.tags
		instances := describeInstances(tags)

		for _, instance := range instances {
			for _, metric := range check.metrics {
				value := getCloudwatchMetricEC2(instance, metric)

				labels := prometheus.Labels{}

				for _, tag := range instance.Tags {
					labels[*tag.Key] = *tag.Value
				}

				gauge := prometheus.NewGauge(prometheus.GaugeOpts{
					Name:        metric.name,
					Help:        "Help is not implemented yet.",
					ConstLabels: labels,
				})

				gauge.Set(value)

				registry.MustRegister(gauge)
			}
		}
	}
}
