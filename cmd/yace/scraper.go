package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	exporter "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/session"
)

type scraper struct {
	cloudwatchSemaphore chan struct{}
	tagSemaphore        chan struct{}
	registry            *prometheus.Registry
}

func NewScraper() *scraper { //nolint:revive
	return &scraper{
		cloudwatchSemaphore: make(chan struct{}, cloudwatchConcurrency),
		tagSemaphore:        make(chan struct{}, tagConcurrency),
		registry:            prometheus.NewRegistry(),
	}
}

func (s *scraper) makeHandler(ctx context.Context, cache session.SessionCache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{
			DisableCompression: false,
		})
		handler.ServeHTTP(w, r)
	}
}

func (s *scraper) decoupled(ctx context.Context, logger logging.Logger, cache session.SessionCache) {
	logger.Debug("Starting scraping async")
	logger.Debug("Scrape initially first time")
	s.scrape(ctx, logger, cache)

	scrapingDuration := time.Duration(scrapingInterval) * time.Second
	ticker := time.NewTicker(scrapingDuration)
	logger.Debug(fmt.Sprintf("Scraping every %d seconds", scrapingInterval))
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

var observedMetricLabels = map[string]model.LabelSet{}

func (s *scraper) scrape(ctx context.Context, logger logging.Logger, cache session.SessionCache) {
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
	exporter.UpdateMetrics(ctx, cfg, newRegistry, metricsPerQuery, labelsSnakeCase, s.cloudwatchSemaphore, s.tagSemaphore, cache, observedMetricLabels, logger)

	// this might have a data race to access registry
	s.registry = newRegistry
	logger.Debug("Metrics scraped.")
}
