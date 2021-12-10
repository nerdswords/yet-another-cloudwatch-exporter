package exporter

import (
	"context"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/databasemigrationservice/databasemigrationserviceiface"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

// taggedResource is an AWS resource with tags
type taggedResource struct {
	// ARN is the unique AWS ARN (Amazon Resource Name) of the resource
	ARN string

	// Namespace identifies the resource type (e.g. EC2)
	Namespace string

	// Region is the AWS regions that the resource belongs to
	Region string

	// Tags is a set of tags associated to the resource
	Tags []Tag
}

// filterThroughTags returns true if all filterTags match
// with tags of the taggedResource, returns false otherwise.
func (r taggedResource) filterThroughTags(filterTags []Tag) bool {
	tagMatches := 0

	for _, resourceTag := range r.Tags {
		for _, filterTag := range filterTags {
			if resourceTag.Key == filterTag.Key {
				r, _ := regexp.Compile(filterTag.Value)
				if r.MatchString(resourceTag.Value) {
					tagMatches++
				}
			}
		}
	}

	return tagMatches == len(filterTags)
}

// metricTags returns a list of tags built from the tags of
// taggedResource, if there's a definition for its namespace
// in tagsOnMetrics.
//
// Returned tags have as key the key from tagsOnMetrics, and
// as value the value from the corresponding tag of the resource,
// if it exists (otherwise an empty string).
func (r taggedResource) metricTags(tagsOnMetrics exportedTagsOnMetrics) []Tag {
	tags := make([]Tag, 0)
	for _, tagName := range tagsOnMetrics[r.Namespace] {
		tag := Tag{
			Key: tagName,
		}
		for _, resourceTag := range r.Tags {
			if resourceTag.Key == tagName {
				tag.Value = resourceTag.Value
				break
			}
		}

		// Always add the tag, even if it's empty, to ensure the same labels are present on all metrics for a single service
		tags = append(tags, tag)
	}
	return tags
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	account          string
	client           resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asgClient        autoscalingiface.AutoScalingAPI
	apiGatewayClient apigatewayiface.APIGatewayAPI
	ec2Client        ec2iface.EC2API
	dmsClient        databasemigrationserviceiface.DatabaseMigrationServiceAPI
}

func (iface tagsInterface) get(job *Job, region string) ([]*taggedResource, error) {
	svc := SupportedServices.GetService(job.Type)
	var resources []*taggedResource

	if len(svc.ResourceFilters) > 0 {
		var inputparams = &resourcegroupstaggingapi.GetResourcesInput{
			ResourceTypeFilters: svc.ResourceFilters,
		}
		c := iface.client
		ctx := context.Background()
		pageNum := 0

		err := c.GetResourcesPagesWithContext(ctx, inputparams, func(page *resourcegroupstaggingapi.GetResourcesOutput, lastPage bool) bool {
			pageNum++
			resourceGroupTaggingAPICounter.Inc()

			if len(page.ResourceTagMappingList) == 0 {
				log.Errorf("Resource tag list is empty (in %s). Tags must be defined for %s to be discovered.", iface.account, job.Type)
			}

			for _, resourceTagMapping := range page.ResourceTagMappingList {
				resource := taggedResource{
					ARN:       aws.StringValue(resourceTagMapping.ResourceARN),
					Namespace: job.Type,
					Region:    region,
				}

				for _, t := range resourceTagMapping.Tags {
					resource.Tags = append(resource.Tags, Tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				} else {
					log.Debugf("Skipping resource %s because search tags do not match", resource.ARN)
				}
			}
			return pageNum < 100
		})
		if err != nil {
			return nil, err
		}
	}

	if svc.ResourceFunc != nil {
		newResources, err := svc.ResourceFunc(iface, job, region)
		if err != nil {
			return nil, err
		}
		resources = append(resources, newResources...)
	}

	if svc.FilterFunc != nil {
		filteredResources, err := svc.FilterFunc(iface, resources)
		if err != nil {
			return nil, err
		}
		resources = filteredResources
	}

	return resources, nil
}

func migrateTagsToPrometheus(tagData []*taggedResource, labelsSnakeCase bool) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

	tagList := make(map[string][]string)

	for _, d := range tagData {
		for _, entry := range d.Tags {
			if !stringInSlice(entry.Key, tagList[d.Namespace]) {
				tagList[d.Namespace] = append(tagList[d.Namespace], entry.Key)
			}
		}
	}

	for _, d := range tagData {
		promNs := strings.ToLower(d.Namespace)
		if !strings.HasPrefix(promNs, "aws") {
			promNs = "aws_" + promNs
		}
		name := promString(promNs) + "_info"
		promLabels := make(map[string]string)
		promLabels["name"] = d.ARN

		for _, entry := range tagList[d.Namespace] {
			labelKey := "tag_" + promStringTag(entry, labelsSnakeCase)
			promLabels[labelKey] = ""

			for _, rTag := range d.Tags {
				if entry == rTag.Key {
					promLabels[labelKey] = rTag.Value
				}
			}
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
