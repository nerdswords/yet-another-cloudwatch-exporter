package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var version = "custom-build"

var (
	addr                  = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile            = flag.String("config.file", "config.yml", "Path to configuration file.")
	debug                 = flag.Bool("debug", false, "Add verbose logging")
	showVersion           = flag.Bool("v", false, "prints current yace version.")
	cloudwatchConcurrency = flag.Int("cloudwatch-concurrency", 5, "Maximum number of concurrent requests to CloudWatch API")
	tagConcurrency        = flag.Int("tag-concurrency", 5, "Maximum number of concurrent requests to Resource Tagging API")

	supportedServices = []string{
		"alb",
		"dynamodb",
		"ebs",
		"ec",
		"ecs-svc",
		"ec2",
		"efs",
		"elb",
		"emr",
		"es",
		"lambda",
		"nlb",
		"rds",
		"s3",
		"kinesis",
		"vpn",
		"asg",
		"sqs",
	}

	config = conf{}
)

func init() {
	flag.Parse()

	// Set JSON structured logging as the default log formatter
	log.SetFormatter(&log.JSONFormatter{})

	// Set the Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// Only log Info severity or above.
	log.SetLevel(log.InfoLevel)

}

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

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Println("Parse config..")
	if err := config.load(configFile); err != nil {
		log.Fatal("Couldn't read ", *configFile, ":", err)
	}

	cloudwatchSemaphore = make(chan struct{}, *cloudwatchConcurrency)
	tagSemaphore = make(chan struct{}, *tagConcurrency)

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
