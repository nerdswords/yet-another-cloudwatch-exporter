package main

import (
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticsearchservice"
)

func createEsSession(region string) *elasticsearchservice.ElasticsearchService {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return elasticsearchservice.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeElasticsearchDomains(discovery discovery) (resources awsResources) {
	c := createEsSession(discovery.Region)

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
		resource.Tags = getEsTags(c, es.ARN)
		resource.Attributes = getElasticsearchAttributes(discovery.ExportedAttributes, es)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources.Resources = append(resources.Resources, &resource)
		}
	}

	resources.CloudwatchInfo = getEsCloudwatchInfo()

	return resources
}

func getEsTags(c *elasticsearchservice.ElasticsearchService, arn *string) (output []*tag) {
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

//*elasticsearchservice.ElasticsearchService
func getElasticsearchAttributes(attributes []string, es *elasticsearchservice.ElasticsearchDomainStatus) map[string]*string {
	output := map[string]*string{
		"VolumeSize":           aws.String(""),
		"DedicatedMasterCount": aws.String(""),
		"InstanceCount":        aws.String(""),
		"ElasticsearchVersion": aws.String(""),
	}

	for _, attribute := range attributes {
		switch attribute {
		case "VolumeSize":
			output["VolumeSize"] = intToString(es.EBSOptions.VolumeSize)
		case "DedicatedMasterCount":
			output["DedicatedMasterCount"] = intToString(es.ElasticsearchClusterConfig.DedicatedMasterCount)
		case "InstanceCount":
			output["InstanceCount"] = intToString(es.ElasticsearchClusterConfig.InstanceCount)
		case "ElasticsearchVersion":
			output["ElasticsearchVersion"] = es.ElasticsearchVersion
		}
	}
	return output
}

func getEsCloudwatchInfo() *cloudwatchInfo {
	output := cloudwatchInfo{}
	output.DimensionName = aws.String("DomainName")
	output.Namespace = aws.String("AWS/ES")
	output.CustomDimension = []*tag{&tag{Key: "ClientId", Value: *getAwsArn()}}
	return &output
}
