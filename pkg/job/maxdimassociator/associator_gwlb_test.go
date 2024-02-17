package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var gatewayLoadBalancer1 = &model.TaggedResource{
	ARN:       "arn:aws:elasticloadbalancing:eu-central-1:123456789012:loadbalancer/gwy/gwlb-1/4a049e69add14452",
	Namespace: "AWS/GatewayELB",
}

var gatewayLoadBalancer2 = &model.TaggedResource{
	ARN:       "arn:aws:elasticloadbalancing:eu-central-1:123456789012:loadbalancer/gwy/gwlb-2/a96cc19724cf1a87",
	Namespace: "AWS/GatewayELB",
}

var targetGroup1 = &model.TaggedResource{
	ARN:       "arn:aws:elasticloadbalancing:eu-central-1:123456789012:targetgroup/gwlb-target-group-1/012e9f368748cd345c",
	Namespace: "AWS/GatewayELB",
}

var gatewayLoadBalancerResources = []*model.TaggedResource{
	gatewayLoadBalancer1,
	gatewayLoadBalancer2,
	targetGroup1,
}

func TestAssociatorGwlb(t *testing.T) {
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
			name: "should match with gateway loadbalancer one dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GatewayELB").ToModelDimensionsRegexp(),
				resources:        gatewayLoadBalancerResources,
				metric: &model.Metric{
					MetricName: "HealthyHostCount",
					Namespace:  "AWS/GatewayELB",
					Dimensions: []model.Dimension{
						{Name: "LoadBalancer", Value: "gwy/gwlb-1/4a049e69add14452"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: gatewayLoadBalancer1,
		},
		{
			name: "should match with gateway loadbalancer target group two dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GatewayELB").ToModelDimensionsRegexp(),
				resources:        gatewayLoadBalancerResources,
				metric: &model.Metric{
					MetricName: "HealthyHostCount",
					Namespace:  "AWS/GatewayELB",
					Dimensions: []model.Dimension{
						{Name: "LoadBalancer", Value: "gwy/gwlb-1/4a049e69add14452"},
						{Name: "TargetGroup", Value: "targetgroup/gwlb-target-group-1/012e9f368748cd345c"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: targetGroup1,
		},
		{
			name: "should not match with any gateway loadbalancer",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/GatewayELB").ToModelDimensionsRegexp(),
				resources:        gatewayLoadBalancerResources,
				metric: &model.Metric{
					MetricName: "HealthyHostCount",
					Namespace:  "AWS/GatewayELB",
					Dimensions: []model.Dimension{
						{Name: "LoadBalancer", Value: "gwy/non-existing-gwlb/a96cc19724cf1a87"},
					},
				},
			},
			expectedSkip:     true,
			expectedResource: nil,
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
