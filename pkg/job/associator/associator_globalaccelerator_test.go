package associator

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/grafana/regexp"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
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
		dimensionRegexps []*regexp.Regexp
		resources        []*model.TaggedResource
		metric           *cloudwatch.Metric
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
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").DimensionRegexps,
				resources:        globalAcceleratorResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("ProcessedBytesOut"),
					Namespace:  aws.String("AWS/GlobalAccelerator"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("Accelerator"), Value: aws.String("super-accelerator")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorAccelerator,
		},
		{
			name: "should match Listener with Accelerator and Listener dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").DimensionRegexps,
				resources:        globalAcceleratorResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("ProcessedBytesOut"),
					Namespace:  aws.String("AWS/GlobalAccelerator"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("Accelerator"), Value: aws.String("super-accelerator")},
						{Name: aws.String("Listener"), Value: aws.String("some_listener")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorListener,
		},
		{
			name: "should match EndpointGroup with Accelerator, Listener and EndpointGroup dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").DimensionRegexps,
				resources:        globalAcceleratorResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("ProcessedBytesOut"),
					Namespace:  aws.String("AWS/GlobalAccelerator"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("Accelerator"), Value: aws.String("super-accelerator")},
						{Name: aws.String("Listener"), Value: aws.String("some_listener")},
						{Name: aws.String("EndpointGroup"), Value: aws.String("eg1")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorEndpointGroup,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			associator := NewAssociator(tc.args.dimensionRegexps, tc.args.resources)
			res, skip := associator.AssociateMetricsToResources(tc.args.metric)
			require.Equal(t, tc.expectedSkip, skip)
			require.Equal(t, tc.expectedResource, res)
		})
	}
}
