package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/session"
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

	logger logging.Logger

	cfg = config.ScrapeConf{}
)

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

	yace.Before = func(ctx *cli.Context) error {
		logger = newLogger(debug)
		return nil
	}

	yace.Commands = []*cli.Command{
		{
			Name: "verify-config", Aliases: []string{"vc"}, Usage: "Loads and attempts to parse config file, then exits. Useful for CI/CD validation",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "config.file", Value: "config.yml", Usage: "Path to configuration file.", Destination: &configFile},
			},
			Action: func(c *cli.Context) error {
				logger.Info("Parsing config")
				if err := cfg.Load(configFile, logger); err != nil {
					logger.Error(err, "Couldn't read config file", "path", configFile)
					os.Exit(1)
				}
				logger.Info("Config file is valid", "path", configFile)
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
		logger.Error(err, "Error running yace")
		os.Exit(1)
	}
}

func startScraper(_ *cli.Context) error {
	logger.Info("Parsing config")
	if err := cfg.Load(configFile, logger); err != nil {
		return fmt.Errorf("Couldn't read %s: %w", configFile, err)
	}

	logger.Info("Startup completed")

	s := NewScraper()
	cache := session.NewSessionCache(cfg, fips, logger)

	ctx, cancelRunningScrape := context.WithCancel(context.Background())
	go s.decoupled(ctx, logger, cache)

	http.HandleFunc("/metrics", s.makeHandler(ctx, cache))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf(`<html>
    <head><title>Yet another cloudwatch exporter</title></head>
    <body>
    <h1>Thanks for using Yace :)</h1>
		Version: %s
    <p><a href="/metrics">Metrics</a></p>
    </body>
    </html>`, version)))
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
		logger.Info("Parsing config")
		if err := cfg.Load(configFile, logger); err != nil {
			logger.Error(err, "Couldn't read config file", "path", configFile)
			return
		}

		logger.Info("Reset session cache")
		cache = session.NewSessionCache(cfg, fips, logger)

		cancelRunningScrape()
		ctx, cancelRunningScrape = context.WithCancel(context.Background())
		go s.decoupled(ctx, logger, cache)
	})

	return http.ListenAndServe(addr, nil)
}

func newLogger(debug bool) logging.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetOutput(os.Stdout)

	if debug {
		l.SetLevel(logrus.DebugLevel)
	} else {
		l.SetLevel(logrus.InfoLevel)
	}

	return logging.NewLogger(l)
}
