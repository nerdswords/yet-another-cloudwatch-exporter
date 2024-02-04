package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var mediaConvertQueue = &model.TaggedResource{
	ARN:       "arn:aws:mediaconvert:eu-west-1:631611414237:queues/a-queue",
	Namespace: "AWS/MediaConvert",
}

var mediaConvertQueueTwo = &model.TaggedResource{
	ARN:       "arn:aws:mediaconvert:eu-west-1:631611414237:queues/a-second-queue",
	Namespace: "AWS/MediaConvert",
}

var mediaConvertResources = []*model.TaggedResource{
	mediaConvertQueue,
	mediaConvertQueueTwo,
}

func TestAssociatorMediaConvert(t *testing.T) {
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
			name: "should match with mediaconvert queue one dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/MediaConvert").ToModelDimensionsRegexp(),
				resources:        mediaConvertResources,
				metric: &model.Metric{
					MetricName: "JobsCompletedCount",
					Namespace:  "AWS/MediaConvert",
					Dimensions: []model.Dimension{
						{Name: "Queue", Value: "arn:aws:mediaconvert:eu-west-1:631611414237:queues/a-queue"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: mediaConvertQueue,
		},
		{
			name: "should match with mediaconvert queue two dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/MediaConvert").ToModelDimensionsRegexp(),
				resources:        mediaConvertResources,
				metric: &model.Metric{
					MetricName: "JobsCompletedCount",
					Namespace:  "AWS/MediaConvert",
					Dimensions: []model.Dimension{
						{Name: "Queue", Value: "arn:aws:mediaconvert:eu-west-1:631611414237:queues/a-second-queue"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: mediaConvertQueueTwo,
		},
		{
			name: "should not match with any mediaconvert queue",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/MediaConvert").ToModelDimensionsRegexp(),
				resources:        mediaConvertResources,
				metric: &model.Metric{
					MetricName: "JobsCompletedCount",
					Namespace:  "AWS/MediaConvert",
					Dimensions: []model.Dimension{
						{Name: "Queue", Value: "arn:aws:mediaconvert:eu-west-1:631611414237:queues/a-non-existing-queue"},
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
