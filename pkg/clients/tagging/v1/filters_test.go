package v1

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice/databasemigrationserviceiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestValidServiceNames(t *testing.T) {
	for svc, filter := range ServiceFilters {
		if config.SupportedServices.GetService(svc) == nil {
			t.Errorf("invalid service name '%s'", svc)
			t.Fail()
		}

		if filter.FilterFunc == nil && filter.ResourceFunc == nil {
			t.Errorf("no filter functions defined for service name '%s'", svc)
			t.FailNow()
		}
	}
}

func TestDMSFilterFunc(t *testing.T) {
	tests := []struct {
		name            string
		iface           client
		inputResources  []*model.TaggedResource
		outputResources []*model.TaggedResource
	}{
		{
			"empty input resources",
			client{},
			[]*model.TaggedResource{},
			[]*model.TaggedResource{},
		},
		{
			"replication tasks and instances",
			client{
				dmsAPI: dmsClient{
					describeReplicationInstancesOutput: &databasemigrationservice.DescribeReplicationInstancesOutput{
						ReplicationInstances: []*databasemigrationservice.ReplicationInstance{
							{
								ReplicationInstanceArn:        aws.String("arn:aws:dms:us-east-1:123123123123:rep:ABCDEFG1234567890"),
								ReplicationInstanceIdentifier: aws.String("repl-instance-identifier-1"),
							},
							{
								ReplicationInstanceArn:        aws.String("arn:aws:dms:us-east-1:123123123123:rep:ZZZZZZZZZZZZZZZZZ"),
								ReplicationInstanceIdentifier: aws.String("repl-instance-identifier-2"),
							},
							{
								ReplicationInstanceArn:        aws.String("arn:aws:dms:us-east-1:123123123123:rep:YYYYYYYYYYYYYYYYY"),
								ReplicationInstanceIdentifier: aws.String("repl-instance-identifier-3"),
							},
						},
					},
					describeReplicationTasksOutput: &databasemigrationservice.DescribeReplicationTasksOutput{
						ReplicationTasks: []*databasemigrationservice.ReplicationTask{
							{
								ReplicationTaskArn:     aws.String("arn:aws:dms:us-east-1:123123123123:task:9999999999999999"),
								ReplicationInstanceArn: aws.String("arn:aws:dms:us-east-1:123123123123:rep:ZZZZZZZZZZZZZZZZZ"),
							},
							{
								ReplicationTaskArn:     aws.String("arn:aws:dms:us-east-1:123123123123:task:2222222222222222"),
								ReplicationInstanceArn: aws.String("arn:aws:dms:us-east-1:123123123123:rep:ZZZZZZZZZZZZZZZZZ"),
							},
							{
								ReplicationTaskArn:     aws.String("arn:aws:dms:us-east-1:123123123123:task:3333333333333333"),
								ReplicationInstanceArn: aws.String("arn:aws:dms:us-east-1:123123123123:rep:WWWWWWWWWWWWWWWWW"),
							},
						},
					},
				},
			},
			[]*model.TaggedResource{
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:rep:ABCDEFG1234567890",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:rep:WXYZ987654321",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 2",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:task:9999999999999999",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 3",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:task:5555555555555555",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 4",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:subgrp:demo-subgrp",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 5",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:endpoint:1111111111111111",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 6",
						},
					},
				},
			},
			[]*model.TaggedResource{
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:rep:ABCDEFG1234567890/repl-instance-identifier-1",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:rep:WXYZ987654321",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 2",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:task:9999999999999999/repl-instance-identifier-2",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 3",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:task:5555555555555555",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 4",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:subgrp:demo-subgrp",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 5",
						},
					},
				},
				{
					ARN:       "arn:aws:dms:us-east-1:123123123123:endpoint:1111111111111111",
					Namespace: "dms",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 6",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dms := ServiceFilters["AWS/DMS"]

			outputResources, err := dms.FilterFunc(context.Background(), test.iface, test.inputResources)
			if err != nil {
				t.Logf("Error from FilterFunc: %v", err)
				t.FailNow()
			}
			if len(outputResources) != len(test.outputResources) {
				t.Logf("len(outputResources) = %d, want %d", len(outputResources), len(test.outputResources))
				t.Fail()
			}
			for i, resource := range outputResources {
				if len(test.outputResources) <= i {
					break
				}
				wantResource := *test.outputResources[i]
				if !reflect.DeepEqual(*resource, wantResource) {
					t.Errorf("outputResources[%d] = %+v, want %+v", i, *resource, wantResource)
				}
			}
		})
	}
}

type dmsClient struct {
	databasemigrationserviceiface.DatabaseMigrationServiceAPI
	describeReplicationInstancesOutput *databasemigrationservice.DescribeReplicationInstancesOutput
	describeReplicationTasksOutput     *databasemigrationservice.DescribeReplicationTasksOutput
}

func (dms dmsClient) DescribeReplicationInstancesPagesWithContext(_ aws.Context, _ *databasemigrationservice.DescribeReplicationInstancesInput, fn func(*databasemigrationservice.DescribeReplicationInstancesOutput, bool) bool, _ ...request.Option) error {
	fn(dms.describeReplicationInstancesOutput, true)
	return nil
}

func (dms dmsClient) DescribeReplicationTasksPagesWithContext(_ aws.Context, _ *databasemigrationservice.DescribeReplicationTasksInput, fn func(*databasemigrationservice.DescribeReplicationTasksOutput, bool) bool, _ ...request.Option) error {
	fn(dms.describeReplicationTasksOutput, true)
	return nil
}
