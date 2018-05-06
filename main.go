package main

import (
	"flag"
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	_ "strings"
	"sync"
)

var (
	addr              = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile        = flag.String("config.file", "config.yml", "Path to configuration file.")
	supportedServices = []string{"rds", "ec2", "elb", "es", "ec"}
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	var c conf
	c.getConf(configFile)

	registry := prometheus.NewRegistry()

	exportedTags := createPrometheusExportedTags(c.Jobs)

	var wg sync.WaitGroup
	wg.Add(len(c.Jobs))

	for i, _ := range c.Jobs {
		job := c.Jobs[i]
		go func() {
			metrics(registry, job, exportedTags[job.Discovery.Type])
			wg.Done()
		}()
	}
	wg.Wait()

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
		<h1>Thanks for using our product :)</h1>
		<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>`))
	})

	flag.Parse()
	http.HandleFunc("/metrics", metricsHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
