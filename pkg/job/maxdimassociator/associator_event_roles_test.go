package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var eventRule0 = &model.TaggedResource{
	ARN:       "arn:aws:events:eu-central-1:112246171613:rule/event-bus-name/rule-name",
	Namespace: "AWS/Events",
}

var eventRuleResources = []*model.TaggedResource{
	eventRule0,
}

func TestAssociatorEventRule(t *testing.T) {
	type args struct {
		dimensionRegexps []model.DimensionsRegexp
		resources        []*model.TaggedResource
		metric           *model.Metric
	}

	type testCase struct {
		name             string
		args             args
		expectedSkip     bool
		expectedResource *model.TaggedResource
	}

	testcases := []testCase{
		{
			name: "2 dimensions should match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Events").ToModelDimensionsRegexp(),
				resources:        eventRuleResources,
				metric: &model.Metric{
					MetricName: "Invocations",
					Namespace:  "AWS/Events",
					Dimensions: []model.Dimension{
						{Name: "EventBusName", Value: "event-bus-name"},
						{Name: "RuleName", Value: "rule-name"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: eventRule0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			associator := NewAssociator(logging.NewNopLogger(), tc.args.dimensionRegexps, tc.args.resources)
			res, skip := associator.AssociateMetricToResource(tc.args.metric)
			require.Equal(t, tc.expectedSkip, skip)
			require.Equal(t, tc.expectedResource, res)
		})
	}
}
