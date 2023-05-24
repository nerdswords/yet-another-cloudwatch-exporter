package exporter

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/job"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

// Metrics is a slice of prometheus metrics specific to the scraping process such API call counters
var Metrics = []prometheus.Collector{
	promutil.CloudwatchAPICounter,
	promutil.CloudwatchAPIErrorCounter,
	promutil.CloudwatchGetMetricDataAPICounter,
	promutil.CloudwatchGetMetricStatisticsAPICounter,
	promutil.ResourceGroupTaggingAPICounter,
	promutil.AutoScalingAPICounter,
	promutil.TargetGroupsAPICounter,
	promutil.APIGatewayAPICounter,
	promutil.Ec2APICounter,
	promutil.DmsAPICounter,
	promutil.StoragegatewayAPICounter,
	promutil.DuplicateMetricsFilteredCounter,
}

const (
	DefaultMetricsPerQuery          = 500
	DefaultLabelsSnakeCase          = false
	DefaultCloudWatchAPIConcurrency = 5
	DefaultTaggingAPIConcurrency    = 5
)

// featureFlagsMap is a map that contains the enabled feature flags. If a key is not present, it means the feature flag
// is disabled.
type featureFlagsMap map[string]struct{}

type options struct {
	metricsPerQuery          int
	labelsSnakeCase          bool
	cloudWatchAPIConcurrency int
	taggingAPIConcurrency    int
	featureFlags             featureFlagsMap
}

// IsFeatureEnabled implements the FeatureFlags interface, allowing us to inject the options-configure feature flags in the rest of the code.
func (ff featureFlagsMap) IsFeatureEnabled(flag string) bool {
	_, ok := ff[flag]
	return ok
}

type OptionsFunc func(*options) error

func MetricsPerQuery(metricsPerQuery int) OptionsFunc {
	return func(o *options) error {
		if metricsPerQuery <= 0 {
			return fmt.Errorf("MetricsPerQuery must be a positive value")
		}

		o.metricsPerQuery = metricsPerQuery
		return nil
	}
}

func LabelsSnakeCase(labelsSnakeCase bool) OptionsFunc {
	return func(o *options) error {
		o.labelsSnakeCase = labelsSnakeCase
		return nil
	}
}

func CloudWatchAPIConcurrency(maxConcurrency int) OptionsFunc {
	return func(o *options) error {
		if maxConcurrency <= 0 {
			return fmt.Errorf("CloudWatchAPIConcurrency must be a positive value")
		}

		o.cloudWatchAPIConcurrency = maxConcurrency
		return nil
	}
}

func TaggingAPIConcurrency(maxConcurrency int) OptionsFunc {
	return func(o *options) error {
		if maxConcurrency <= 0 {
			return fmt.Errorf("TaggingAPIConcurrency must be a positive value")
		}

		o.taggingAPIConcurrency = maxConcurrency
		return nil
	}
}

// EnableFeatureFlag is an option that enables a feature flag on the YACE's entrypoint.
func EnableFeatureFlag(flags ...string) OptionsFunc {
	return func(o *options) error {
		for _, flag := range flags {
			o.featureFlags[flag] = struct{}{}
		}
		return nil
	}
}

func defaultOptions() options {
	return options{
		metricsPerQuery:          DefaultMetricsPerQuery,
		labelsSnakeCase:          DefaultLabelsSnakeCase,
		cloudWatchAPIConcurrency: DefaultCloudWatchAPIConcurrency,
		taggingAPIConcurrency:    DefaultTaggingAPIConcurrency,
		featureFlags:             make(featureFlagsMap),
	}
}

// UpdateMetrics is the entrypoint to scrape metrics from AWS on demand.
//
// Parameters are:
// - `ctx`: a context for the request
// - `config`: this is the struct representation of the configuration defined in top-level configuration
// - `logger`: any implementation of the `Logger` interface
// - `registry`: any prometheus compatible registry where scraped AWS metrics will be written
// - `cache`: any implementation of the `Cache`
// - `optFuncs`: (optional) any number of options funcs
//
// You can pre-register any of the default metrics with the provided `registry` if you want them
// included in the AWS scrape results. If you are using multiple instances of `registry` it
// might make more sense to register these metrics in the application using YACE as a library to better
// track them over the lifetime of the application.
func UpdateMetrics(
	ctx context.Context,
	logger logging.Logger,
	cfg config.ScrapeConf,
	registry *prometheus.Registry,
	cache clients.Cache,
	optFuncs ...OptionsFunc,
) error {
	options := defaultOptions()
	for _, f := range optFuncs {
		if err := f(&options); err != nil {
			return err
		}
	}

	// add feature flags to context passed down to all other layers
	ctx = config.CtxWithFlags(ctx, options.featureFlags)

	tagsData, cloudwatchData := job.ScrapeAwsData(
		ctx,
		logger,
		cfg,
		cache,
		options.metricsPerQuery,
		options.cloudWatchAPIConcurrency,
		options.taggingAPIConcurrency,
	)

	metrics, observedMetricLabels, err := promutil.BuildMetrics(cloudwatchData, options.labelsSnakeCase, logger)
	if err != nil {
		logger.Error(err, "Error migrating cloudwatch metrics to prometheus metrics")
		return nil
	}
	metrics, observedMetricLabels = promutil.BuildNamespaceInfoMetrics(tagsData, metrics, observedMetricLabels, options.labelsSnakeCase, logger)
	metrics = promutil.EnsureLabelConsistencyAndRemoveDuplicates(metrics, observedMetricLabels)

	registry.MustRegister(promutil.NewPrometheusCollector(metrics))
	return nil
}
