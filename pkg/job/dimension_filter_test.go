package job

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var someEC2Instance = &model.TaggedResource{
	ARN:       "arn:aws:ec2:us-east-1:123456789012:instance/i-bla",
	Namespace: "AWS/EC2",
	Region:    "us-east-2",
	Tags:      []model.Tag{{"name", "test-instance"}},
}

var globalAcceleratorAccelerator = &model.TaggedResource{
	ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/super-accelerator",
	Namespace: "AWS/GlobalAccelerator",
	Region:    "us-east-2",
}
var globalAcceleratorListener = &model.TaggedResource{
	ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/super-accelerator/listener/some_listener",
	Namespace: "AWS/GlobalAccelerator",
	Region:    "us-east-2",
}
var globalAcceleratorEndpointGroup = &model.TaggedResource{
	ARN:       "arn:aws:globalaccelerator::012345678901:accelerator/super-accelerator/listener/some_listener/endpoint-group/eg1",
	Namespace: "AWS/GlobalAccelerator",
	Region:    "us-east-2",
}
var globalAcceleratorResources = []*model.TaggedResource{
	globalAcceleratorAccelerator,
	globalAcceleratorListener,
	globalAcceleratorEndpointGroup,
}

var ecsCluster = &model.TaggedResource{
	ARN:       "arn:aws:ecs:af-south-1:123456789222:cluster/sampleCluster",
	Namespace: "AWS/ECS",
	Region:    "af-south-1",
}
var ecsService1 = &model.TaggedResource{
	ARN:       "arn:aws:ecs:af-south-1:123456789222:service/sampleCluster/service1",
	Namespace: "AWS/ECS",
	Region:    "af-south-1",
}

var ecsService2 = &model.TaggedResource{
	ARN:       "arn:aws:ecs:af-south-1:123456789222:service/sampleCluster/service1",
	Namespace: "AWS/ECS",
	Region:    "af-south-1",
}
var ecsResources = []*model.TaggedResource{
	ecsCluster,
	ecsService1,
	ecsService2,
}

func TestDimensionsFilter(t *testing.T) {
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
	for _, tc := range []testCase{
		{
			name: "no resource match found if there are no regex filters",
			args: args{
				dimensionRegexps: nil,
				resources: []*model.TaggedResource{
					someEC2Instance,
				},
				metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/EC2"),
					MetricName: aws.String("CPUUtilization"),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("InstanceId"),
							Value: aws.String("i-bla"),
						},
					},
				},
			},
			expectedSkip: true,
		},
		{
			name: "pass through",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/EC2").DimensionRegexps,
				resources: []*model.TaggedResource{
					someEC2Instance,
				},
				metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/EC2"),
					MetricName: aws.String("CPUUtilization"),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("InstanceId"),
							Value: aws.String("i-bla"),
						},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: someEC2Instance,
		},
		{
			name: "metric dropped",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/EC2").DimensionRegexps,
				resources: []*model.TaggedResource{
					someEC2Instance,
				},
				metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/EC2"),
					MetricName: aws.String("CPUUtilization"),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("InstanceId"),
							Value: aws.String("i-not-bla"),
						},
					},
				},
			},
			expectedSkip: true,
		},
		// The tests below exercise cases in which there's a metrics that might apply to more than one resource
		// depending on the set of dimensions it has.
		{
			name: "multiple ga resources, should match accelerator",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GlobalAccelerator").DimensionRegexps,
				resources:        globalAcceleratorResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("ProcessedBytesOut"),
					Namespace:  aws.String("AWS/GlobalAccelerator"),
					Dimensions: []*cloudwatch.Dimension{{
						Name: aws.String("Accelerator"), Value: aws.String("super-accelerator"),
					}},
				},
			},
			expectedSkip:     false,
			expectedResource: globalAcceleratorAccelerator,
		},
		{
			name: "multiple ga resources, should match listener",
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
			name: "multiple ga resources, should match endpoint group",
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
		{
			name: "multiple ecs resources, cluster metric should be assigned cluster resource",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ECS").DimensionRegexps,
				resources:        ecsResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("MemoryReservation"),
					Namespace:  aws.String("AWS/ECS"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("ClusterName"), Value: aws.String("sampleCluster")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: ecsCluster,
		},
		{
			name: "multiple ecs resources, service metric should be assigned service1 resource",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ECS").DimensionRegexps,
				resources:        ecsResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("CPUUtilization"),
					Namespace:  aws.String("AWS/ECS"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("ClusterName"), Value: aws.String("sampleCluster")},
						{Name: aws.String("ServiceName"), Value: aws.String("service1")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: ecsService1,
		},
		{
			name: "multiple ecs resources, service metric should be assigned service2 resource",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ECS").DimensionRegexps,
				resources:        ecsResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("CPUUtilization"),
					Namespace:  aws.String("AWS/ECS"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("ClusterName"), Value: aws.String("sampleCluster")},
						{Name: aws.String("ServiceName"), Value: aws.String("service2")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: ecsService2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			filter := buildDimensionsFilter(tc.args.dimensionRegexps, tc.args.resources)
			res, skip := filter.filterAndFindMatchingResource(tc.args.metric)
			require.Equal(t, tc.expectedSkip, skip)
			require.Equal(t, tc.expectedResource, res)
		})
	}
}
