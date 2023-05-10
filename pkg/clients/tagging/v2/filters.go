package v2

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/storagegateway"
	"github.com/grafana/regexp"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

type serviceFilter struct {
	// ResourceFunc can be used to fetch additional resources
	ResourceFunc func(context.Context, client, *config.Job, string) ([]*model.TaggedResource, error)

	// FilterFunc can be used to the input resources or to drop based on some condition
	FilterFunc func(context.Context, client, []*model.TaggedResource) ([]*model.TaggedResource, error)
}

// serviceFilters maps a service namespace to (optional) serviceFilter
var serviceFilters = map[string]serviceFilter{
	"AWS/ApiGateway": {
		FilterFunc: func(ctx context.Context, client client, inputResources []*model.TaggedResource) ([]*model.TaggedResource, error) {
			var limit int32 = 500 // max number of results per page. default=25, max=500
			const maxPages = 10
			input := apigateway.GetRestApisInput{Limit: &limit}
			output := apigateway.GetRestApisOutput{}
			var pageNum int

			paginator := apigateway.NewGetRestApisPaginator(client.apiGatewayAPI, &input, func(options *apigateway.GetRestApisPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for paginator.HasMorePages() && pageNum <= maxPages {
				page, err := paginator.NextPage(ctx)
				promutil.APIGatewayAPICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling apiGatewayAPI.GetRestApis, %w", err)
				}
				pageNum++
				output.Items = append(output.Items, page.Items...)
			}

			var outputResources []*model.TaggedResource
			for _, resource := range inputResources {
				for i, gw := range output.Items {
					searchString := regexp.MustCompile(fmt.Sprintf(".*apis/%s$", *gw.Id))
					if searchString.MatchString(resource.ARN) {
						r := resource
						r.ARN = strings.ReplaceAll(resource.ARN, *gw.Id, *gw.Name)
						outputResources = append(outputResources, r)
						output.Items = append(output.Items[:i], output.Items[i+1:]...)
						break
					}
				}
			}

			return outputResources, nil
		},
	},
	"AWS/AutoScaling": {
		ResourceFunc: func(ctx context.Context, client client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			paginator := autoscaling.NewDescribeAutoScalingGroupsPaginator(client.autoscalingAPI, &autoscaling.DescribeAutoScalingGroupsInput{}, func(options *autoscaling.DescribeAutoScalingGroupsPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for paginator.HasMorePages() && pageNum < 100 {
				page, err := paginator.NextPage(ctx)
				promutil.AutoScalingAPICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling autoscalingAPI.DescribeAutoScalingGroups, %w", err)
				}
				pageNum++

				for _, asg := range page.AutoScalingGroups {
					resource := model.TaggedResource{
						ARN:       *asg.AutoScalingGroupARN,
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
			}

			return resources, nil
		},
	},
	"AWS/DMS": {
		// Append the replication instance identifier to DMS task and instance ARNs
		FilterFunc: func(ctx context.Context, client client, inputResources []*model.TaggedResource) ([]*model.TaggedResource, error) {
			if len(inputResources) == 0 {
				return inputResources, nil
			}

			replicationInstanceIdentifiers := make(map[string]string)

			pageNum := 0
			instancesPaginator := databasemigrationservice.NewDescribeReplicationInstancesPaginator(client.dmsAPI, &databasemigrationservice.DescribeReplicationInstancesInput{}, func(options *databasemigrationservice.DescribeReplicationInstancesPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for instancesPaginator.HasMorePages() && pageNum < 100 {
				page, err := instancesPaginator.NextPage(ctx)
				promutil.DmsAPICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling dmsAPI.DescribeReplicationInstances, %w", err)
				}
				pageNum++

				for _, instance := range page.ReplicationInstances {
					replicationInstanceIdentifiers[*instance.ReplicationInstanceArn] = *instance.ReplicationInstanceIdentifier
				}
			}

			pageNum = 0
			tasksPaginator := databasemigrationservice.NewDescribeReplicationTasksPaginator(client.dmsAPI, &databasemigrationservice.DescribeReplicationTasksInput{}, func(options *databasemigrationservice.DescribeReplicationTasksPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for tasksPaginator.HasMorePages() && pageNum < 100 {
				page, err := tasksPaginator.NextPage(ctx)
				promutil.DmsAPICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling dmsAPI.DescribeReplicationTasks, %w", err)
				}
				pageNum++

				for _, task := range page.ReplicationTasks {
					taskInstanceArn := *task.ReplicationInstanceArn
					if instanceIdentifier, ok := replicationInstanceIdentifiers[taskInstanceArn]; ok {
						replicationInstanceIdentifiers[*task.ReplicationTaskArn] = instanceIdentifier
					}
				}
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
		ResourceFunc: func(ctx context.Context, client client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			paginator := ec2.NewDescribeSpotFleetRequestsPaginator(client.ec2API, &ec2.DescribeSpotFleetRequestsInput{}, func(options *ec2.DescribeSpotFleetRequestsPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for paginator.HasMorePages() && pageNum < 100 {
				page, err := paginator.NextPage(ctx)
				promutil.Ec2APICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling describing ec2API.DescribeSpotFleetRequests, %w", err)
				}
				pageNum++

				for _, ec2Spot := range page.SpotFleetRequestConfigs {
					resource := model.TaggedResource{
						ARN:       *ec2Spot.SpotFleetRequestId,
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
			}

			return resources, nil
		},
	},
	"AWS/Prometheus": {
		ResourceFunc: func(ctx context.Context, client client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			paginator := amp.NewListWorkspacesPaginator(client.prometheusSvcAPI, &amp.ListWorkspacesInput{}, func(options *amp.ListWorkspacesPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for paginator.HasMorePages() && pageNum < 100 {
				page, err := paginator.NextPage(ctx)
				promutil.ManagedPrometheusAPICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error while calling prometheusSvcAPI.ListWorkspaces, %w", err)
				}
				pageNum++

				for _, ws := range page.Workspaces {
					resource := model.TaggedResource{
						ARN:       *ws.Arn,
						Namespace: job.Type,
						Region:    region,
					}

					for key, value := range ws.Tags {
						resource.Tags = append(resource.Tags, model.Tag{Key: key, Value: value})
					}

					if resource.FilterThroughTags(job.SearchTags) {
						resources = append(resources, &resource)
					}
				}
			}

			return resources, nil
		},
	},
	"AWS/StorageGateway": {
		ResourceFunc: func(ctx context.Context, client client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			paginator := storagegateway.NewListGatewaysPaginator(client.storageGatewayAPI, &storagegateway.ListGatewaysInput{}, func(options *storagegateway.ListGatewaysPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for paginator.HasMorePages() && pageNum < 100 {
				page, err := paginator.NextPage(ctx)
				promutil.StoragegatewayAPICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling storageGatewayAPI.ListGateways, %w", err)
				}
				pageNum++

				for _, gwa := range page.Gateways {
					resource := model.TaggedResource{
						ARN:       fmt.Sprintf("%s/%s", *gwa.GatewayId, *gwa.GatewayName),
						Namespace: job.Type,
						Region:    region,
					}

					tagsRequest := &storagegateway.ListTagsForResourceInput{
						ResourceARN: gwa.GatewayARN,
					}
					tagsResponse, _ := client.storageGatewayAPI.ListTagsForResource(ctx, tagsRequest)
					promutil.StoragegatewayAPICounter.Inc()

					for _, t := range tagsResponse.Tags {
						resource.Tags = append(resource.Tags, model.Tag{Key: *t.Key, Value: *t.Value})
					}

					if resource.FilterThroughTags(job.SearchTags) {
						resources = append(resources, &resource)
					}
				}
			}

			return resources, nil
		},
	},
	"AWS/TransitGateway": {
		ResourceFunc: func(ctx context.Context, client client, job *config.Job, region string) ([]*model.TaggedResource, error) {
			pageNum := 0
			var resources []*model.TaggedResource
			paginator := ec2.NewDescribeTransitGatewayAttachmentsPaginator(client.ec2API, &ec2.DescribeTransitGatewayAttachmentsInput{}, func(options *ec2.DescribeTransitGatewayAttachmentsPaginatorOptions) {
				options.StopOnDuplicateToken = true
			})
			for paginator.HasMorePages() && pageNum < 100 {
				page, err := paginator.NextPage(ctx)
				promutil.Ec2APICounter.Inc()
				if err != nil {
					return nil, fmt.Errorf("error calling ec2API.DescribeTransitGatewayAttachments, %w", err)
				}
				pageNum++

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
			}

			return resources, nil
		},
	},
}
