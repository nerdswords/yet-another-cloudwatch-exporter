package promutil

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitString(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  "GlobalTopicCount",
			output: "Global.Topic.Count",
		},
		{
			input:  "CPUUtilization",
			output: "CPUUtilization",
		},
		{
			input:  "StatusCheckFailed_Instance",
			output: "Status.Check.Failed_Instance",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.output, splitString(tc.input))
	}
}

func TestSanitize(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  "Global.Topic.Count",
			output: "Global_Topic_Count",
		},
		{
			input:  "Status.Check.Failed_Instance",
			output: "Status_Check_Failed_Instance",
		},
		{
			input:  "IHaveA%Sign",
			output: "IHaveA_percentSign",
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.output, sanitize(tc.input))
	}
}

func TestPromStringTag(t *testing.T) {
	testCases := []struct {
		name        string
		label       string
		toSnakeCase bool
		ok          bool
		out         string
	}{
		{
			name:        "valid",
			label:       "labelName",
			toSnakeCase: false,
			ok:          true,
			out:         "labelName",
		},
		{
			name:        "valid, convert to snake case",
			label:       "labelName",
			toSnakeCase: true,
			ok:          true,
			out:         "label_name",
		},
		{
			name:        "valid (snake case)",
			label:       "label_name",
			toSnakeCase: false,
			ok:          true,
			out:         "label_name",
		},
		{
			name:        "valid (snake case) unchanged",
			label:       "label_name",
			toSnakeCase: true,
			ok:          true,
			out:         "label_name",
		},
		{
			name:        "invalid chars",
			label:       "invalidChars@$",
			toSnakeCase: false,
			ok:          false,
			out:         "",
		},
		{
			name:        "invalid chars, convert to snake case",
			label:       "invalidChars@$",
			toSnakeCase: true,
			ok:          false,
			out:         "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok, out := PromStringTag(tc.label, tc.toSnakeCase)
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.out, out)
			}
		})
	}
}

func TestPrometheusMetric(t *testing.T) {
	t.Run("NewPrometheusMetric panics with wrong label size", func(t *testing.T) {
		require.Panics(t, func() {
			NewPrometheusMetric("metric1", []string{"key1"}, []string{}, 1.0)
		})
		require.Panics(t, func() {
			NewPrometheusMetric("metric1", []string{}, []string{"label1"}, 1.0)
		})
		require.Panics(t, func() {
			NewPrometheusMetric("metric1", []string{"key1", "key2"}, []string{"label1"}, 1.0)
		})
		require.Panics(t, func() {
			NewPrometheusMetric("metric1", []string{"key1"}, []string{"label1", "label2"}, 1.0)
		})
	})

	t.Run("NewPrometheusMetric sorts labels", func(t *testing.T) {
		metric := NewPrometheusMetric("metric", []string{"key2", "key1"}, []string{"value2", "value1"}, 1.0)
		keys, vals := metric.Labels()
		require.Equal(t, []string{"key1", "key2"}, keys)
		require.Equal(t, []string{"value1", "value2"}, vals)
	})

	t.Run("AddIfMissingLabelPair keeps labels sorted", func(t *testing.T) {
		metric := NewPrometheusMetric("metric", []string{"key2"}, []string{"value2"}, 1.0)
		metric.AddIfMissingLabelPair("key1", "value1")
		keys, vals := metric.Labels()
		require.Equal(t, []string{"key1", "key2"}, keys)
		require.Equal(t, []string{"value1", "value2"}, vals)
	})

	t.Run("RemoveDuplicateLabels", func(t *testing.T) {
		metric := NewPrometheusMetric("metric", []string{"key1", "key2", "key1", "key3"}, []string{"value-key1", "value-key2", "value-dup-key1", "value-key3"}, 1.0)
		require.Equal(t, 4, metric.LabelsLen())
		duplicates := metric.RemoveDuplicateLabels()
		keys, vals := metric.Labels()
		require.Equal(t, []string{"key1", "key2", "key3"}, keys)
		require.Equal(t, []string{"value-key1", "value-key2", "value-key3"}, vals)
		require.Equal(t, []string{"key1"}, duplicates)
	})
}

func TestNewPrometheusCollector_CanReportMetricsAndErrors(t *testing.T) {
	metrics := []*PrometheusMetric{
		NewPrometheusMetric("this*is*not*valid", []string{}, []string{}, 0),
		NewPrometheusMetric("this_is_valid", []string{"key"}, []string{"value1"}, 0),
	}
	collector := NewPrometheusCollector(metrics)
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))
	families, err := registry.Gather()
	assert.Error(t, err)
	assert.Len(t, families, 1)
	family := families[0]
	assert.Equal(t, "this_is_valid", family.GetName())
}

