package exporter

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

type tagsData struct {
	ID        *string
	Tags      map[string]string
	Namespace *string
	Region    *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	account          string
	client           resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asgClient        autoscalingiface.AutoScalingAPI
	apiGatewayClient apigatewayiface.APIGatewayAPI
	ec2Client        ec2iface.EC2API
}

func (iface tagsInterface) get(job *Job, region string) (resources []*tagsData, err error) {
	svc := SupportedServices.GetService(job.Type)
	if len(svc.ResourceFilters) > 0 {
		var inputparams = r.GetResourcesInput{
			ResourceTypeFilters: svc.ResourceFilters,
		}
		c := iface.client
		ctx := context.Background()
		pageNum := 0

		err = c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
			pageNum++
			resourceGroupTaggingAPICounter.Inc()

			if len(page.ResourceTagMappingList) == 0 {
				log.Errorf("Resource tag list is empty (in %s). Tags must be defined for %s to be discovered.", iface.account, job.Type)
			}

			for _, resourceTagMapping := range page.ResourceTagMappingList {
				resource := tagsData{
					ID:        resourceTagMapping.ResourceARN,
					Namespace: &job.Type,
					Region:    &region,
				}

				for _, t := range resourceTagMapping.Tags {
					resource.Tags[*t.Key] = *t.Value
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				} else {
					log.Debugf("Skipping resource %s because search tags do not match", *resource.ID)
				}
			}
			return pageNum < 100
		})
	}
	if svc.ResourceFunc != nil {
		newResources, err := svc.ResourceFunc(iface, job, region)
		if err != nil {
			return nil, err
		}
		resources = append(resources, newResources...)
	}
	if svc.FilterFunc != nil {
		resources, err = svc.FilterFunc(iface, resources)
		if err != nil {
			return nil, err
		}
	}
	return resources, err
}

func migrateTagsToPrometheus(tagData []*tagsData, labelsSnakeCase bool) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

	tagList := make(map[string][]string)

	for _, d := range tagData {
		for _, entry := range d.Tags {
			if !stringInSlice(entry, tagList[*d.Namespace]) {
				tagList[*d.Namespace] = append(tagList[*d.Namespace], entry)
			}
		}
	}

	for _, d := range tagData {
		promNs := strings.ToLower(*d.Namespace)
		if !strings.HasPrefix(promNs, "aws") {
			promNs = "aws_" + promNs
		}
		name := promString(promNs) + "_info"
		promLabels := make(map[string]string)
		promLabels["name"] = *d.ID

		for _, entry := range tagList[*d.Namespace] {
			labelKey := "tag_" + promStringTag(entry, labelsSnakeCase)
			promLabels[labelKey] = d.Tags[entry]
		}

		var i int
		f := float64(i)

		p := PrometheusMetric{
			name:   &name,
			labels: promLabels,
			value:  &f,
		}

		output = append(output, &p)
	}

	return output
}
