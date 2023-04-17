package promutil

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/grafana/regexp"
	prom_model "github.com/prometheus/common/model"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var Percentile = regexp.MustCompile(`^p(\d{1,2}(\.\d{0,2})?|100)$`)

func BuildNamespaceInfoMetrics(tagData []*model.TaggedResource, metrics []*PrometheusMetric, observedMetricLabels map[string]model.LabelSet, labelsSnakeCase bool, logger logging.Logger) ([]*PrometheusMetric, map[string]model.LabelSet) {
	for _, d := range tagData {
		sb := strings.Builder{}
		promNs := PromString(d.Namespace)
		if !strings.HasPrefix(promNs, "aws") {
			sb.WriteString("aws_")
		}
		sb.WriteString(promNs)
		sb.WriteString("_info")
		metricName := sb.String()

		promLabels := make(map[string]string, len(d.Tags))
		promLabels["name"] = d.ARN

		for _, tag := range d.Tags {
			ok, promTag := PromStringTag(tag.Key, labelsSnakeCase)
			if !ok {
				logger.Warn("tag name is an invalid prometheus label name", "tag", tag.Key)
				continue
			}

			labelName := "tag_" + promTag
			promLabels[labelName] = tag.Value
		}

		observedMetricLabels = recordLabelsForMetric(metricName, promLabels, observedMetricLabels)
		metrics = append(metrics, &PrometheusMetric{
			Name:   &metricName,
			Labels: promLabels,
			Value:  aws.Float64(0),
		})
	}

	return metrics, observedMetricLabels
}

func BuildMetrics(cwd []*model.CloudwatchData, labelsSnakeCase bool, observedMetricLabels map[string]model.LabelSet, logger logging.Logger) ([]*PrometheusMetric, map[string]model.LabelSet, error) {
	output := make([]*PrometheusMetric, 0)

	for _, c := range cwd {
		for _, statistic := range c.Statistics {
			var includeTimestamp bool
			if c.AddCloudwatchTimestamp != nil {
				includeTimestamp = *c.AddCloudwatchTimestamp
			}
			exportedDatapoint, timestamp, err := getDatapoint(c, statistic)
			if err != nil {
				return nil, nil, err
			}
			if exportedDatapoint == nil && (c.AddCloudwatchTimestamp == nil || !*c.AddCloudwatchTimestamp) {
				exportedDatapoint = aws.Float64(math.NaN())
				includeTimestamp = false
				if *c.NilToZero {
					exportedDatapoint = aws.Float64(0)
				}
			}

			sb := strings.Builder{}
			promNs := PromString(*c.Namespace)
			if !strings.HasPrefix(promNs, "aws") {
				sb.WriteString("aws_")
			}
			sb.WriteString(promNs)
			sb.WriteString("_")
			sb.WriteString(PromString(*c.Metric))
			sb.WriteString("_")
			sb.WriteString(PromString(statistic))
			name := sb.String()

			if exportedDatapoint != nil {
				promLabels := createPrometheusLabels(c, labelsSnakeCase, logger)
				observedMetricLabels = recordLabelsForMetric(name, promLabels, observedMetricLabels)
				output = append(output, &PrometheusMetric{
					Name:             &name,
					Labels:           promLabels,
					Value:            exportedDatapoint,
					Timestamp:        timestamp,
					IncludeTimestamp: includeTimestamp,
				})
			}
		}
	}

	return output, observedMetricLabels, nil
}

