package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	exporter "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

type scraper struct {
	registry     *prometheus.Registry
	featureFlags []string
}

func NewScraper(featureFlags []string) *scraper { //nolint:revive
	return &scraper{
		registry:     prometheus.NewRegistry(),
		featureFlags: featureFlags,
	}
}

func (s *scraper) makeHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{
			DisableCompression: false,
		})
		handler.ServeHTTP(w, r)
	}
}

func (s *scraper) decoupled(ctx context.Context, logger logging.Logger, cache clients.ClientsCache) {
	logger.Debug("Starting scraping async")
	s.scrape(ctx, logger, cache)

	scrapingDuration := time.Duration(scrapingInterval) * time.Second
	ticker := time.NewTicker(scrapingDuration)
	logger.Debug("Initial scrape completed", "scraping_interval", scrapingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Debug("Starting scraping async")
			go s.scrape(ctx, logger, cache)
		}
	}
}

func (s *scraper) scrape(ctx context.Context, logger logging.Logger, cache clients.ClientsCache) {
	if !sem.TryAcquire(1) {
		// This shouldn't happen under normal use, users should adjust their configuration when this occurs.
		// Let them know by logging a warning.
		logger.Warn("Another scrape is already in process, will not start a new one. " +
			"Adjust your configuration to ensure the previous scrape completes first.")
		return
	}
	defer sem.Release(1)

	newRegistry := prometheus.NewRegistry()
	for _, metric := range exporter.Metrics {
		if err := newRegistry.Register(metric); err != nil {
			logger.Warn("Could not register cloudwatch api metric")
		}
	}

	err := exporter.UpdateMetrics(
		ctx,
		logger,
		cfg,
		newRegistry,
		cache,
		exporter.MetricsPerQuery(metricsPerQuery),
		exporter.LabelsSnakeCase(labelsSnakeCase),
		exporter.EnableFeatureFlag(s.featureFlags...),
		exporter.CloudWatchAPIConcurrency(cloudwatchConcurrency),
		exporter.TaggingAPIConcurrency(tagConcurrency),
	)
	if err != nil {
		logger.Error(err, "error updating metrics")
	}

	// this might have a data race to access registry
	s.registry = newRegistry
	logger.Debug("Metrics scraped")
}
