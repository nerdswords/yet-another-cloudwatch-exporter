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

	"github.com/ivx/yet-another-cloudwatch-exporter/pkg"
)

var version = "custom-build"

var (
	addr                  = flag.String("listen-address", ":5000", "The address to listen on.")
	configFile            = flag.String("config.file", "config.yml", "Path to configuration file.")
	debug                 = flag.Bool("debug", false, "Add verbose logging.")
	fips                  = flag.Bool("fips", false, "Use FIPS compliant aws api.")
	showVersion           = flag.Bool("v", false, "prints current yace version.")
	cloudwatchConcurrency = flag.Int("cloudwatch-concurrency", 5, "Maximum number of concurrent requests to CloudWatch API.")
	tagConcurrency        = flag.Int("tag-concurrency", 5, "Maximum number of concurrent requests to Resource Tagging API.")
	scrapingInterval      = flag.Int("scraping-interval", 300, "Seconds to wait between scraping the AWS metrics if decoupled scraping.")
	decoupledScraping     = flag.Bool("decoupled-scraping", true, "Decouples scraping and serving of metrics.")
	metricsPerQuery       = flag.Int("metrics-per-query", 500, "Number of metrics made in a single GetMetricsData request")
	labelsSnakeCase       = flag.Bool("labels-snake-case", false, "If labels should be output in snake case instead of camel case")
	floatingTimeWindow    = flag.Bool("floating-time-window", false, "Use a floating start/end time window instead of rounding times to 5 min intervals")
	verifyConfig          = flag.Bool("verify-config", false, "Loads and attempts to parse config file, then exits. Useful for CICD validation")

	config = exporter.ScrapeConf{}
)

func init() {

	// Set JSON structured logging as the default log formatter
	log.SetFormatter(&log.JSONFormatter{})

	// Set the Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// Only log Info severity or above.
	log.SetLevel(log.InfoLevel)

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
	if err := config.Load(configFile); err != nil {
		log.Fatal("Couldn't read ", *configFile, ": ", err)
		os.Exit(1)
	}
	if *verifyConfig {
		log.Info("Config ", *configFile, " is valid")
		os.Exit(0)
	}

	cloudwatchSemaphore := make(chan struct{}, *cloudwatchConcurrency)
	tagSemaphore := make(chan struct{}, *tagConcurrency)

	registry := prometheus.NewRegistry()

	log.Println("Startup completed")
	//Variables to hold last scrape time
	var now time.Time
	//variable to hold total processing time.
	var processingtimeTotal time.Duration
	maxjoblength := 0
	for _, discoveryJob := range config.Discovery.Jobs {
		length := exporter.GetMetricDataInputLength(discoveryJob)
		//S3 can have upto 1 day to day will need to address it in seprate block
		//TBD
		svc := exporter.SupportedServices.GetService(discoveryJob.Type)
		if (maxjoblength < length) && !svc.IgnoreLength {
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
				endtime := exporter.UpdateMetrics(config, newRegistry, now, *metricsPerQuery, *fips, *debug, *floatingTimeWindow, *labelsSnakeCase, cloudwatchSemaphore, tagSemaphore)
				now = endtime
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
						log.Debug("Unable to sleep since we lagging behind please try adjusting your scrape interval or running this instance with less number of metrics")
						continue
					} else {
						log.Debug("Sleeping smaller intervals to catchup with lag", sleepinterval)
						time.Sleep(time.Duration(sleepinterval) * time.Second)
					}

				} else {
					log.Debug("Sleeping at regular sleep interval ", *scrapingInterval)
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
			exporter.UpdateMetrics(config, newRegistry, now, *metricsPerQuery, *fips, *debug, *floatingTimeWindow, *labelsSnakeCase, cloudwatchSemaphore, tagSemaphore)
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
