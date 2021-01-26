package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"
)

type tagsData struct {
	ID      *string
	Matcher *string
	Tags    []*tag
	Service *string
	Region  *string
}

// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface/
type tagsInterface struct {
	client           resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asgClient        autoscalingiface.AutoScalingAPI
	apiGatewayClient apigatewayiface.APIGatewayAPI
	ec2Client        ec2iface.EC2API
	elbv2Client      elbv2.ELBV2
}

func createSession(roleArn string, config *aws.Config) *session.Session {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session due to %v", err)
	}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}
	return sess
}

func createTagSession(region *string, roleArn string) *r.ResourceGroupsTaggingAPI {
	maxResourceGroupTaggingRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxResourceGroupTaggingRetries}
	if *fips {
		// ToDo: Resource Groups Tagging API does not have FIPS compliant endpoints
		// https://docs.aws.amazon.com/general/latest/gr/arg.html
		// endpoint := fmt.Sprintf("https://tagging-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}
	return r.New(createSession(roleArn, config), config)
}

func createASGSession(region *string, roleArn string) autoscalingiface.AutoScalingAPI {
	maxAutoScalingAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAutoScalingAPIRetries}
	if *fips {
		// ToDo: Autoscaling does not have a FIPS endpoint
		// https://docs.aws.amazon.com/general/latest/gr/autoscaling_region.html
		// endpoint := fmt.Sprintf("https://autoscaling-plans-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}
	return autoscaling.New(createSession(roleArn, config), config)
}

