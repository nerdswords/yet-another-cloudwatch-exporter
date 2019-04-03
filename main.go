package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "custom-build"

var (
	addr        = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile  = flag.String("config.file", "config.yml", "Path to configuration file.")
	debug       = flag.Bool("debug", false, "Add verbose logging")
	showVersion = flag.Bool("v", false, "prints current yace version.")

	supportedServices = []string{
		"alb",
		"dynamodb",
		"ebs",
		"ec",
		"ec2",
		"efs",
		"elb",
		"es",
		"lambda",
		"rds",
		"s3",
		"kinesis",
		"vpn",
	}

	config = conf{}
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	tagsData, cloudwatchData := scrapeAwsData(config)

	var metrics []*PrometheusMetric

	metrics = append(metrics, migrateCloudwatchToPrometheus(cloudwatchData)...)
	metrics = append(metrics, migrateTagsToPrometheus(tagsData)...)

	registry := prometheus.NewRegistry()
	registry.MustRegister(NewPrometheusCollector(metrics))

	if err := registry.Register(cloudwatchAPICounter); err != nil {
		log.Fatal("Could not publish cloudwatch api metric")
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Println("Parse config..")
	if err := config.load(configFile); err != nil {
		log.Fatal("Couldn't read ", *configFile, ":", err)
	}

	log.Println("Startup completed")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
		<head><title>Yet another cloudwatch exporter</title></head>
		<body>
		<h1>Thanks for using our product :)</h1>
		<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>`))
	})

	http.HandleFunc("/metrics", metricsHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
