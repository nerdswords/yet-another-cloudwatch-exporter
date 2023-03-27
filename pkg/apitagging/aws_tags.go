package apitagging

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

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

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type TagsInterface struct {
	Client               resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	AsgClient            autoscalingiface.AutoScalingAPI
	APIGatewayClient     apigatewayiface.APIGatewayAPI
	Ec2Client            ec2iface.EC2API
	DynamoDBClient       dynamodbiface.DynamoDBAPI
	DmsClient            databasemigrationserviceiface.DatabaseMigrationServiceAPI
	PrometheusClient     prometheusserviceiface.PrometheusServiceAPI
	StoragegatewayClient storagegatewayiface.StorageGatewayAPI
	Logger               logging.Logger
}

func (iface TagsInterface) Get(ctx context.Context, job *config.Job, region string) ([]*model.TaggedResource, error) {
	svc := config.SupportedServices.GetService(job.Type)
	var resources []*model.TaggedResource

	if len(svc.ResourceFilters) > 0 {
		inputparams := &resourcegroupstaggingapi.GetResourcesInput{
			ResourceTypeFilters: svc.ResourceFilters,
			ResourcesPerPage:    aws.Int64(100), // max allowed value according to API docs
		}
		pageNum := 0

		err := iface.Client.GetResourcesPagesWithContext(ctx, inputparams, func(page *resourcegroupstaggingapi.GetResourcesOutput, lastPage bool) bool {
			pageNum++
			promutil.ResourceGroupTaggingAPICounter.Inc()

			if len(page.ResourceTagMappingList) == 0 {
				iface.Logger.Error(errors.New("resource tag list is empty"), "Account contained no tagged resource. Tags must be defined for resources to be discovered.")
			}

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
					iface.Logger.Debug("Skipping resource because search tags do not match", "arn", resource.ARN)
				}
			}
			return !lastPage
		})
		if err != nil {
			return nil, err
		}

		iface.Logger.Debug("GetResourcesPages finished", "total", len(resources))
	}

	if ext, ok := serviceFilters[svc.Namespace]; ok {
		if ext.ResourceFunc != nil {
			newResources, err := ext.ResourceFunc(ctx, iface, job, region)
			if err != nil {
				return nil, err
			}
			resources = append(resources, newResources...)
			iface.Logger.Debug("ResourceFunc finished", "total", len(resources))
		}

		if ext.FilterFunc != nil {
			filteredResources, err := ext.FilterFunc(ctx, iface, resources)
			if err != nil {
				return nil, err
			}
			resources = filteredResources
			iface.Logger.Debug("FilterFunc finished", "total", len(resources))
		}
	}

	return resources, nil
}
