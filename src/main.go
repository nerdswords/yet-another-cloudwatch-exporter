package main

import (
	"flag"
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

var (
	addr                  = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile            = flag.String("config.file", "config.yml", "Path to configuration file.")
	supportedServices     = []string{"rds", "ec2", "elb", "es", "ec", "s3"}
	config                = conf{}
	CloudwatchApiRequests = uint64(0)
)

func metricsHandler(w http.ResponseWriter, req *http.Request) {
	awsInfoData, cloudwatchData := scrapeAwsData(config.Jobs)

	registry := createPrometheusMetrics(awsInfoData, cloudwatchData)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		DisableCompression: false,
	})

	handler.ServeHTTP(w, req)
}

func main() {
	flag.Parse()

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
