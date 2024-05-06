package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var protectedResources1 = &model.TaggedResource{
	ARN:       "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123",
	Namespace: "AWS/DDoSProtection",
}

var protectedResources2 = &model.TaggedResource{
	ARN:       "arn:aws:ec2:us-east-1:123456789012:instance/i-def456",
	Namespace: "AWS/DDoSProtection",
}

var protectedResources = []*model.TaggedResource{
	protectedResources1,
	protectedResources2,
}

func TestAssociatorDDoSProtection(t *testing.T) {
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
			name: "should match with ResourceArn dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/DDoSProtection").ToModelDimensionsRegexp(),
				resources:        protectedResources,
				metric: &model.Metric{
					Namespace:  "AWS/DDoSProtection",
					MetricName: "CPUUtilization",
					Dimensions: []model.Dimension{
						{Name: "ResourceArn", Value: "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: protectedResources1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			associator := NewAssociator(logging.NewNopLogger(), tc.args.dimensionRegexps, tc.args.resources)
			res, skip := associator.AssociateMetricToResource(tc.args.metric)
			assert.Equal(t, tc.expectedSkip, skip)
			assert.Equal(t, tc.expectedResource, res)
		})
	}
}
