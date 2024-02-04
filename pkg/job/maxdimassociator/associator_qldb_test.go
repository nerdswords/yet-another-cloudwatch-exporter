package maxdimassociator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

var validQldbInstance = &model.TaggedResource{
	ARN:       "arn:aws:qldb:us-east-1:123456789012:ledger/test1",
	Namespace: "AWS/QLDB",
}

func TestAssociatorQLDB(t *testing.T) {
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
			name: "should match with ledger name dimension",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/QLDB").ToModelDimensionsRegexp(),
				resources:        []*model.TaggedResource{validQldbInstance},
				metric: &model.Metric{
					Namespace:  "AWS/QLDB",
					MetricName: "JournalStorage",
					Dimensions: []model.Dimension{
						{Name: "LedgerName", Value: "test2"},
					},
				},
			},
			expectedSkip:     true,
			expectedResource: nil,
		},
		{
			name: "should not match with ledger name dimension when QLDB arn is not valid",
			args: args{
				dimensionRegexps: config.SupportedServices.GetService("AWS/QLDB").ToModelDimensionsRegexp(),
				resources:        []*model.TaggedResource{validQldbInstance},
				metric: &model.Metric{
					Namespace:  "AWS/QLDB",
					MetricName: "JournalStorage",
					Dimensions: []model.Dimension{
						{Name: "LedgerName", Value: "test1"},
					},
				},
			},
			expectedSkip:     false,
			expectedResource: validQldbInstance,
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