func createEC2Session(region *string, roleArn string) ec2iface.EC2API {
	maxEC2APIRetries := 10
	config := &aws.Config{Region: region, MaxRetries: &maxEC2APIRetries}
	if *fips {
		// https://docs.aws.amazon.com/general/latest/gr/ec2-service.html
		endpoint := fmt.Sprintf("https://ec2-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}
	return ec2.New(createSession(roleArn, config), config)
}

func createAPIGatewaySession(region *string, roleArn string) apigatewayiface.APIGatewayAPI {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	maxApiGatewaygAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxApiGatewaygAPIRetries}
	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, roleArn)
	}
	if *fips {
		// https://docs.aws.amazon.com/general/latest/gr/apigateway.html
		endpoint := fmt.Sprintf("https://apigateway-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}
	return apigateway.New(sess, config)
}

func createELBV2Session(region *string, roleArn string) elbv2.ELBV2 {
	config := &aws.Config{Region: region}
	return *elbv2.New(createSession(roleArn, config), config)
}

func (iface tagsInterface) get(job job, region string) (resources []*tagsData, err error) {
	switch job.Type {
	case "asg":
		return iface.getTaggedAutoscalingGroups(job, region)
	case "ec2Spot":
		return iface.getTaggedEC2SpotInstances(job, region)
	case "tgwa":
		return iface.getTaggedTransitGatewayAttachments(job, region)

	}

	allResourceTypesFilters := map[string][]string{
		"alb":                   {"elasticloadbalancing:loadbalancer/app", "elasticloadbalancing:targetgroup"},
		"apigateway":            {"apigateway"},
		"appsync":               {"appsync"},
		"cf":                    {"cloudfront"},
		"docdb":                 {"rds:db", "rds:cluster"},
		"dynamodb":              {"dynamodb:table"},
		"ebs":                   {"ec2:volume"},
		"ec":                    {"elasticache:cluster"},
		"ec2":                   {"ec2:instance"},
		"ecs-svc":               {"ecs:cluster", "ecs:service"},
		"ecs-containerinsights": {"ecs:cluster", "ecs:service"},
		"efs":                   {"elasticfilesystem:file-system"},
		"elb":                   {"elasticloadbalancing:loadbalancer"},
		"emr":                   {"elasticmapreduce:cluster"},
		"es":                    {"es:domain"},
		"firehose":              {"firehose"},
		"fsx":                   {"fsx:file-system"},
		"gamelift":              {"gamelift"},
		"kinesis":               {"kinesis:stream"},
		"lambda":                {"lambda:function"},
		"ngw":                   {"ec2:natgateway"},
		"nlb":                   {"elasticloadbalancing:loadbalancer/net", "elasticloadbalancing:targetgroup"},
		"rds":                   {"rds:db", "rds:cluster"},
		"redshift":              {"redshift:cluster"},
		"r53r":                  {"route53resolver"},
		"s3":                    {"s3"},
		"sfn":                   {"states"},
		"sns":                   {"sns"},
		"sqs":                   {"sqs"},
		"tgw":                   {"ec2:transit-gateway"},
		"vpn":                   {"ec2:vpn-connection"},
		"kafka":                 {"kafka:cluster"},
		"wafv2":                 {"wafv2"},
	}
	var inputparams r.GetResourcesInput
	if resourceTypeFilters, ok := allResourceTypesFilters[job.Type]; ok {
		var filters []*string
		for _, filter := range resourceTypeFilters {
			filters = append(filters, aws.String(filter))
		}
		inputparams.ResourceTypeFilters = filters
	} else {
		log.Fatal("Not implemented resources:" + job.Type)
	}
	c := iface.client
	ctx := context.Background()
	pageNum := 0
	resourcePages := c.GetResourcesPagesWithContext(ctx, &inputparams, func(page *r.GetResourcesOutput, lastPage bool) bool {
		pageNum++
		resourceGroupTaggingAPICounter.Inc()
		for _, resourceTagMapping := range page.ResourceTagMappingList {
			resource := tagsData{}

			resource.ID = resourceTagMapping.ResourceARN

			resource.Service = &job.Type
			resource.Region = &region

			for _, t := range resourceTagMapping.Tags {
				resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
			}

			if resource.filterThroughTags(job.SearchTags) {
				resources = append(resources, &resource)
			}
		}
		return pageNum < 100
	})

	switch job.Type {
	case "alb", "nlb":
		var chunkedFilteredResources []*tagsData
		var targetGroupArns []*string
		var albArns []*string
		for _, r := range resources {
			if strings.Contains(*r.ID, "targetgroup/") {
				targetGroupArns = append(targetGroupArns, aws.String(*r.ID))
			} else {
				// Add all resources except target groups
				albArns = append(albArns, r.ID)
			}
		}
		// You can only describe 20 target groups at a time
		// Break the array of strings of target group names to multiple arrays
		// Each with a max size of 20
		chunkSize := 20
		var chunkedTargetGroupArns = chunkArrayOfStrings(targetGroupArns, chunkSize)

		// Loop through our target group sets (manual pagination!)
		for _, chunkedTargetGroupArnSet := range chunkedTargetGroupArns {
			// Describe the target groups and filter ones which have a matching alb
			targetGroupFilteredResults, err := iface.getTargetGroups(resources, chunkedTargetGroupArnSet, albArns)
			if err != nil {
				log.Errorf("Error describeTargetGroups for %s , err: %s", job.Type, err)
				// Do not clear the resource list. The old behavior.
				break
			}
			chunkedFilteredResources = append(chunkedFilteredResources, targetGroupFilteredResults...)
		}
		// Replace the list of resources which contained both albs and target groups with our filtered list
		resources = chunkedFilteredResources
	case "apigateway":
		// Get all the api gateways from aws
		apiGateways, errGet := iface.getTaggedApiGateway()
		if errGet != nil {
			log.Errorf("tagsInterface.get: apigateway: getTaggedApiGateway: %v", errGet)
			return resources, errGet
		}
		var filteredResources []*tagsData
		for _, r := range resources {
			// For each tagged resource, find the associated restApi
			// And swap out the ID with the name
			if strings.Contains(*r.ID, "/restapis") {
				restApiId := strings.Split(*r.ID, "/")[2]
				for _, apiGateway := range apiGateways.Items {
					if *apiGateway.Id == restApiId {
						r.Matcher = apiGateway.Name
					}
				}
				if r.Matcher == nil {
					log.Errorf("tagsInterface.get: apigateway: resource=%s restApiId=%s could not find gateway", *r.ID, restApiId)
					continue // exclude resource to avoid crash later
				}
				filteredResources = append(filteredResources, r)
			}
		}
		resources = filteredResources
	}
	return resources, resourcePages
}

// Breaks a single array of strings into a multiple array of strings
func chunkArrayOfStrings(targetGroupArns []*string, chunkSize int) (chunkedTargetGroupArns [][]*string) {
	for i := 0; i < len(targetGroupArns); i += chunkSize {
		end := i + chunkSize
		if end > len(targetGroupArns) {
			end = len(targetGroupArns)
		}
		chunkedTargetGroupArns = append(chunkedTargetGroupArns, targetGroupArns[i:end])
	}
	return chunkedTargetGroupArns
}

// We want to make this request in a background thread with pagination
// https://docs.aws.amazon.com/sdk-for-go/api/service/elbv2/#ELBV2.DescribeTargetGroups
func (iface tagsInterface) getTargetGroups(resources []*tagsData, targetGroupArns []*string, albArns []*string) (filteredResources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	// You cannot describe more than '20' target groups at a time
	// It feels like pageSize would handle this but it does not.
	// Using pageSize anyways
	pageSize := aws.Int64(20)
	return filteredResources, iface.elbv2Client.DescribeTargetGroupsPagesWithContext(ctx, &elbv2.DescribeTargetGroupsInput{TargetGroupArns: targetGroupArns, PageSize: pageSize},
		func(page *elbv2.DescribeTargetGroupsOutput, more bool) bool {
			pageNum++
			targetGroupsAPICounter.Inc()

			// For each of the target groups
			for _, tg := range page.TargetGroups {
				if len(tg.LoadBalancerArns) == 0 {
					log.Debugf("No LoadBalancerArns in targetGroup %s", *tg.TargetGroupArn)
					continue
				}
				// for each alb associated with this target group
				for _, tgLoadBalancerArn := range tg.LoadBalancerArns {
					// check each alb ARN
					for _, albArn := range albArns {
						if *albArn == *tgLoadBalancerArn {
							for _, res := range resources {
								// If our resource list has an entry for this target group
								// Which has an associated alb, add it to our filtered list
								if *res.ID == *tg.TargetGroupArn {
									filteredResources = append(filteredResources, res)
									break
								}
							}
						}
					}
				}
			}
			return pageNum < 100
		})
}

// Once the resourcemappingapi supports ASGs then this workaround method can be deleted
// https://docs.aws.amazon.com/sdk-for-go/api/service/resourcegroupstaggingapi/
func (iface tagsInterface) getTaggedAutoscalingGroups(job job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.asgClient.DescribeAutoScalingGroupsPagesWithContext(ctx, &autoscaling.DescribeAutoScalingGroupsInput{},
		func(page *autoscaling.DescribeAutoScalingGroupsOutput, more bool) bool {
			pageNum++
			autoScalingAPICounter.Inc()

			for _, asg := range page.AutoScalingGroups {
				resource := tagsData{}

				// Transform the ASG ARN into something which looks more like an ARN from the ResourceGroupTaggingAPI
				parts := strings.Split(*asg.AutoScalingGroupARN, ":")
				resource.ID = aws.String(fmt.Sprintf("arn:%s:autoscaling:%s:%s:%s", parts[1], parts[3], parts[4], parts[7]))

				resource.Service = &job.Type
				resource.Region = &region

				for _, t := range asg.Tags {
					resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				}
			}
			return pageNum < 100
		})
}

// Get all ApiGateways REST
func (iface tagsInterface) getTaggedApiGateway() (*apigateway.GetRestApisOutput, error) {
	ctx := context.Background()
	apiGatewayAPICounter.Inc()
	var limit int64 = 500 // max number of results per page. default=25, max=500
	const maxPages = 10
	input := apigateway.GetRestApisInput{Limit: &limit}
	output := apigateway.GetRestApisOutput{}
	var pageNum int
	err := iface.apiGatewayClient.GetRestApisPagesWithContext(ctx, &input, func(page *apigateway.GetRestApisOutput, lastPage bool) bool {
		pageNum++
		output.Items = append(output.Items, page.Items...)
		return pageNum <= maxPages
	})
	return &output, err
}

func (iface tagsInterface) getTaggedTransitGatewayAttachments(job job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.ec2Client.DescribeTransitGatewayAttachmentsPagesWithContext(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{},
		func(page *ec2.DescribeTransitGatewayAttachmentsOutput, more bool) bool {
			pageNum++
			ec2APICounter.Inc()

			for _, tgwa := range page.TransitGatewayAttachments {
				resource := tagsData{}

				resource.ID = aws.String(fmt.Sprintf("%s/%s", *tgwa.TransitGatewayId, *tgwa.TransitGatewayAttachmentId))

				resource.Service = &job.Type
				resource.Region = &region

				for _, t := range tgwa.Tags {
					resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				}
			}
			return pageNum < 100
		})
}

func (iface tagsInterface) getTaggedEC2SpotInstances(job job, region string) (resources []*tagsData, err error) {
	ctx := context.Background()
	pageNum := 0
	return resources, iface.ec2Client.DescribeSpotFleetRequestsPagesWithContext(ctx, &ec2.DescribeSpotFleetRequestsInput{},
		func(page *ec2.DescribeSpotFleetRequestsOutput, more bool) bool {
			pageNum++
			ec2APICounter.Inc()

			for _, ec2Spot := range page.SpotFleetRequestConfigs {
				resource := tagsData{}

				resource.ID = aws.String(*ec2Spot.SpotFleetRequestId)

				resource.Service = &job.Type
				resource.Region = &region

				for _, t := range ec2Spot.Tags {
					resource.Tags = append(resource.Tags, &tag{Key: *t.Key, Value: *t.Value})
				}

				if resource.filterThroughTags(job.SearchTags) {
					resources = append(resources, &resource)
				}
			}
			return pageNum < 100
		})
}

func migrateTagsToPrometheus(tagData []*tagsData) []*PrometheusMetric {
	output := make([]*PrometheusMetric, 0)

	tagList := make(map[string][]string)

	for _, d := range tagData {
		for _, entry := range d.Tags {
			if !stringInSlice(entry.Key, tagList[*d.Service]) {
				tagList[*d.Service] = append(tagList[*d.Service], entry.Key)
			}
		}
	}

	for _, d := range tagData {
		name := "aws_" + promString(*d.Service) + "_info"
		promLabels := make(map[string]string)
		promLabels["name"] = *d.ID

		for _, entry := range tagList[*d.Service] {
			labelKey := "tag_" + promStringTag(entry)
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
