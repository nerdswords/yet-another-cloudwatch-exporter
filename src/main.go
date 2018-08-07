package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
)

const yaceVersion = "0.4.0"

var (
	addr              = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile        = flag.String("config.file", "config.yml", "Path to configuration file.")
	version           = flag.Bool("v", false, "prints current yace version")
	supportedServices = []string{"rds", "ec2", "elb", "es", "ec", "s3", "efs"}
	config            = conf{}
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	tagsData, cloudwatchData := scrapeAwsData(config.Jobs)

	var promData []*prometheusData

	promData = append(promData, migrateCloudwatchToPrometheus(cloudwatchData)...)
	promData = append(promData, migrateTagsToPrometheus(tagsData)...)

	promData = removePromDouble(promData)

	registry := fillRegistry(promData)

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

	if *version {
		fmt.Println(yaceVersion)
		os.Exit(0)
	}

	log.Println("Parse config..")
	if err := config.load(configFile); err != nil {
		log.Fatal("Couldn't read config", *configFile, ":", err)
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