func TestNewPrometheusCollector_CanReportMetrics(t *testing.T) {
	ts := time.Now()

	labelSet1 := map[string]string{"key1": "value", "key2": "value", "key3": "value"}
	labelSet2 := map[string]string{"key2": "out", "key3": "of", "key1": "order"}
	labelSet3 := map[string]string{"key2": "out", "key1": "of", "key3": "order"}
	metrics := []*PrometheusMetric{
		NewPrometheusMetric(
			"metric_with_labels",
			[]string{"key1", "key2", "key3"},
			[]string{"value", "value", "value"},
			1,
		),
		NewPrometheusMetric(
			"metric_with_labels",
			[]string{"key2", "key3", "key1"},
			[]string{"out", "of", "order"},
			2,
		),
		NewPrometheusMetric(
			"metric_with_labels",
			[]string{"key2", "key1", "key3"},
			[]string{"out", "of", "order"},
			3,
		),
		NewPrometheusMetricWithTimestamp(
			"metric_with_timestamp",
			[]string{},
			[]string{},
			1,
			true,
			ts,
		),
	}

	collector := NewPrometheusCollector(metrics)
	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))
	families, err := registry.Gather()
	assert.NoError(t, err)
	assert.Len(t, families, 2)

	var metricWithLabels *dto.MetricFamily
	var metricWithTs *dto.MetricFamily

	for _, metricFamily := range families {
		assert.Equal(t, dto.MetricType_GAUGE, metricFamily.GetType())

		switch {
		case metricFamily.GetName() == "metric_with_labels":
			metricWithLabels = metricFamily
		case metricFamily.GetName() == "metric_with_timestamp":
			metricWithTs = metricFamily
		default:
			require.Failf(t, "Encountered an unexpected metric family %s", metricFamily.GetName())
		}
	}
	require.NotNil(t, metricWithLabels)
	require.NotNil(t, metricWithTs)

	assert.Len(t, metricWithLabels.Metric, 3)
	for _, metric := range metricWithLabels.Metric {
		assert.Len(t, metric.Label, 3)
		var labelSetToMatch map[string]string
		switch *metric.Gauge.Value {
		case 1.0:
			labelSetToMatch = labelSet1
		case 2.0:
			labelSetToMatch = labelSet2
		case 3.0:
			labelSetToMatch = labelSet3
		default:
			require.Fail(t, "Encountered an metric value value %v", *metric.Gauge.Value)
		}

		for _, labelPairs := range metric.Label {
			require.Contains(t, labelSetToMatch, *labelPairs.Name)
			require.Equal(t, labelSetToMatch[*labelPairs.Name], *labelPairs.Value)
		}
	}

	require.Len(t, metricWithTs.Metric, 1)
	tsMetric := metricWithTs.Metric[0]
	assert.Equal(t, ts.UnixMilli(), *tsMetric.TimestampMs)
	assert.Equal(t, 1.0, *tsMetric.Gauge.Value)
}

func Benchmark_NewPrometheusCollector(b *testing.B) {
	metrics := []*PrometheusMetric{
		NewPrometheusMetric("metric1", []string{"key1"}, []string{"value11"}, 1.0),
		NewPrometheusMetric("metric1", []string{"key1"}, []string{"value12"}, 1.0),
		NewPrometheusMetric("metric2", []string{"key2"}, []string{"value21"}, 2.0),
		NewPrometheusMetric("metric2", []string{"key2"}, []string{"value22"}, 2.0),
		NewPrometheusMetric("metric3", []string{"key3"}, []string{"value31"}, 3.0),
		NewPrometheusMetric("metric3", []string{"key3"}, []string{"value32"}, 3.0),
		NewPrometheusMetric("metric4", []string{"key4"}, []string{"value41"}, 4.0),
		NewPrometheusMetric("metric4", []string{"key4"}, []string{"value42"}, 4.0),
		NewPrometheusMetric("metric5", []string{"key5"}, []string{"value51"}, 5.0),
		NewPrometheusMetric("metric5", []string{"key5"}, []string{"value52"}, 5.0),
	}

	var collector *PrometheusCollector

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector = NewPrometheusCollector(metrics)
	}

	registry := prometheus.NewRegistry()
	require.NoError(b, registry.Register(collector))
}
