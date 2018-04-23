package main

import (
	_ "encoding/json"
	"flag"
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	_ "strings"
)

type Config struct {
	checks []Check
}

type Tag struct {
	key   string
	value string
}

type Metric struct {
	name       string
	statistics string
	period     int
}

type Check struct {
	name    string
	tags    []Tag
	metrics []Metric
}

var (
	addr       = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile = flag.String("config.file", "config.yml", "Path to configuration file.")
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {

	//yamlParse := readConfig
	tag := Tag{key: "Name", value: "test-name"}
	tags := []Tag{tag}
	metric := Metric{name: "CPUUtilization", statistics: "Average", period: 60}
	metrics := []Metric{metric}
	check := Check{name: "jenkins", tags: tags, metrics: metrics}
	checks := []Check{check}
	//config := Config{checks: checks}

	registry := prometheus.NewRegistry()
	prometheus.DefaultGatherer = registry
	ec2Metrics(registry, checks)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Prometheus Cloudwatch Exporter</title></head>
            <body>
            <h1>Prometheus Cloudwatch Exporter</h1>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})
	flag.Parse()
	http.HandleFunc("/metrics", metricsHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
