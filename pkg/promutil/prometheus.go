package promutil

import (
	"strings"
	"time"

	"github.com/grafana/regexp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

var (
	CloudwatchAPIErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "yace_cloudwatch_request_errors",
		Help: "Help is not implemented yet.",
	}, []string{"api_name"})
	CloudwatchAPICounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "yace_cloudwatch_requests_total",
		Help: "Number of calls made to the CloudWatch APIs",
	}, []string{"api_name"})
	CloudwatchGetMetricDataAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_getmetricdata_requests_total",
		Help: "DEPRECATED: replaced by yace_cloudwatch_requests_total with api_name label",
	})
	CloudwatchGetMetricDataAPIMetricsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_getmetricdata_metrics_requested_total",
		Help: "Number of metrics requested from the CloudWatch GetMetricData API which is how AWS bills",
	})
	CloudwatchGetMetricStatisticsAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_getmetricstatistics_requests_total",
		Help: "DEPRECATED: replaced by yace_cloudwatch_requests_total with api_name label",
	})
	ResourceGroupTaggingAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_resourcegrouptaggingapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	AutoScalingAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_autoscalingapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	TargetGroupsAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_targetgroupapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	APIGatewayAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_apigatewayapi_requests_total",
	})
	APIGatewayAPIV2Counter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_apigatewayapiv2_requests_total",
	})
	Ec2APICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_ec2api_requests_total",
		Help: "Help is not implemented yet.",
	})
	ShieldAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_shieldapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	ManagedPrometheusAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_managedprometheusapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	StoragegatewayAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_storagegatewayapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	DmsAPICounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_dmsapi_requests_total",
		Help: "Help is not implemented yet.",
	})
	DuplicateMetricsFilteredCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yace_cloudwatch_duplicate_metrics_filtered",
		Help: "Help is not implemented yet.",
	})
)

var replacer = strings.NewReplacer(
	" ", "_",
	",", "_",
	"\t", "_",
	"/", "_",
	"\\", "_",
	".", "_",
	"-", "_",
	":", "_",
	"=", "_",
	"“", "_",
	"@", "_",
	"<", "_",
	">", "_",
	"%", "_percent",
)
var splitRegexp = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type PrometheusMetric struct {
	Name             *string
	Labels           map[string]string
	Value            float64
	IncludeTimestamp bool
	Timestamp        time.Time
}

type PrometheusCollector struct {
	metrics []*PrometheusMetric
}

func NewPrometheusCollector(metrics []*PrometheusMetric) *PrometheusCollector {
	return &PrometheusCollector{
		metrics: metrics,
	}
}

func (p *PrometheusCollector) Describe(_ chan<- *prometheus.Desc) {
	// The exporter produces a dynamic set of metrics and the docs for prometheus.Collector Describe say
	// 	Sending no descriptor at all marks the Collector as “unchecked”,
	// 	i.e. no checks will be performed at registration time, and the
	// 	Collector may yield any Metric it sees fit in its Collect method.
	// Based on our use an "unchecked" collector is perfectly fine
}

func (p *PrometheusCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, metric := range p.metrics {
		metrics <- createMetric(metric)
	}
}

func createMetric(metric *PrometheusMetric) prometheus.Metric {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        *metric.Name,
		Help:        "Help is not implemented yet.",
		ConstLabels: metric.Labels,
	})

	gauge.Set(metric.Value)

	if !metric.IncludeTimestamp {
		return gauge
	}

	return prometheus.NewMetricWithTimestamp(metric.Timestamp, gauge)
}

func PromString(text string) string {
	text = splitString(text)
	return strings.ToLower(sanitize(text))
}

func PromStringTag(text string, labelsSnakeCase bool) (bool, string) {
	var s string
	if labelsSnakeCase {
		s = PromString(text)
	} else {
		s = sanitize(text)
	}
	return model.LabelName(s).IsValid(), s
}

func sanitize(text string) string {
	return replacer.Replace(text)
}

func splitString(text string) string {
	return splitRegexp.ReplaceAllString(text, `$1.$2`)
}
