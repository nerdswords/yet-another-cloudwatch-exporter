package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var rabbitMQBroker = &model.TaggedResource{
	ARN:       "arn:aws:mq:us-east-2:123456789012:broker:rabbitmq-broker:b-000-111-222-333",
	Namespace: "AWS/AmazonMQ",
}

var activeMQBroker = &model.TaggedResource{
	ARN:       "arn:aws:mq:us-east-2:123456789012:broker:activemq-broker:b-000-111-222-333",
	Namespace: "AWS/AmazonMQ",
}

func TestAssociatorMQ(t *testing.T) {
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
			name: "should match with Broker dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/AmazonMQ").ToModelDimensionsRegexp(),
				resources:        []*model.TaggedResource{rabbitMQBroker},
				metric: &model.Metric{
					MetricName: "ProducerCount",
					Namespace:  "AWS/AmazonMQ",
					Dimensions: []model.Dimension{
						{Name: "Broker", Value: "rabbitmq-broker"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: rabbitMQBroker,
		},
		{
			// ActiveMQ allows active/standby modes where the `Broker` dimension has values
			// like `brokername-1` and `brokername-2` which don't match the ARN (the dimension
			// regex will extract `Broker` as `brokername` from ARN)
			name: "should match with Broker dimension when broker name has a number suffix",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/AmazonMQ").ToModelDimensionsRegexp(),
				resources:        []*model.TaggedResource{activeMQBroker},
				metric: &model.Metric{
					MetricName: "ProducerCount",
					Namespace:  "AWS/AmazonMQ",
					Dimensions: []model.Dimension{
						{Name: "Broker", Value: "activemq-broker-1"},
					},
				},
			},

			expectedSkip:     false,
			expectedResource: activeMQBroker,
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
