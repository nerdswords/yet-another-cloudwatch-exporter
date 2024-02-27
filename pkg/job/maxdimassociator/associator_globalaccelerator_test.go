package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var globalAcceleratorAccelerator = &model.TaggedResource{
	ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/super-accelerator",
	Namespace: "AWS/GlobalAccelerator",
}

var globalAcceleratorListener = &model.TaggedResource{
	ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/super-accelerator/listener/some_listener",
	Namespace: "AWS/GlobalAccelerator",
}

var globalAcceleratorEndpointGroup = &model.TaggedResource{
	ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/super-accelerator/listener/some_listener/endpoint-group/eg1",
	Namespace: "AWS/GlobalAccelerator",
}

var globalAcceleratorResources = []*model.TaggedResource{
	globalAcceleratorAccelerator,
	globalAcceleratorListener,
	globalAcceleratorEndpointGroup,
}

func TestAssociatorGlobalAccelerator(t *testing.T) {
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
			name: "should match with Accelerator dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").ToModelDimensionsRegexp(),
				resources:        globalAcceleratorResources,
				metric: &model.Metric{
					MetricName: "ProcessedBytesOut",
					Namespace:  "AWS/GlobalAccelerator",
					Dimensions: []model.Dimension{
						{Name: "Accelerator", Value: "super-accelerator"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorAccelerator,
		},
		{
			name: "should match Listener with Accelerator and Listener dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").ToModelDimensionsRegexp(),
				resources:        globalAcceleratorResources,
				metric: &model.Metric{
					MetricName: "ProcessedBytesOut",
					Namespace:  "AWS/GlobalAccelerator",
					Dimensions: []model.Dimension{
						{Name: "Accelerator", Value: "super-accelerator"},
						{Name: "Listener", Value: "some_listener"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorListener,
		},
		{
			name: "should match EndpointGroup with Accelerator, Listener and EndpointGroup dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").ToModelDimensionsRegexp(),
				resources:        globalAcceleratorResources,
				metric: &model.Metric{
					MetricName: "ProcessedBytesOut",
					Namespace:  "AWS/GlobalAccelerator",
					Dimensions: []model.Dimension{
						{Name: "Accelerator", Value: "super-accelerator"},
						{Name: "Listener", Value: "some_listener"},
						{Name: "EndpointGroup", Value: "eg1"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorEndpointGroup,
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
