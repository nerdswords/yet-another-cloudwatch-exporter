package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

type tagsData struct {
	ID        *string
	Tags      []*Tag
	Namespace *string
	Region    *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	client           resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asgClient        autoscalingiface.AutoScalingAPI
	apiGatewayClient apigatewayiface.APIGatewayAPI
	ec2Client        ec2iface.EC2API
}

func createTagSession(sess *session.Session, region *string, role Role, fips bool) *r.ResourceGroupsTaggingAPI {
	maxResourceGroupTaggingRetries := 5
	config := &aws.Config{
		Region:                        region,
		MaxRetries:                    &maxResourceGroupTaggingRetries,
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	if fips {
		// ToDo: Resource Groups Tagging API does not have FIPS compliant endpoints
		// https://docs.aws.amazon.com/general/latest/gr/arg.html
		// endpoint := fmt.Sprintf("https://tagging-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}

	return r.New(sess, setSTSCreds(sess, config, role))
}

func createASGSession(sess *session.Session, region *string, role Role, fips bool) autoscalingiface.AutoScalingAPI {
	maxAutoScalingAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAutoScalingAPIRetries}
	if fips {
		// ToDo: Autoscaling does not have a FIPS endpoint
		// https://docs.aws.amazon.com/general/latest/gr/autoscaling_region.html
		// endpoint := fmt.Sprintf("https://autoscaling-plans-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}

	return autoscaling.New(sess, setSTSCreds(sess, config, role))
}

func createEC2Session(sess *session.Session, region *string, role Role, fips bool) ec2iface.EC2API {
	maxEC2APIRetries := 10
	config := &aws.Config{Region: region, MaxRetries: &maxEC2APIRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/ec2-service.html
		endpoint := fmt.Sprintf("https://ec2-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	return ec2.New(sess, setSTSCreds(sess, config, role))
}

func createAPIGatewaySession(sess *session.Session, region *string, role Role, fips bool) apigatewayiface.APIGatewayAPI {
	maxApiGatewaygAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxApiGatewaygAPIRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/apigateway.html
		endpoint := fmt.Sprintf("https://apigateway-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	return apigateway.New(sess, setSTSCreds(sess, config, role))
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
				log.Debugf("Resource tag list is empty. Tags must be defined for %s to be discovered.", job.Type)
			}

			for _, resourceTagMapping := range page.ResourceTagMappingList {
				resource := tagsData{
					ID:        resourceTagMapping.ResourceARN,
					Namespace: &job.Type,
					Region:    &region,
				}

				for _, t := range resourceTagMapping.Tags {
					resource.Tags = append(resource.Tags, &Tag{Key: *t.Key, Value: *t.Value})
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
			if !stringInSlice(entry.Key, tagList[*d.Namespace]) {
				tagList[*d.Namespace] = append(tagList[*d.Namespace], entry.Key)
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
