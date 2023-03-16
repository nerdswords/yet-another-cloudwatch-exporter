package apitagging

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/prometheusservice"
	"github.com/aws/aws-sdk-go/service/storagegateway"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

type serviceFilter struct {
	// ResourceFunc can be used to fetch additional resources
	ResourceFunc func(context.Context, Client, *config.Job, string) ([]*model.TaggedResource, error)

	// FilterFunc can be used to the input resources or to drop based on some condition
	FilterFunc func(context.Context, Client, []*model.TaggedResource) ([]*model.TaggedResource, error)
}

// serviceFilters maps a service namespace to (optional) serviceFilter
var serviceFilters = map[string]serviceFilter{
	"AWS/ApiGateway": {
		FilterFunc: func(ctx context.Context, client Client, inputResources []*model.TaggedResource) ([]*model.TaggedResource, error) {
			const maxPages = 10

			var (
				limit           int64 = 500 // max number of results per page. default=25, max=500
				input                 = apigateway.GetRestApisInput{Limit: &limit}
				output                = apigateway.GetRestApisOutput{}
				pageNum         int
				outputResources []*model.TaggedResource
			)

			err := client.apiGatewayAPI.GetRestApisPagesWithContext(ctx, &input, func(page *apigateway.GetRestApisOutput, lastPage bool) bool {
				promutil.APIGatewayAPICounter.Inc()
				pageNum++
				output.Items = append(output.Items, page.Items...)
				return pageNum <= maxPages
			})
			if err != nil {
				return nil, fmt.Errorf("error calling apiGatewayAPI.GetRestApisPages, %w", err)
			}
			outputV2, err := client.apiGatewayV2API.GetApisWithContext(ctx, &apigatewayv2.GetApisInput{})
			promutil.APIGatewayAPIV2Counter.Inc()
			if err != nil {
				return nil, fmt.Errorf("error calling apiGatewayAPIv2.GetApis, %w", err)
			}

			for _, resource := range inputResources {
				for i, gw := range output.Items {
					searchString := regexp.MustCompile(fmt.Sprintf(".*restapis/%s$", *gw.Id))
					if searchString.MatchString(resource.ARN) {
						r := resource
						r.ARN = strings.ReplaceAll(resource.ARN, *gw.Id, *gw.Name)
						outputResources = append(outputResources, r)
						output.Items = append(output.Items[:i], output.Items[i+1:]...)
						break
					}
				}
				for i, gw := range outputV2.Items {
					searchString := regexp.MustCompile(fmt.Sprintf(".*apis/%s$", *gw.ApiId))
					if searchString.MatchString(resource.ARN) {
						outputResources = append(outputResources, resource)
						outputV2.Items = append(outputV2.Items[:i], outputV2.Items[i+1:]...)
						break
					}
				}
			}
			return outputResources, nil
		},
	},
	"AWS/AutoScaling": {
		ResourceFunc: func(ctx context.Context, client Client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			err := client.autoscalingAPI.DescribeAutoScalingGroupsPagesWithContext(ctx, &autoscaling.DescribeAutoScalingGroupsInput{},
				func(page *autoscaling.DescribeAutoScalingGroupsOutput, more bool) bool {
					pageNum++
					promutil.AutoScalingAPICounter.Inc()

					for _, asg := range page.AutoScalingGroups {
						resource := model.TaggedResource{
							ARN:       aws.StringValue(asg.AutoScalingGroupARN),
							Namespace: job.Type,
							Region:    region,
						}

						for _, t := range asg.Tags {
							resource.Tags = append(resource.Tags, model.Tag{Key: *t.Key, Value: *t.Value})
						}

						if resource.FilterThroughTags(job.SearchTags) {
							resources = append(resources, &resource)
						}
					}
					return pageNum < 100
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error calling autoscalingAPI.DescribeAutoScalingGroups, %w", err)
			}
			return resources, nil
		},
	},
	"AWS/DMS": {
		// Append the replication instance identifier to DMS task and instance ARNs
		FilterFunc: func(ctx context.Context, client Client, inputResources []*model.TaggedResource) ([]*model.TaggedResource, error) {
			if len(inputResources) == 0 {
				return inputResources, nil
			}

			replicationInstanceIdentifiers := make(map[string]string)
			pageNum := 0
			if err := client.dmsAPI.DescribeReplicationInstancesPagesWithContext(ctx, nil,
				func(page *databasemigrationservice.DescribeReplicationInstancesOutput, lastPage bool) bool {
					pageNum++
					promutil.DmsAPICounter.Inc()

					for _, instance := range page.ReplicationInstances {
						replicationInstanceIdentifiers[aws.StringValue(instance.ReplicationInstanceArn)] = aws.StringValue(instance.ReplicationInstanceIdentifier)
					}

					return pageNum < 100
				},
			); err != nil {
				return nil, fmt.Errorf("error calling dmsAPI.DescribeReplicationInstances, %w", err)
			}
			pageNum = 0
			if err := client.dmsAPI.DescribeReplicationTasksPagesWithContext(ctx, nil,
				func(page *databasemigrationservice.DescribeReplicationTasksOutput, lastPage bool) bool {
					pageNum++
					promutil.DmsAPICounter.Inc()

					for _, task := range page.ReplicationTasks {
						taskInstanceArn := aws.StringValue(task.ReplicationInstanceArn)
						if instanceIdentifier, ok := replicationInstanceIdentifiers[taskInstanceArn]; ok {
							replicationInstanceIdentifiers[aws.StringValue(task.ReplicationTaskArn)] = instanceIdentifier
						}
					}

					return pageNum < 100
				},
			); err != nil {
				return nil, fmt.Errorf("error calling dmsAPI.DescribeReplicationTasks, %w", err)
			}

			var outputResources []*model.TaggedResource
			for _, resource := range inputResources {
				r := resource
				// Append the replication instance identifier to replication instance and task ARNs
				if instanceIdentifier, ok := replicationInstanceIdentifiers[r.ARN]; ok {
					r.ARN = fmt.Sprintf("%s/%s", r.ARN, instanceIdentifier)
				}
				outputResources = append(outputResources, r)
			}
			return outputResources, nil
		},
	},
	"AWS/EC2Spot": {
		ResourceFunc: func(ctx context.Context, client Client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			err := client.ec2API.DescribeSpotFleetRequestsPagesWithContext(ctx, &ec2.DescribeSpotFleetRequestsInput{},
				func(page *ec2.DescribeSpotFleetRequestsOutput, more bool) bool {
					pageNum++
					promutil.Ec2APICounter.Inc()

					for _, ec2Spot := range page.SpotFleetRequestConfigs {
						resource := model.TaggedResource{
							ARN:       aws.StringValue(ec2Spot.SpotFleetRequestId),
							Namespace: job.Type,
							Region:    region,
						}

						for _, t := range ec2Spot.Tags {
							resource.Tags = append(resource.Tags, model.Tag{Key: *t.Key, Value: *t.Value})
						}

						if resource.FilterThroughTags(job.SearchTags) {
							resources = append(resources, &resource)
						}
					}
					return pageNum < 100
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error calling describing ec2API.DescribeSpotFleetRequests, %w", err)
			}
			return resources, nil
		},
	},
	"AWS/Prometheus": {
		ResourceFunc: func(ctx context.Context, client Client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			err := client.prometheusSvcAPI.ListWorkspacesPagesWithContext(ctx, &prometheusservice.ListWorkspacesInput{},
				func(page *prometheusservice.ListWorkspacesOutput, more bool) bool {
					pageNum++
					promutil.ManagedPrometheusAPICounter.Inc()

					for _, ws := range page.Workspaces {
						resource := model.TaggedResource{
							ARN:       aws.StringValue(ws.Arn),
							Namespace: job.Type,
							Region:    region,
						}

						for key, value := range ws.Tags {
							resource.Tags = append(resource.Tags, model.Tag{Key: key, Value: *value})
						}

						if resource.FilterThroughTags(job.SearchTags) {
							resources = append(resources, &resource)
						}
					}
					return pageNum < 100
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error while calling prometheusSvcAPI.ListWorkspaces, %w", err)
			}
			return resources, nil
		},
	},
	"AWS/StorageGateway": {
		ResourceFunc: func(ctx context.Context, client Client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			err := client.storageGatewayAPI.ListGatewaysPagesWithContext(ctx, &storagegateway.ListGatewaysInput{},
				func(page *storagegateway.ListGatewaysOutput, more bool) bool {
					pageNum++
					promutil.StoragegatewayAPICounter.Inc()

					for _, gwa := range page.Gateways {
						resource := model.TaggedResource{
							ARN:       fmt.Sprintf("%s/%s", *gwa.GatewayId, *gwa.GatewayName),
							Namespace: job.Type,
							Region:    region,
						}

						tagsRequest := &storagegateway.ListTagsForResourceInput{
							ResourceARN: gwa.GatewayARN,
						}
						tagsResponse, _ := client.storageGatewayAPI.ListTagsForResource(tagsRequest)
						promutil.StoragegatewayAPICounter.Inc()

						for _, t := range tagsResponse.Tags {
							resource.Tags = append(resource.Tags, model.Tag{Key: *t.Key, Value: *t.Value})
						}

						if resource.FilterThroughTags(job.SearchTags) {
							resources = append(resources, &resource)
						}
					}

					return pageNum < 100
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error calling storageGatewayAPI.ListGateways, %w", err)
			}
			return resources, nil
		},
	},
	"AWS/TransitGateway": {
		ResourceFunc: func(ctx context.Context, client Client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			err := client.ec2API.DescribeTransitGatewayAttachmentsPagesWithContext(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{},
				func(page *ec2.DescribeTransitGatewayAttachmentsOutput, more bool) bool {
					pageNum++
					promutil.Ec2APICounter.Inc()

					for _, tgwa := range page.TransitGatewayAttachments {
						resource := model.TaggedResource{
							ARN:       fmt.Sprintf("%s/%s", *tgwa.TransitGatewayId, *tgwa.TransitGatewayAttachmentId),
							Namespace: job.Type,
							Region:    region,
						}

						for _, t := range tgwa.Tags {
							resource.Tags = append(resource.Tags, model.Tag{Key: *t.Key, Value: *t.Value})
						}

						if resource.FilterThroughTags(job.SearchTags) {
							resources = append(resources, &resource)
						}
					}
					return pageNum < 100
				},
			)
			if err != nil {
				return nil, fmt.Errorf("error calling ec2API.DescribeTransitGatewayAttachments, %w", err)
			}
			return resources, nil
		},
	},
}
