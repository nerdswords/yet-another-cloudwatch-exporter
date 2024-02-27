package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var sagemakerInfRecJobOne = &model.TaggedResource{
	ARN:       "arn:aws:sagemaker:us-west-2:123456789012:inference-recommendations-job/example-inf-rec-job-one",
	Namespace: "/aws/sagemaker/InferenceRecommendationsJobs",
}

var sagemakerInfRecJobResources = []*model.TaggedResource{
	sagemakerInfRecJobOne,
}

func TestAssociatorSagemakerInfRecJob(t *testing.T) {
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
				dimensionRegexps: config.SupportedServices.GetService("/aws/sagemaker/InferenceRecommendationsJobs").ToModelDimensionsRegexp(),
				resources:        sagemakerInfRecJobResources,
				metric: &model.Metric{
					MetricName: "ClientInvocations",
					Namespace:  "/aws/sagemaker/InferenceRecommendationsJobs",
					Dimensions: []model.Dimension{
						{Name: "JobName", Value: "example-inf-rec-job-one"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: sagemakerInfRecJobOne,
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
