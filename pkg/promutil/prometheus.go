package promutil

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	prom_model "github.com/prometheus/common/model"

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

// labelPair joins two slices of keys and values
// and allows simultaneous sorting.
type labelPair struct {
	keys []string
	vals []string
}

func (p labelPair) Len() int {
	return len(p.keys)
}

func (p labelPair) Swap(i, j int) {
	p.keys[i], p.keys[j] = p.keys[j], p.keys[i]
	p.vals[i], p.vals[j] = p.vals[j], p.vals[i]
}

func (p labelPair) Less(i, j int) bool {
	return p.keys[i] < p.keys[j]
}

// PrometheusMetric is a precursor of prometheus.Metric.
// Labels are kept sorted by key to ensure consistent ordering.
type PrometheusMetric struct {
	name             string
	labels           labelPair
	value            float64
	includeTimestamp bool
	timestamp        time.Time
}

func NewPrometheusMetric(name string, labelKeys, labelValues []string, value float64) *PrometheusMetric {
	return NewPrometheusMetricWithTimestamp(name, labelKeys, labelValues, value, false, time.Time{})
}

func NewPrometheusMetricWithTimestamp(name string, labelKeys, labelValues []string, value float64, includeTimestamp bool, timestamp time.Time) *PrometheusMetric {
	if len(labelKeys) != len(labelValues) {
		panic("labelKeys and labelValues have different length")
	}

	labels := labelPair{labelKeys, labelValues}
	sort.Sort(labels)

	return &PrometheusMetric{
		name:             name,
		labels:           labels,
		value:            value,
		includeTimestamp: includeTimestamp,
		timestamp:        timestamp,
	}
}

func (p *PrometheusMetric) Name() string {
	return p.name
}

func (p *PrometheusMetric) Labels() ([]string, []string) {
	return p.labels.keys, p.labels.vals
}

func (p *PrometheusMetric) Value() float64 {
	return p.value
}

// SetValue should be used only for testing
func (p *PrometheusMetric) SetValue(v float64) {
	p.value = v
}

func (p *PrometheusMetric) IncludeTimestamp() bool {
	return p.includeTimestamp
}

func (p *PrometheusMetric) Timestamp() time.Time {
	return p.timestamp
}

var separatorByteSlice = []byte{prom_model.SeparatorByte}

// LabelsSignature returns a hash of the labels. It emulates
// prometheus' LabelsToSignature implementation but works on
// labelPair instead of map[string]string. Assumes that
// the labels are sorted.
func (p *PrometheusMetric) LabelsSignature() uint64 {
	xxh := xxhash.New()
	for i, key := range p.labels.keys {
		_, _ = xxh.WriteString(key)
		_, _ = xxh.Write(separatorByteSlice)
		_, _ = xxh.WriteString(p.labels.vals[i])
		_, _ = xxh.Write(separatorByteSlice)
	}
	return xxh.Sum64()
}

func (p *PrometheusMetric) AddIfMissingLabelPair(key, val string) {
	// TODO(cristian): might use binary search here
	if !slices.Contains(p.labels.keys, key) {
		p.labels.keys = append(p.labels.keys, key)
		p.labels.vals = append(p.labels.vals, val)
		sort.Sort(p.labels)
	}
}

type PrometheusCollector struct {
	metrics []prometheus.Metric
}

func NewPrometheusCollector(metrics []*PrometheusMetric) *PrometheusCollector {
	return &PrometheusCollector{
		metrics: toConstMetrics(metrics),
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
		metrics <- metric
	}
}

func toConstMetrics(metrics []*PrometheusMetric) []prometheus.Metric {
	// Keep a fast lookup map for the prometheus.Desc of a metric which can be reused for each metric with
	// the same name and the expected label key order of a particular metric name (sorting of keys and values
	// is guaranteed by the implementation of PrometheusMetric).
	// The prometheus.Desc object is expensive to create and being able to reuse it for all metrics with the same name
	// results in large performance gain.
	metricToDesc := map[string]*prometheus.Desc{}

	result := make([]prometheus.Metric, 0, len(metrics))
	for _, metric := range metrics {
		metricName := metric.Name()
		labelKeys, labelValues := metric.Labels()

		if _, ok := metricToDesc[metricName]; !ok {
			metricToDesc[metricName] = prometheus.NewDesc(metricName, "Help is not implemented yet.", labelKeys, nil)
		}
		metricsDesc := metricToDesc[metricName]

		promMetric, err := prometheus.NewConstMetric(metricsDesc, prometheus.GaugeValue, metric.Value(), labelValues...)
		if err != nil {
			// If for whatever reason the metric or metricsDesc is considered invalid this will ensure the error is
			// reported through the collector
			promMetric = prometheus.NewInvalidMetric(metricsDesc, err)
		} else if metric.IncludeTimestamp() {
			promMetric = prometheus.NewMetricWithTimestamp(metric.Timestamp(), promMetric)
		}

		result = append(result, promMetric)
	}

	return result
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

// sanitize replaces some invalid chars with an underscore
func sanitize(text string) string {
	if strings.ContainsAny(text, "“%") {
		// fallback to the replacer for complex cases:
		// - '“' is non-ascii rune
		// - '%' is replaced with a whole string
		return replacer.Replace(text)
	}

	b := []byte(text)
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case ' ', ',', '\t', '/', '\\', '.', '-', ':', '=', '@', '<', '>':
			b[i] = '_'
		}
	}
	return string(b)
}

// splitString replaces consecutive occurrences of a lowercase and uppercase letter,
// or a number and an upper case letter, by putting a dot between the two chars.
//
// This is an optimised version of the original implementation:
//
//	  var splitRegexp = regexp.MustCompile(`([a-z0-9])([A-Z])`)
//
//		func splitString(text string) string {
//		  return splitRegexp.ReplaceAllString(text, `$1.$2`)
//		}
func splitString(text string) string {
	sb := strings.Builder{}
	sb.Grow(len(text) + 4) // make some room for replacements

	i := 0
	for i < len(text) {
		c := text[i]
		sb.WriteByte(c)
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			if i < (len(text) - 1) {
				c = text[i+1]
				if c >= 'A' && c <= 'Z' {
					sb.WriteByte('.')
					sb.WriteByte(c)
					i++
				}
			}
		}
		i++
	}
	return sb.String()
}
