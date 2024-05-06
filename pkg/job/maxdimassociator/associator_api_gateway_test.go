package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var apiGatewayV1 = &model.TaggedResource{
	ARN:       "arn:aws:apigateway:us-east-2::/restapis/test-api",
	Namespace: "AWS/ApiGateway",
}

var apiGatewayV1Stage = &model.TaggedResource{
	ARN:       "arn:aws:apigateway:us-east-2::/restapis/test-api/stages/test",
	Namespace: "AWS/ApiGateway",
}

var apiGatewayV2 = &model.TaggedResource{
	ARN:       "arn:aws:apigateway:us-east-2::/apis/98765fghij",
	Namespace: "AWS/ApiGateway",
}

var apiGatewayV2Stage = &model.TaggedResource{
	ARN:       "arn:aws:apigateway:us-east-2::/apis/98765fghij/stages/$default",
	Namespace: "AWS/ApiGateway",
}

var apiGatewayResources = []*model.TaggedResource{apiGatewayV1, apiGatewayV1Stage, apiGatewayV2, apiGatewayV2Stage}

func TestAssociatorAPIGateway(t *testing.T) {
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
			name: "should match API Gateway V2 with ApiId dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ApiGateway").ToModelDimensionsRegexp(),
				resources:        apiGatewayResources,
				metric: &model.Metric{
					MetricName: "5xx",
					Namespace:  "AWS/ApiGateway",
					Dimensions: []model.Dimension{
						{Name: "ApiId", Value: "98765fghij"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: apiGatewayV2,
		},
		{
			name: "should match API Gateway V2 with ApiId and Stage dimensions",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ApiGateway").ToModelDimensionsRegexp(),
				resources:        apiGatewayResources,
				metric: &model.Metric{
					MetricName: "5xx",
					Namespace:  "AWS/ApiGateway",
					Dimensions: []model.Dimension{
						{Name: "ApiId", Value: "98765fghij"},
						{Name: "Stage", Value: "$default"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: apiGatewayV2Stage,
		},
		{
			name: "should match API Gateway V1 with ApiName dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ApiGateway").ToModelDimensionsRegexp(),
				resources:        apiGatewayResources,
				metric: &model.Metric{
					MetricName: "5xx",
					Namespace:  "AWS/ApiGateway",
					Dimensions: []model.Dimension{
						{Name: "ApiName", Value: "test-api"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: apiGatewayV1,
		},
		{
			name: "should match API Gateway V1 with ApiName and Stage dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ApiGateway").ToModelDimensionsRegexp(),
				resources:        apiGatewayResources,
				metric: &model.Metric{
					MetricName: "5xx",
					Namespace:  "AWS/ApiGateway",
					Dimensions: []model.Dimension{
						{Name: "ApiName", Value: "test-api"},
						{Name: "Stage", Value: "test"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: apiGatewayV1Stage,
		},
		{
			name: "should match API Gateway V1 with ApiName (Stage is not matched)",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/ApiGateway").ToModelDimensionsRegexp(),
				resources:        apiGatewayResources,
				metric: &model.Metric{
					MetricName: "5xx",
					Namespace:  "AWS/ApiGateway",
					Dimensions: []model.Dimension{
						{Name: "ApiName", Value: "test-api"},
						{Name: "Stage", Value: "dev"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: apiGatewayV1,
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
