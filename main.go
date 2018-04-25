package main

import (
	_ "encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	_ "strings"
)

var (
	addr       = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile = flag.String("config.file", "config.yml", "Path to configuration file.")
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	var c conf
	c.getConf(configFile)

	registry := prometheus.NewRegistry()

	for _, job := range c.Jobs {
		if job.Type == "EC2" {
			ec2Metrics(registry, job)
		}
		if job.Type == "ELB" {
			elbMetrics(registry, job)
		}
	}

	//prometheus.DefaultGatherer = registry //- Do i need this? :D

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>Yet another cloudwatch exporter</title></head>
            <body>
            <h1>Exporter with benefits</h1>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	flag.Parse()
	http.HandleFunc("/metrics", metricsHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
