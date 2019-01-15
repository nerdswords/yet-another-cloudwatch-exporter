package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
	"strings"
)

var (
	cloudwatchAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_requests_total",
		Help: "Help is not implemented yet.",
	})
)

type prometheusData struct {
	name   *string
	labels map[string]string
	value  *float64
}

func createPrometheusMetrics(p prometheusData) *prometheus.Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        *p.name,
		Help:        "Help is not implemented yet.",
		ConstLabels: p.labels,
	})

	gauge.Set(*p.value)

	return &gauge
}

func removePromDouble(data []*prometheusData) []*prometheusData {
	keys := make(map[string]bool)
	list := []*prometheusData{}
	for _, entry := range data {
		check := *entry.name + entry.labels["name"]
		if _, value := keys[check]; !value {
			keys[check] = true
			list = append(list, entry)
		}
	}
	return list
}

func fillRegistry(promData []*prometheusData) *prometheus.Registry {
	registry := prometheus.NewRegistry()

	for _, point := range promData {
		gauge := createPrometheusMetrics(*point)

		if err := registry.Register(*gauge); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
				fmt.Println("Already registered")
			} else {
				panic(err)
			}
		}
	}

	return registry
}

func promString(text string) string {
	text = splitString(text)
	return replaceWithUnderscores(text)
}

func promStringTag(text string) string {
	return replaceWithUnderscores(text)
}

func replaceWithUnderscores(text string) string {
	replacer := strings.NewReplacer(" ", "_", ",", "_", "\t", "_", ",", "_", "/", "_", "\\", "_", ".", "_", "-", "_", ":", "_")
	return replacer.Replace(text)
}

func splitString(text string) string {
	splitRegexp := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	return splitRegexp.ReplaceAllString(text, `$1.$2`)
}
