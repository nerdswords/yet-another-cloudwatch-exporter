package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var sagemakerTransformJobOne = &model.TaggedResource{
	ARN:       "arn:aws:sagemaker:us-west-2:123456789012:transform-job/example-transform-job-one",
	Namespace: "/aws/sagemaker/TransformJobs",
}

var sagemakerTransformJobResources = []*model.TaggedResource{
	sagemakerTransformJobOne,
}

func TestAssociatorSagemakerTransformJob(t *testing.T) {
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
			name: "1 dimension should not match but not skip",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("/aws/sagemaker/TransformJobs").ToModelDimensionsRegexp(),
				resources:        sagemakerTransformJobResources,
				metric: &model.Metric{
					MetricName: "CPUUtilization",
					Namespace:  "/aws/sagemaker/TransformJobs",
					Dimensions: []model.Dimension{
						{Name: "Host", Value: "example-transform-job-one/algo-1"},
					},
				},
			},
			expectedSkip:     false,
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
