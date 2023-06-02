package maxdimassociator

import (
	"testing"

	"github.com/grafana/regexp"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var sagemakerEndpointInvocationOne = &model.TaggedResource{
	ARN:       "arn:aws:sagemaker:us-west-2:123456789012:endpoint/example-endpoint-one",
	Namespace: "AWS/SageMaker",
}

var sagemakerEndpointInvocationTwo = &model.TaggedResource{
	ARN:       "arn:aws:sagemaker:us-west-2:123456789012:endpoint/example-endpoint-two",
	Namespace: "AWS/SageMaker",
}

var sagemakerInvocationResources = []*model.TaggedResource{
	sagemakerEndpointInvocationOne,
	sagemakerEndpointInvocationTwo,
}

func TestAssociatorSagemaker(t *testing.T) {
	type args struct {
		dimensionRegexps []*regexp.Regexp
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
			name: "3 dimensions should match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/SageMaker").DimensionRegexps,
				resources:        sagemakerInvocationResources,
				metric: &model.Metric{
					MetricName: "Invocations",
					Namespace:  "AWS/SageMaker",
					Dimensions: []*model.Dimension{
						{Name: "EndpointName", Value: "example-endpoint-one"},
						{Name: "VariantName", Value: "example-endpoint-one-variant-one"},
						{Name: "EndpointConfigName", Value: "example-endpoint-one-endpoint-config"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: sagemakerEndpointInvocationOne,
		},
		{
			name: "2 dimenions should match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/SageMaker").DimensionRegexps,
				resources:        sagemakerInvocationResources,
				metric: &model.Metric{
					MetricName: "Invocations",
					Namespace:  "AWS/SageMaker",
					Dimensions: []*model.Dimension{
						{Name: "EndpointName", Value: "example-endpoint-two"},
						{Name: "VariantName", Value: "example-endpoint-two-variant-one"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: sagemakerEndpointInvocationTwo,
		},
		{
			name: "2 dimenions should not match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/SageMaker").DimensionRegexps,
				resources:        sagemakerInvocationResources,
				metric: &model.Metric{
					MetricName: "Invocations",
					Namespace:  "AWS/SageMaker",
					Dimensions: []*model.Dimension{
						{Name: "EndpointName", Value: "example-endpoint-three"},
						{Name: "VariantName", Value: "example-endpoint-three-variant-one"},
					},
				},
			},
			expectedSkip:     true,
			expectedResource: nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			associator := NewAssociator(tc.args.dimensionRegexps, tc.args.resources)
			res, skip := associator.AssociateMetricToResource(tc.args.metric)
			require.Equal(t, tc.expectedSkip, skip)
			require.Equal(t, tc.expectedResource, res)
		})
	}
}
