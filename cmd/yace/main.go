package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	v2 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/v2"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"

	exporter "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg"
	v1 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/v1"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

const (
	enableFeatureFlag = "enable-feature"
	htmlVersion       = `<html>
<head><title>Yet Another CloudWatch Exporter</title></head>
<body>
<h1>Thanks for using YACE :)</h1>
Version: %s
<p><a href="/metrics">Metrics</a></p>
%s
</body>
</html>`
	htmlPprof = `<p><a href="/debug/pprof">Pprof</a><p>`
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
	profilingEnabled      bool
	awsSdkV2Enabled       bool

	logger logging.Logger

	cfg = config.ScrapeConf{}
)

func main() {
	app := NewYACEApp()
	if err := app.Run(os.Args); err != nil {
		logger.Error(err, "Error running yace")
		os.Exit(1)
	}
}

// NewYACEApp creates a new cli.App implementing the YACE entrypoints and CLI arguments.
func NewYACEApp() *cli.App {
	yace := cli.NewApp()
	yace.Name = "Yet Another CloudWatch Exporter"
	yace.Version = version
	yace.Usage = "YACE configured to retrieve CloudWatch metrics through the AWS API"
	yace.Description = ""
	yace.Authors = []*cli.Author{
		{Name: "", Email: ""},
	}

	yace.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "listen-address",
			Value:       ":5000",
			Usage:       "The address to listen on",
			Destination: &addr,
			EnvVars:     []string{"listen-address"},
		},
		&cli.StringFlag{
			Name:        "config.file",
			Value:       "config.yml",
			Usage:       "Path to configuration file",
			Destination: &configFile,
			EnvVars:     []string{"config.file"},
		},
		&cli.BoolFlag{
			Name:        "debug",
			Value:       false,
			Usage:       "Verbose logging",
			Destination: &debug,
			EnvVars:     []string{"debug"},
		},
		&cli.BoolFlag{
			Name:        "fips",
			Value:       false,
			Usage:       "Use FIPS compliant AWS API endpoints",
			Destination: &fips,
		},
		&cli.IntFlag{
			Name:        "cloudwatch-concurrency",
			Value:       exporter.DefaultCloudWatchAPIConcurrency,
			Usage:       "Maximum number of concurrent requests to CloudWatch API.",
			Destination: &cloudwatchConcurrency,
		},
		&cli.IntFlag{
			Name:        "tag-concurrency",
			Value:       exporter.DefaultTaggingAPIConcurrency,
			Usage:       "Maximum number of concurrent requests to Resource Tagging API.",
			Destination: &tagConcurrency,
		},
		&cli.IntFlag{
			Name:        "scraping-interval",
			Value:       300,
			Usage:       "Seconds to wait between scraping the AWS metrics",
			Destination: &scrapingInterval,
			EnvVars:     []string{"scraping-interval"},
		},
		&cli.IntFlag{
			Name:        "metrics-per-query",
			Value:       exporter.DefaultMetricsPerQuery,
			Usage:       "Number of metrics made in a single GetMetricsData request",
			Destination: &metricsPerQuery,
			EnvVars:     []string{"metrics-per-query"},
		},
		&cli.BoolFlag{
			Name:        "labels-snake-case",
			Value:       exporter.DefaultLabelsSnakeCase,
			Usage:       "Whether labels should be output in snake case instead of camel case",
			Destination: &labelsSnakeCase,
		},
		&cli.BoolFlag{
			Name:        "profiling.enabled",
			Value:       false,
			Usage:       "Enable pprof endpoints",
			Destination: &profilingEnabled,
		},
		&cli.StringSliceFlag{
			Name:  enableFeatureFlag,
			Usage: "Comma-separated list of enabled features",
		},
		&cli.BoolFlag{
			Name:        "aws.sdk.v2.enabled",
			Value:       false,
			Usage:       "Enable the use of aws sdk v2",
			Destination: &awsSdkV2Enabled,
		},
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

	return yace
}

func startScraper(c *cli.Context) error {
	logger.Info("Parsing config")
	if err := cfg.Load(configFile, logger); err != nil {
		return fmt.Errorf("Couldn't read %s: %w", configFile, err)
	}

	logger.Info("Yace startup completed", "version", version)

	featureFlags := c.StringSlice(enableFeatureFlag)

	s := NewScraper(featureFlags)
	cache := v1.NewClientCache(cfg, fips, logger)
	if awsSdkV2Enabled {
		var err error
		// Can't override cache while also creating err
		cache, err = v2.NewCache(cfg, fips, logger)
		if err != nil {
			return fmt.Errorf("failed to construct aws sdk v2 client cache: %w", err)
		}
	}

	ctx, cancelRunningScrape := context.WithCancel(context.Background())
	go s.decoupled(ctx, logger, cache)

	mux := http.NewServeMux()

	if profilingEnabled {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	mux.HandleFunc("/metrics", s.makeHandler())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pprofLink := ""
		if profilingEnabled {
			pprofLink = htmlPprof
		}

		_, _ = w.Write([]byte(fmt.Sprintf(htmlVersion, version, pprofLink)))
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Info("Parsing config")
		if err := cfg.Load(configFile, logger); err != nil {
			logger.Error(err, "Couldn't read config file", "path", configFile)
			return
		}

		logger.Info("Reset clients cache")
		cache = v1.NewClientCache(cfg, fips, logger)

		cancelRunningScrape()
		ctx, cancelRunningScrape = context.WithCancel(context.Background())
		go s.decoupled(ctx, logger, cache)
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.ListenAndServe()
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
