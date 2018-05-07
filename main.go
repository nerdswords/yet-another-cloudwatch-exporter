package main

import (
	"flag"
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"sync"
)

var (
	addr              = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile        = flag.String("config.file", "config.yml", "Path to configuration file.")
	supportedServices = []string{"rds", "ec2", "elb", "es", "ec"}
	c                 = conf{}
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	registry := prometheus.NewRegistry()

	mux := &sync.Mutex{}

	cloudwatchHelper := make([]*cloudwatchData, 0)
	awsHelper := make([]*awsResource, 0)

	var wg sync.WaitGroup
	for i, _ := range c.Jobs {
		wg.Add(1)
		job := c.Jobs[i]
		go func() {
			resources := describeResources(job.Discovery)

			for _, resource := range resources.Resources {
				mux.Lock()
				awsHelper = append(awsHelper, resource)
				mux.Unlock()

				for _, metric := range job.Metrics {
					data := getCloudwatchData(resource, metric)
					mux.Lock()
					if data.Value != nil {
						cloudwatchHelper = append(cloudwatchHelper, data)
					}
					mux.Unlock()
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	exportedTags := findExportedTags(awsHelper)

	createPrometheusMetrics(registry, awsHelper, cloudwatchHelper, exportedTags)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	c.getConf(configFile)

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
