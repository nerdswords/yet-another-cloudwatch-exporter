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
