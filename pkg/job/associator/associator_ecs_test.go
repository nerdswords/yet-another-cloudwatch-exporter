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

var ecsCluster = &model.TaggedResource{
	ARN:       "arn:aws:ecs:af-south-1:123456789222:cluster/sampleCluster",
	Namespace: "AWS/ECS",
}

var ecsService1 = &model.TaggedResource{
	ARN:       "arn:aws:ecs:af-south-1:123456789222:service/sampleCluster/service1",
	Namespace: "AWS/ECS",
}

var ecsService2 = &model.TaggedResource{
	ARN:       "arn:aws:ecs:af-south-1:123456789222:service/sampleCluster/service2",
	Namespace: "AWS/ECS",
}

var ecsResources = []*model.TaggedResource{
	ecsCluster,
	ecsService1,
	ecsService2,
}

func TestAssociatorECS(t *testing.T) {
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
			name: "cluster metric should be assigned cluster resource",
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
			name: "service metric should be assigned service1 resource",
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
			name: "service metric should be assigned service2 resource",
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
