package main

import (
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
		metrics(registry, job)
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	fmt.Println("Started..")
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
