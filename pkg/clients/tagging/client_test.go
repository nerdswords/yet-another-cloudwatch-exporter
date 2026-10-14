// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tagging

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/prometheus/common/promslog"
	"github.com/stretchr/testify/require"

	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/promutil"
)

func TestGetResources(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		mappings          []types.ResourceTagMapping
		expectedResources []*model.TaggedResource
		expectedErr       bool
	}{
		{
			name:        "unsupported namespace",
			namespace:   "AWS/DoesNotExist",
			expectedErr: true,
		},
		{
			name:      "resources with tags",
			namespace: "AWS/SQS",
			mappings: []types.ResourceTagMapping{
				{
					ResourceARN: aws.String("arn:aws:sqs:us-east-1:123456789012:queue-1"),
					Tags:        []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
				},
			},
			expectedResources: []*model.TaggedResource{
				{
					ARN:       "arn:aws:sqs:us-east-1:123456789012:queue-1",
					Namespace: "AWS/SQS",
					Region:    "us-east-1",
					Tags:      []model.Tag{{Key: "env", Value: "prod"}},
				},
			},
		},
		{
			name:      "mapping without ARN is skipped",
			namespace: "AWS/SQS",
			mappings: []types.ResourceTagMapping{
				{
					ResourceARN: nil,
					Tags:        []types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}},
				},
				{
					ResourceARN: aws.String("arn:aws:sqs:us-east-1:123456789012:queue-1"),
				},
			},
			expectedResources: []*model.TaggedResource{
				{
					ARN:       "arn:aws:sqs:us-east-1:123456789012:queue-1",
					Namespace: "AWS/SQS",
					Region:    "us-east-1",
					Tags:      []model.Tag{},
				},
			},
		},
		{
			name:      "tag with nil value",
			namespace: "AWS/SQS",
			mappings: []types.ResourceTagMapping{
				{
					ResourceARN: aws.String("arn:aws:sqs:us-east-1:123456789012:queue-1"),
					Tags:        []types.Tag{{Key: aws.String("env"), Value: nil}},
				},
			},
			expectedResources: []*model.TaggedResource{
				{
					ARN:       "arn:aws:sqs:us-east-1:123456789012:queue-1",
					Namespace: "AWS/SQS",
					Region:    "us-east-1",
					Tags:      []model.Tag{{Key: "env", Value: ""}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := client{
				logger:        promslog.NewNopLogger(),
				scrapeMetrics: promutil.Discard,
				taggingAPI: taggingClientAdapter{
					getResources: func(_ context.Context, _ *resourcegroupstaggingapi.GetResourcesInput, _ ...func(*resourcegroupstaggingapi.Options)) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
						return &resourcegroupstaggingapi.GetResourcesOutput{ResourceTagMappingList: tc.mappings}, nil
					},
				},
			}

			resources, err := c.GetResources(context.Background(), model.DiscoveryJob{Namespace: tc.namespace}, "us-east-1")
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedResources, resources)
		})
	}
}
