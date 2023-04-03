package apitagging

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice/databasemigrationserviceiface"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/prometheusservice/prometheusserviceiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/storagegateway/storagegatewayiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
)

var ErrExpectedToFindResources = errors.New("expected to discover resources but none were found")

type Client struct {
	logger            logging.Logger
	taggingAPI        resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	autoscalingAPI    autoscalingiface.AutoScalingAPI
	apiGatewayAPI     apigatewayiface.APIGatewayAPI
	ec2API            ec2iface.EC2API
	dmsAPI            databasemigrationserviceiface.DatabaseMigrationServiceAPI
	prometheusSvcAPI  prometheusserviceiface.PrometheusServiceAPI
	storageGatewayAPI storagegatewayiface.StorageGatewayAPI
}

func NewClient(
	logger logging.Logger,
	taggingAPI resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI,
	autoscalingAPI autoscalingiface.AutoScalingAPI,
	apiGatewayAPI apigatewayiface.APIGatewayAPI,
	ec2API ec2iface.EC2API,
	dmsClient databasemigrationserviceiface.DatabaseMigrationServiceAPI,
	prometheusClient prometheusserviceiface.PrometheusServiceAPI,
	storageGatewayAPI storagegatewayiface.StorageGatewayAPI,
) *Client {
	return &Client{
		logger:            logger,
		taggingAPI:        taggingAPI,
		autoscalingAPI:    autoscalingAPI,
		apiGatewayAPI:     apiGatewayAPI,
		ec2API:            ec2API,
		dmsAPI:            dmsClient,
		prometheusSvcAPI:  prometheusClient,
		storageGatewayAPI: storageGatewayAPI,
	}
}

func (c Client) GetResources(ctx context.Context, job *config.Job, region string) ([]*model.TaggedResource, error) {
	svc := config.SupportedServices.GetService(job.Type)
	var resources []*model.TaggedResource
	shouldHaveDiscoveredResources := false

	if len(svc.ResourceFilters) > 0 {
		shouldHaveDiscoveredResources = true
		inputparams := &resourcegroupstaggingapi.GetResourcesInput{
			ResourceTypeFilters: svc.ResourceFilters,
			ResourcesPerPage:    aws.Int64(100), // max allowed value according to API docs
		}
		pageNum := 0

		err := c.taggingAPI.GetResourcesPagesWithContext(ctx, inputparams, func(page *resourcegroupstaggingapi.GetResourcesOutput, lastPage bool) bool {
			pageNum++
			promutil.ResourceGroupTaggingAPICounter.Inc()

			for _, resourceTagMapping := range page.ResourceTagMappingList {
				resource := model.TaggedResource{
					ARN:       aws.StringValue(resourceTagMapping.ResourceARN),
					Namespace: job.Type,
					Region:    region,
				}

				for _, t := range resourceTagMapping.Tags {
					resource.Tags = append(resource.Tags, model.Tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.FilterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				} else {
					c.logger.Debug("Skipping resource because search tags do not match", "arn", resource.ARN)
				}
			}
			return !lastPage
		})
		if err != nil {
			return nil, err
		}

		c.logger.Debug("GetResourcesPages finished", "total", len(resources))
	}

	if ext, ok := serviceFilters[svc.Namespace]; ok {
		if ext.ResourceFunc != nil {
			shouldHaveDiscoveredResources = true
			newResources, err := ext.ResourceFunc(ctx, c, job, region)
			if err != nil {
				return nil, fmt.Errorf("failed to apply ResourceFunc for %s, %w", svc.Namespace, err)
			}
			resources = append(resources, newResources...)
			c.logger.Debug("ResourceFunc finished", "total", len(resources))
		}

		if ext.FilterFunc != nil {
			filteredResources, err := ext.FilterFunc(ctx, c, resources)
			if err != nil {
				return nil, fmt.Errorf("failed to apply FilterFunc for %s, %w", svc.Namespace, err)
			}
			resources = filteredResources
			c.logger.Debug("FilterFunc finished", "total", len(resources))
		}
	}

	if shouldHaveDiscoveredResources && len(resources) == 0 {
		return nil, ErrExpectedToFindResources
	}

	return resources, nil
}
