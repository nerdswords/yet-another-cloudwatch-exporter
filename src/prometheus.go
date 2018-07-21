package main

import (
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	_ "sync/atomic"
)

type PrometheusData struct {
	name   *string
	labels map[string]string
	value  *float64
}

func createPrometheusMetrics(p PrometheusData) *prometheus.Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        *p.name,
		Help:        "Help is not implemented yet.",
		ConstLabels: p.labels,
	})

	gauge.Set(*p.value)

	return &gauge
}

func removePromDouble(data []*PrometheusData) []*PrometheusData {
	keys := make(map[string]bool)
	list := []*PrometheusData{}
	for _, entry := range data {
		if _, value := keys[*entry.name]; !value {
			keys[*entry.name] = true
			list = append(list, entry)
		}
	}
	return list
}

func fillRegistry(promData []*PrometheusData) *prometheus.Registry {
	registry := prometheus.NewRegistry()

	for _, point := range promData {
		gauge := createPrometheusMetrics(*point)
		registry.MustRegister(*gauge)
	}

	return registry
}

func PromString(text string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_")
	return replacer.Replace(text)
}
