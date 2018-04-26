package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticsearchservice"
)

func createEcSession(region string) *elasticsearchservice.ElasticsearchService {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return elasticsearchservice.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeElasticsearchServices(discovery discovery) (resources awsResources) {
	c := createEcSession("eu-west-1")

	listResp, err := c.ListDomainNames(&elasticsearchservice.ListDomainNamesInput{})
	if err != nil {
		panic(err)
	}

	domainNames := []*string{}

	for _, e := range listResp.DomainNames {
		domainNames = append(domainNames, e.DomainName)
	}

	params := &elasticsearchservice.DescribeElasticsearchDomainsInput{DomainNames: domainNames}

	resp, err := c.DescribeElasticsearchDomains(params)
	if err != nil {
		panic(err)
	}

	for _, es := range resp.DomainStatusList {
		resource := awsResource{}
		resource.Id = es.DomainName
		resource.Service = aws.String("es")
		resource.Tags = getEsTags(es.ARN)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources.Resources = append(resources.Resources, &resource)
		}
	}

	resources.CloudwatchInfo = getEsCloudwatchInfo()

	return resources
}

func getEsTags(arn *string) (output []*tag) {
	c := createEcSession("eu-west-1")

	input := elasticsearchservice.ListTagsInput{ARN: arn}

	awsTags, err := c.ListTags(&input)
	if err != nil {
		panic(err)
	}

	for _, awsTag := range awsTags.TagList {
		tag := tag{Key: *awsTag.Key, Value: *awsTag.Value}
		output = append(output, &tag)
	}
	return output
}

func getEsCloudwatchInfo() *cloudwatchInfo {
	output := cloudwatchInfo{}
	output.DimensionName = aws.String("DomainName")
	output.Namespace = aws.String("AWS/ES")
	output.CustomDimension = []*tag{&tag{Key: "ClientId", Value: "NOT_IMPLEMENTED_YET"}}
	return &output
}
