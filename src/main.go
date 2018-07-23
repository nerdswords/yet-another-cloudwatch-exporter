package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
)

const AppVersion = "0.4.0"

var (
	addr                  = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile            = flag.String("config.file", "config.yml", "Path to configuration file.")
	supportedServices     = []string{"rds", "ec2", "elb", "es", "ec", "s3"}
	config                = conf{}
	CloudwatchApiRequests = uint64(0)
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	tagsData, cloudwatchData := scrapeAwsData(config.Jobs)

	var promData []*PrometheusData

	promData = append(promData, migrateCloudwatchToPrometheus(cloudwatchData)...)
	promData = append(promData, migrateTagsToPrometheus(tagsData)...)

	promData = removePromDouble(promData)

	registry := fillRegistry(promData)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	version := flag.Bool("v", false, "prints current yace version")

	flag.Parse()

	if *version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	log.Println("Parse config..")
	config.getConf(configFile)
	config.setDefaults()
	config.verifyService()
	log.Println("Config was parsed successfully")

	log.Println("Startup completed")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
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
