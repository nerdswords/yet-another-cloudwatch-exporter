package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var sagemakerPipelineOne = &model.TaggedResource{
	ARN:       "arn:aws:sagemaker:us-west-2:123456789012:pipeline/example-pipeline-one",
	Namespace: "AWS/Sagemaker/ModelBuildingPipeline",
}

var sagemakerPipelineTwo = &model.TaggedResource{
	ARN:       "arn:aws:sagemaker:us-west-2:123456789012:pipeline/example-pipeline-two",
	Namespace: "AWS/Sagemaker/ModelBuildingPipeline",
}

var sagemakerPipelineResources = []*model.TaggedResource{
	sagemakerPipelineOne,
	sagemakerPipelineTwo,
}

func TestAssociatorSagemakerPipeline(t *testing.T) {
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
			name: "2 dimensions should match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Sagemaker/ModelBuildingPipeline").ToModelDimensionsRegexp(),
				resources:        sagemakerPipelineResources,
				metric: &model.Metric{
					MetricName: "ExecutionStarted",
					Namespace:  "AWS/Sagemaker/ModelBuildingPipeline",
					Dimensions: []model.Dimension{
						{Name: "PipelineName", Value: "example-pipeline-one"},
						{Name: "StepName", Value: "example-pipeline-one-step-two"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: sagemakerPipelineOne,
		},
		{
			name: "1 dimension should match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Sagemaker/ModelBuildingPipeline").ToModelDimensionsRegexp(),
				resources:        sagemakerPipelineResources,
				metric: &model.Metric{
					MetricName: "ExecutionStarted",
					Namespace:  "AWS/Sagemaker/ModelBuildingPipeline",
					Dimensions: []model.Dimension{
						{Name: "PipelineName", Value: "example-pipeline-two"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: sagemakerPipelineTwo,
		},
		{
			name: "2 dimensions should not match",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/Sagemaker/ModelBuildingPipeline").ToModelDimensionsRegexp(),
				resources:        sagemakerPipelineResources,
				metric: &model.Metric{
					MetricName: "ExecutionStarted",
					Namespace:  "AWS/Sagemaker/ModelBuildingPipeline",
					Dimensions: []model.Dimension{
						{Name: "PipelineName", Value: "example-pipeline-three"},
						{Name: "StepName", Value: "example-pipeline-three-step-two"},
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
