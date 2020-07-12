package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var version = "custom-build"

var (
	addr                  = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile            = flag.String("config.file", "config.yml", "Path to configuration file.")
	debug                 = flag.Bool("debug", false, "Add verbose logging.")
	showVersion           = flag.Bool("v", false, "prints current yace version.")
	cloudwatchConcurrency = flag.Int("cloudwatch-concurrency", 5, "Maximum number of concurrent requests to CloudWatch API.")
	tagConcurrency        = flag.Int("tag-concurrency", 5, "Maximum number of concurrent requests to Resource Tagging API.")
	scrapingInterval      = flag.Int("scraping-interval", 300, "Seconds to wait between scraping the AWS metrics if decoupled scraping.")
	decoupledScraping     = flag.Bool("decoupled-scraping", true, "Decouples scraping and serving of metrics.")
	metricsPerQuery       = flag.Int("metrics-per-query", 500, "Number of metrics made in a single GetMetricsData request")

	supportedServices = []string{
		"alb",
		"appsync",
		"asg",
		"cf",
		"dynamodb",
		"ebs",
		"ec",
		"ec2",
		"ecs-svc",
		"ecs-containerinsights",
		"efs",
		"elb",
		"emr",
		"es",
		"firehose",
		"fsx",
		"kafka",
		"kinesis",
		"lambda",
		"ngw",
		"nlb",
		"rds",
		"r53r",
		"s3",
		"sfn",
		"sns",
		"sqs",
		"tgw",
		"vpn",
		"acm-certificates",
		"yle-ec2",
		"yle-ecs",
	}

	config = conf{}
)

func init() {

	// Set JSON structured logging as the default log formatter
	log.SetFormatter(&log.JSONFormatter{})

	// Set the Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// Only log Info severity or above.
	log.SetLevel(log.InfoLevel)

}

func updateMetrics(registry *prometheus.Registry) {
	tagsData, cloudwatchData := scrapeAwsData(config)

	var metrics []*PrometheusMetric

	metrics = append(metrics, migrateCloudwatchToPrometheus(cloudwatchData)...)
	metrics = append(metrics, migrateTagsToPrometheus(tagsData)...)

	log.Debugf("updateMetrics with %d metrics", len(metrics))
	registry.MustRegister(NewPrometheusCollector(metrics))
	for _, counter := range []prometheus.Counter{cloudwatchAPICounter, cloudwatchGetMetricDataAPICounter, cloudwatchGetMetricStatisticsAPICounter, resourceGroupTaggingAPICounter, autoScalingAPICounter} {
		if err := registry.Register(counter); err != nil {
			log.Warning("Could not publish cloudwatch api metric")
		}
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Println("Parse config..")
	if err := config.load(configFile); err != nil {
		log.Fatal("Couldn't read ", *configFile, ": ", err)
	}

	cloudwatchSemaphore = make(chan struct{}, *cloudwatchConcurrency)
	tagSemaphore = make(chan struct{}, *tagConcurrency)

	registry := prometheus.NewRegistry()

	log.Println("Startup completed")

	if *decoupledScraping {
		go func() {
			for {
				newRegistry := prometheus.NewRegistry()
				updateMetrics(newRegistry)
				log.Debug("Metrics scraped.")
				registry = newRegistry
				time.Sleep(time.Duration(*scrapingInterval) * time.Second)
			}
		}()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
		<head><title>Yet another cloudwatch exporter</title></head>
		<body>
		<h1>Thanks for using our product :)</h1>
		<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>`))
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !(*decoupledScraping) {
			newRegistry := prometheus.NewRegistry()
			updateMetrics(newRegistry)
			log.Debug("Metrics scraped.")
			registry = newRegistry
		}
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			DisableCompression: false,
		})
		handler.ServeHTTP(w, r)
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}
