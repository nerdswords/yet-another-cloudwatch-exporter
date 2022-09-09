package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"

	exporter "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg"
)

var version = "custom-build"

var sem = semaphore.NewWeighted(1)

var (
	addr                  string
	configFile            string
	debug                 bool
	fips                  bool
	cloudwatchConcurrency int
	tagConcurrency        int
	scrapingInterval      int
	metricsPerQuery       int
	labelsSnakeCase       bool

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
	yace := cli.NewApp()
	yace.Name = "Yet Another CloudWatch Exporter"
	yace.Version = version
	yace.Usage = "YACE configured to retrieve CloudWatch metrics through the AWS API"
	yace.Description = ""
	yace.Authors = []*cli.Author{
		{Name: "", Email: ""},
	}

	yace.Flags = []cli.Flag{
		&cli.StringFlag{Name: "listen-address", Value: ":5000", Usage: "The address to listen on.", Destination: &addr, EnvVars: []string{"listen-address"}},
		&cli.StringFlag{Name: "config.file", Value: "config.yml", Usage: "Path to configuration file.", Destination: &configFile, EnvVars: []string{"config.file"}},
		&cli.BoolFlag{Name: "debug", Value: false, Usage: "Add verbose logging.", Destination: &debug, EnvVars: []string{"debug"}},
		&cli.BoolFlag{Name: "fips", Value: false, Usage: "Use FIPS compliant aws api.", Destination: &fips},
		&cli.IntFlag{Name: "cloudwatch-concurrency", Value: 5, Usage: "Maximum number of concurrent requests to CloudWatch API.", Destination: &cloudwatchConcurrency},
		&cli.IntFlag{Name: "tag-concurrency", Value: 5, Usage: "Maximum number of concurrent requests to Resource Tagging API.", Destination: &tagConcurrency},
		&cli.IntFlag{Name: "scraping-interval", Value: 300, Usage: "Seconds to wait between scraping the AWS metrics", Destination: &scrapingInterval, EnvVars: []string{"scraping-interval"}},
		&cli.IntFlag{Name: "metrics-per-query", Value: 500, Usage: "Number of metrics made in a single GetMetricsData request", Destination: &metricsPerQuery, EnvVars: []string{"metrics-per-query"}},
		&cli.BoolFlag{Name: "labels-snake-case", Value: false, Usage: "If labels should be output in snake case instead of camel case", Destination: &labelsSnakeCase},
	}

	yace.Commands = []*cli.Command{
		{Name: "verify-config", Aliases: []string{"vc"}, Usage: "Loads and attempts to parse config file, then exits. Useful for CICD validation",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "config.file", Value: "config.yml", Usage: "Path to configuration file.", Destination: &configFile},
			},
			Action: func(c *cli.Context) error {
				log.Info("Config ", configFile, " is valid")
				os.Exit(0)
				return nil
			}},
		{Name: "version", Aliases: []string{"v"}, Usage: "prints current yace version.",
			Action: func(c *cli.Context) error {
				fmt.Println(version)
				os.Exit(0)
				return nil
			}},
	}

	err := yace.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Println("Parse config..")
	if err := config.Load(&configFile); err != nil {
		log.Fatal("Couldn't read ", configFile, ": ", err)
		os.Exit(1)
	}

	log.Println("Startup completed")

	s := NewScraper()
	cache := exporter.NewSessionCache(config, fips, exporter.NewLogrusLogger(log.StandardLogger()))

	ctx, cancelRunningScrape := context.WithCancel(context.Background())
	go s.decoupled(ctx, cache)

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

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println("Parse config..")
		if err := config.Load(&configFile); err != nil {
			log.Fatal("Couldn't read ", &configFile, ": ", err)
		}

		log.Println("Reset session cache")
		cache = exporter.NewSessionCache(config, fips, exporter.NewLogrusLogger(log.StandardLogger()))

		cancelRunningScrape()
		// TODO: Pipe ctx through to the AWS calls.
		ctx, cancelRunningScrape = context.WithCancel(context.Background())
		go s.decoupled(ctx, cache)
	})

	log.Fatal(http.ListenAndServe(addr, nil))
}

type scraper struct {
	cloudwatchSemaphore chan struct{}
	tagSemaphore        chan struct{}
	registry            *prometheus.Registry
}

func NewScraper() *scraper {
	return &scraper{
		cloudwatchSemaphore: make(chan struct{}, cloudwatchConcurrency),
		tagSemaphore:        make(chan struct{}, tagConcurrency),
		registry:            prometheus.NewRegistry(),
	}
}

func (s *scraper) makeHandler(ctx context.Context, cache exporter.SessionCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{
			DisableCompression: false,
		})
		handler.ServeHTTP(w, r)
	}
}

func (s *scraper) decoupled(ctx context.Context, cache exporter.SessionCache) {
	log.Debug("Starting scraping async")
	log.Debug("Scrape initially first time")
	s.scrape(ctx, cache)

	scrapingDuration := time.Duration(scrapingInterval) * time.Second
	ticker := time.NewTicker(scrapingDuration)
	log.Debugf("Scraping every %d seconds", scrapingInterval)
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

var observedMetricLabels = map[string]exporter.LabelSet{}

func (s *scraper) scrape(ctx context.Context, cache exporter.SessionCache) {
	if !sem.TryAcquire(1) {
		// This shouldn't happen under normal use, users should adjust their configuration when this occurs.
		// Let them know by logging a warning.
		log.Warn("Another scrape is already in process, will not start a new one. " +
			"Adjust your configuration to ensure the previous scrape completes first.")
		return
	}
	defer sem.Release(1)

	newRegistry := prometheus.NewRegistry()
	for _, metric := range exporter.Metrics {
		if err := newRegistry.Register(metric); err != nil {
			log.Warning("Could not register cloudwatch api metric")
		}
	}
	exporter.UpdateMetrics(ctx, config, newRegistry, metricsPerQuery, labelsSnakeCase, s.cloudwatchSemaphore, s.tagSemaphore, cache, observedMetricLabels, exporter.NewLogrusLogger(log.StandardLogger()))

	// this might have a data race to access registry
	s.registry = newRegistry
	log.Debug("Metrics scraped.")
}
