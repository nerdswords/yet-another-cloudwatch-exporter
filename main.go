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
					cloudwatchHelper = append(cloudwatchHelper, data)
					mux.Unlock()
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	for _, c := range awsHelper {
		metric := createInfoMetric(c)
		registry.MustRegister(metric)
	}

	for _, c := range cloudwatchHelper {
		metric := createCloudwatchMetric(*c)
		registry.MustRegister(metric)
	}

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