func getDatapoint(cwd *model.CloudwatchData, statistic string) (*float64, time.Time, error) {
	if cwd.GetMetricDataPoint != nil {
		return cwd.GetMetricDataPoint, *cwd.GetMetricDataTimestamps, nil
	}
	var averageDataPoints []*cloudwatch.Datapoint

	// sorting by timestamps so we can consistently export the most updated datapoint
	// assuming Timestamp field in cloudwatch.Datapoint struct is never nil
	for _, datapoint := range sortByTimestamp(cwd.Points) {
		switch {
		case statistic == "Maximum":
			if datapoint.Maximum != nil {
				return datapoint.Maximum, *datapoint.Timestamp, nil
			}
		case statistic == "Minimum":
			if datapoint.Minimum != nil {
				return datapoint.Minimum, *datapoint.Timestamp, nil
			}
		case statistic == "Sum":
			if datapoint.Sum != nil {
				return datapoint.Sum, *datapoint.Timestamp, nil
			}
		case statistic == "SampleCount":
			if datapoint.SampleCount != nil {
				return datapoint.SampleCount, *datapoint.Timestamp, nil
			}
		case statistic == "Average":
			if datapoint.Average != nil {
				averageDataPoints = append(averageDataPoints, datapoint)
			}
		case Percentile.MatchString(statistic):
			if data, ok := datapoint.ExtendedStatistics[statistic]; ok {
				return data, *datapoint.Timestamp, nil
			}
		default:
			return nil, time.Time{}, fmt.Errorf("invalid statistic requested on metric %s: %s", *cwd.Metric, statistic)
		}
	}

	if len(averageDataPoints) > 0 {
		var total float64
		var timestamp time.Time

		for _, p := range averageDataPoints {
			if p.Timestamp.After(timestamp) {
				timestamp = *p.Timestamp
			}
			total += *p.Average
		}
		average := total / float64(len(averageDataPoints))
		return &average, timestamp, nil
	}
	return nil, time.Time{}, nil
}

func sortByTimestamp(datapoints []*cloudwatch.Datapoint) []*cloudwatch.Datapoint {
	sort.Slice(datapoints, func(i, j int) bool {
		jTimestamp := *datapoints[j].Timestamp
		return datapoints[i].Timestamp.After(jTimestamp)
	})
	return datapoints
}

func createPrometheusLabels(cwd *model.CloudwatchData, labelsSnakeCase bool, logger logging.Logger) map[string]string {
	labels := make(map[string]string)
	labels["name"] = *cwd.ID
	labels["region"] = *cwd.Region
	labels["account_id"] = *cwd.AccountID

	// Inject the sfn name back as a label
	for _, dimension := range cwd.Dimensions {
		ok, promTag := PromStringTag(*dimension.Name, labelsSnakeCase)
		if !ok {
			logger.Warn("dimension name is an invalid prometheus label name", "dimension", *dimension.Name)
			continue
		}
		labels["dimension_"+promTag] = *dimension.Value
	}

	for _, label := range cwd.CustomTags {
		ok, promTag := PromStringTag(label.Key, labelsSnakeCase)
		if !ok {
			logger.Warn("custom tag name is an invalid prometheus label name", "tag", label.Key)
			continue
		}
		labels["custom_tag_"+promTag] = label.Value
	}

	for _, tag := range cwd.Tags {
		ok, promTag := PromStringTag(tag.Key, labelsSnakeCase)
		if !ok {
			logger.Warn("metric tag name is an invalid prometheus label name", "tag", tag.Key)
			continue
		}
		labels["tag_"+promTag] = tag.Value
	}

	return labels
}

// recordLabelsForMetric adds any missing labels from promLabels in to the LabelSet for the metric name and returns
// the updated observedMetricLabels
func recordLabelsForMetric(metricName string, promLabels map[string]string, observedMetricLabels map[string]model.LabelSet) map[string]model.LabelSet {
	if _, ok := observedMetricLabels[metricName]; !ok {
		observedMetricLabels[metricName] = make(model.LabelSet)
	}
	for label := range promLabels {
		if _, ok := observedMetricLabels[metricName][label]; !ok {
			observedMetricLabels[metricName][label] = struct{}{}
		}
	}

	return observedMetricLabels
}

// EnsureLabelConsistencyAndRemoveDuplicates ensures that every metric has the same set of labels based on the data
// in observedMetricLabels and that there are no duplicate metrics.
// Prometheus requires that all metrics with the same name have the same set of labels and that no duplicates are registered
func EnsureLabelConsistencyAndRemoveDuplicates(metrics []*PrometheusMetric, observedMetricLabels map[string]model.LabelSet) []*PrometheusMetric {
	metricKeys := make(map[string]struct{}, len(metrics))
	output := make([]*PrometheusMetric, 0, len(metrics))

	for _, metric := range metrics {
		for observedLabels := range observedMetricLabels[*metric.Name] {
			if _, ok := metric.Labels[observedLabels]; !ok {
				metric.Labels[observedLabels] = ""
			}
		}

		metricKey := fmt.Sprintf("%s-%d", *metric.Name, prom_model.LabelsToSignature(metric.Labels))
		if _, exists := metricKeys[metricKey]; !exists {
			metricKeys[metricKey] = struct{}{}
			output = append(output, metric)
		} else {
			DuplicateMetricsFilteredCounter.Inc()
		}
	}

	return output
}
