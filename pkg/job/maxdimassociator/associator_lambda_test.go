package maxdimassociator

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/grafana/regexp"
	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var lambdaFunction = &model.TaggedResource{
	ARN:       "arn:aws:lambda:us-east-2:123456789012:function:lambdaFunction",
	Namespace: "AWS/Lambda",
}

var lambdaResources = []*model.TaggedResource{lambdaFunction}

func TestAssociatorLambda(t *testing.T) {
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
			name: "should match with FunctionName dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Lambda").DimensionRegexps,
				resources:        lambdaResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("Invocations"),
					Namespace:  aws.String("AWS/Lambda"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("FunctionName"), Value: aws.String("lambdaFunction")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: lambdaFunction,
		},
		{
			name: "should skip with unmatched FunctionName dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Lambda").DimensionRegexps,
				resources:        lambdaResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("Invocations"),
					Namespace:  aws.String("AWS/Lambda"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("FunctionName"), Value: aws.String("anotherLambdaFunction")},
					},
				},
			},
			expectedSkip:     true,
			expectedResource: nil,
		},
		{
			name: "should match with FunctionName and Resource dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Lambda").DimensionRegexps,
				resources:        lambdaResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("Invocations"),
					Namespace:  aws.String("AWS/Lambda"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("FunctionName"), Value: aws.String("lambdaFunction")},
						{Name: aws.String("Resource"), Value: aws.String("lambdaFunction")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: lambdaFunction,
		},
		{
			name: "should not skip when empty dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Lambda").DimensionRegexps,
				resources:        lambdaResources,
				metric: &cloudwatch.Metric{
					MetricName: aws.String("Invocations"),
					Namespace:  aws.String("AWS/Lambda"),
					Dimensions: []*cloudwatch.Dimension{},
				},
			},
			expectedSkip:     false,
			expectedResource: nil,
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
