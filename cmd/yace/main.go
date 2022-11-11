package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

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
		{
			Name: "verify-config", Aliases: []string{"vc"}, Usage: "Loads and attempts to parse config file, then exits. Useful for CI/CD validation",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "config.file", Value: "config.yml", Usage: "Path to configuration file.", Destination: &configFile},
			},
			Action: func(c *cli.Context) error {
				log.Println("Parse config..")
				if err := config.Load(&configFile); err != nil {
					log.Fatal("Couldn't read ", configFile, ": ", err)
					os.Exit(1)
				}
				log.Info("Config ", configFile, " is valid")
				os.Exit(0)
				return nil
			},
		},
		{
			Name: "version", Aliases: []string{"v"}, Usage: "prints current yace version.",
			Action: func(c *cli.Context) error {
				fmt.Println(version)
				os.Exit(0)
				return nil
			},
		},
	}

	yace.Action = startScraper

	if err := yace.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startScraper(_ *cli.Context) error {
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Println("Parse config..")
	if err := config.Load(&configFile); err != nil {
		return fmt.Errorf("Couldn't read %s: %w", configFile, err)
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
		ctx, cancelRunningScrape = context.WithCancel(context.Background())
		go s.decoupled(ctx, cache)
	})

	return http.ListenAndServe(addr, nil)
}
