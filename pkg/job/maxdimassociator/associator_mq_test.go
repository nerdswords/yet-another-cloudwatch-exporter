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

var mqBroker = &model.TaggedResource{
	ARN:       "arn:aws:mq:af-south-1:123456789222:broker:sampleBroker:b-deadbeef",
	Namespace: "AWS/ActiveMQ",
	Region:    "af-south-1",
}

func TestAssociatorAMQ(t *testing.T) {
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
			name: "activemq broker metrics have a dimension with a dash-number suffix",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/AmazonMQ").DimensionRegexps,
				resources: []*model.TaggedResource{
					mqBroker,
				},
				metric: &cloudwatch.Metric{
					MetricName: aws.String("CPUUtilization"),
					Namespace:  aws.String("AWS/ActiveMQ"),
					Dimensions: []*cloudwatch.Dimension{
						{Name: aws.String("Broker"), Value: aws.String("sampleBroker-1")},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: mqBroker,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			associator := NewAssociator(tc.args.dimensionRegexps, tc.args.resources)
			res, skip := associator.AssociateMetricToResource(tc.args.metric)
			require.Equal(t, tc.expectedSkip, skip)
			require.Equal(t, tc.expectedResource, res)
		})
	}
}
