package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	exporter "github.com/ivx/yet-another-cloudwatch-exporter/pkg"
)

var version = "custom-build"

var sem = semaphore.NewWeighted(1)

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

	log.Println("Startup completed")

	var maxJobLength int
	for _, discoveryJob := range config.Discovery.Jobs {
		length := exporter.GetMetricDataInputLength(discoveryJob)
		//S3 can have upto 1 day to day will need to address it in seperate block
		//TBD
		svc := exporter.SupportedServices.GetService(discoveryJob.Type)
		if (maxJobLength < length) && !svc.IgnoreLength {
			maxJobLength = length
		}
	}

	// To avoid future timestamp issue we need make sure scrape interval is at least at the same level as that of highest job length
	if *scrapingInterval < maxJobLength {
		*scrapingInterval = maxJobLength
	}

	s := NewScraper()
	cache := exporter.NewSessionCache(config, *fips)

	ctx := context.Background() // ideally this should be carried to the aws calls

	if *decoupledScraping {
		go s.decoupled(ctx, cache)
	}

	http.HandleFunc("/metrics", s.makeHandler(ctx, cache))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
		<head><title>Yet another cloudwatch exporter</title></head>
		<body>
		<h1>Thanks for using our product :)</h1>
		<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>`))
	})
	http.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println("Parse config..")
		if err := config.Load(configFile); err != nil {
			log.Fatal("Couldn't read ", *configFile, ": ", err)
		}
		if *decoupledScraping {
			s.decoupled(ctx)
		} else {
			s.scrape(ctx)
		}
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}

type scraper struct {
	cloudwatchSemaphore chan struct{}
	tagSemaphore        chan struct{}
	now                 time.Time
	registry            *prometheus.Registry
}

func NewScraper() *scraper {
	return &scraper{
		cloudwatchSemaphore: make(chan struct{}, *cloudwatchConcurrency),
		tagSemaphore:        make(chan struct{}, *tagConcurrency),
		registry:            prometheus.NewRegistry(),
	}
}

func (s *scraper) makeHandler(ctx context.Context, cache exporter.SessionCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(*decoupledScraping) {
			s.scrape(ctx, cache)
		}
		handler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{
			DisableCompression: false,
		})
		handler.ServeHTTP(w, r)
	}
}

func (s *scraper) decoupled(ctx context.Context, cache exporter.SessionCache) {
	log.Debug("Starting scraping async")
	go s.scrape(ctx)

	ticker := time.NewTicker(time.Duration(*scrapingInterval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
      log.Debug("Starting scraping async")
			go s.scrape(ctx, cache)
		}
	}
}

func (s *scraper) scrape(ctx context.Context, cache exporter.SessionCache) (err error) {
	if !sem.TryAcquire(1) {
		log.Debug("Another scrape is already in process, will not start a new one")
		return errors.New("scaper already in process")
	}
	defer sem.Release(1)

	newRegistry := prometheus.NewRegistry()
	endtime := exporter.UpdateMetrics(config, newRegistry, s.now, *metricsPerQuery, *fips, *floatingTimeWindow, *labelsSnakeCase, s.cloudwatchSemaphore, s.tagSemaphore, cache)

	// this might have a data race to access registry
	s.registry = newRegistry
	log.Debug("Metrics scraped.")
	return nil
}