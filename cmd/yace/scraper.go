package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	exporter "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg"
)

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
