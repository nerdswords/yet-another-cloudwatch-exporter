package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice/databasemigrationserviceiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestValidServiceNames(t *testing.T) {
	for svc, filter := range serviceFilters {
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

func TestApiGatewayFilterFunc(t *testing.T) {
	tests := []struct {
		name            string
		iface           TagsInterface
		inputResources  []*TaggedResource
		outputResources []*TaggedResource
	}{
		{
			"api gateway resources skip stages",
			TagsInterface{
				APIGatewayClient: apiGatewayClient{
					getRestApisOutput: &apigateway.GetRestApisOutput{
						Items: []*apigateway.RestApi{
							{
								ApiKeySource:              nil,
								BinaryMediaTypes:          nil,
								CreatedDate:               nil,
								Description:               nil,
								DisableExecuteApiEndpoint: nil,
								EndpointConfiguration:     nil,
								Id:                        aws.String("gwid1234"),
								MinimumCompressionSize:    nil,
								Name:                      aws.String("apiname"),
								Policy:                    nil,
								Tags:                      nil,
								Version:                   nil,
								Warnings:                  nil,
							},
						},
						Position: nil,
					},
				},
			},
			[]*TaggedResource{
				{
					ARN:       "arn:aws:apigateway:us-east-1::/restapis/gwid1234/stages/main",
					Namespace: "apigateway",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value",
						},
					},
				},
				{
					ARN:       "arn:aws:apigateway:us-east-1::/restapis/gwid1234",
					Namespace: "apigateway",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 2",
						},
					},
				},
			},
			[]*TaggedResource{
				{
					ARN:       "arn:aws:apigateway:us-east-1::/restapis/apiname",
					Namespace: "apigateway",
					Region:    "us-east-1",
					Tags: []model.Tag{
						{
							Key:   "Test",
							Value: "Value 2",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			apigateway := serviceFilters["AWS/ApiGateway"]

			outputResources, err := apigateway.FilterFunc(context.Background(), test.iface, test.inputResources)
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

func TestDMSFilterFunc(t *testing.T) {
	tests := []struct {
		name            string
		iface           TagsInterface
		inputResources  []*TaggedResource
		outputResources []*TaggedResource
	}{
		{
			"empty input resources",
			TagsInterface{},
			[]*TaggedResource{},
			[]*TaggedResource{},
		},
		{
			"replication tasks and instances",
			TagsInterface{
				DmsClient: dmsClient{
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
			[]*TaggedResource{
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
			[]*TaggedResource{
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
			dms := serviceFilters["AWS/DMS"]

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

func (dms dmsClient) DescribeReplicationInstancesPagesWithContext(_ aws.Context, input *databasemigrationservice.DescribeReplicationInstancesInput, fn func(*databasemigrationservice.DescribeReplicationInstancesOutput, bool) bool, opts ...request.Option) error {
	fn(dms.describeReplicationInstancesOutput, true)
	return nil
}

func (dms dmsClient) DescribeReplicationTasksPagesWithContext(_ aws.Context, input *databasemigrationservice.DescribeReplicationTasksInput, fn func(*databasemigrationservice.DescribeReplicationTasksOutput, bool) bool, opts ...request.Option) error {
	fn(dms.describeReplicationTasksOutput, true)
	return nil
}

type apiGatewayClient struct {
	apigatewayiface.APIGatewayAPI
	getRestApisOutput *apigateway.GetRestApisOutput
}

func (apigateway apiGatewayClient) GetRestApisPagesWithContext(_ aws.Context, input *apigateway.GetRestApisInput, fn func(*apigateway.GetRestApisOutput, bool) bool, opts ...request.Option) error {
	fn(apigateway.getRestApisOutput, true)
	return nil
}
