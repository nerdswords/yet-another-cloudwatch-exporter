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
	labelsSnakeCase       = flag.Bool("labels-snake-case", false, "If labels should be output in snake case instead of camel case")

	supportedServices = []string{
		"alb",
		"apigateway",
		"appsync",
		"asg",
		"cf",
		"dynamodb",
		"ebs",
		"ec",
		"ec2",
		"ec2Spot",
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
		"redshift",
		"r53r",
		"s3",
		"sfn",
		"sns",
		"sqs",
		"tgw",
		"tgwa",
		"vpn",
		"wafv2",
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

func updateMetrics(registry *prometheus.Registry, now time.Time) time.Time {
	tagsData, cloudwatchData, nendtime := scrapeAwsData(config, now)
	var metrics []*PrometheusMetric

	metrics = append(metrics, migrateCloudwatchToPrometheus(cloudwatchData)...)
	metrics = append(metrics, migrateTagsToPrometheus(tagsData)...)

	metrics = ensureLabelConsistencyForMetrics(metrics)

	registry.MustRegister(NewPrometheusCollector(metrics))
	for _, counter := range []prometheus.Counter{cloudwatchAPICounter, cloudwatchGetMetricDataAPICounter, cloudwatchGetMetricStatisticsAPICounter, resourceGroupTaggingAPICounter, autoScalingAPICounter, apiGatewayAPICounter} {
		if err := registry.Register(counter); err != nil {
			log.Warning("Could not publish cloudwatch api metric")
		}
	}
	return *nendtime
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
	//swtich this to perdiod right now testing it for 5 minutes granuality and roundtime time to 5 minutes for exaple 12:00,12:05 etc etc
	//Variables to hold last scrape time
	var now time.Time
	//variable to hold total processing time.
	var processingtimeTotal time.Duration
	maxjoblength := 0
	for _, discoveryJob := range config.Discovery.Jobs {
		length := getMetricDataInputLength(discoveryJob)
		//S3 can have upto 1 day to day will need to address it in seprate block
		//TBD
		if (maxjoblength < length) && (discoveryJob.Type != "s3") {
			maxjoblength = length
		}
	}

	//To aviod future timestamp issue we need make sure scrape intervel is atleast at the same level as that of highest job length
	if *scrapingInterval < maxjoblength {
		*scrapingInterval = maxjoblength
	}

	if *decoupledScraping {

		go func() {
			for {
				t0 := time.Now()
				newRegistry := prometheus.NewRegistry()
				nendtime := updateMetrics(newRegistry, now)
				now = nendtime
				log.Debug("Metrics scraped.")
				registry = newRegistry
				t1 := time.Now()
				processingtime := t1.Sub(t0)
				processingtimeTotal = processingtimeTotal + processingtime
				if processingtimeTotal.Seconds() > 60.0 {
					sleepinterval := *scrapingInterval - int(processingtimeTotal.Seconds())
					//reset processingtimeTotal
					processingtimeTotal = 0
					if sleepinterval <= 0 {
						//TBD use cases is when metrics like EC2 and EBS take more scrapping interval like 6 to 7 minutes to finish
						continue
					} else {
						time.Sleep(time.Duration(sleepinterval) * time.Second)
					}

				} else {
					time.Sleep(time.Duration(*scrapingInterval) * time.Second)
				}

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
			updateMetrics(newRegistry, now)
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
