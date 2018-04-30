package main

import (
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

func createElasticacheSession(region string) *elasticache.ElastiCache {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return elasticache.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeElasticacheDomains(discovery discovery) (resources awsResources) {
	c := createElasticacheSession(discovery.Region)
	resp, err := c.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		panic(err)
	}

	for _, ec := range resp.CacheClusters {
		resource := awsResource{}
		resource.Id = ec.CacheClusterId
		resource.Service = aws.String("elasticache")
		resource.Attributes = getElasticacheAttributes(discovery.ExportedAttributes, ec)
		resource.Tags = getElasticacheTags(c, resource.Id, discovery.Region)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources.Resources = append(resources.Resources, &resource)
		}
	}

	resources.CloudwatchInfo = getElasticacheCloudwatchInfo()

	return resources
}

func getElasticacheTags(c *elasticache.ElastiCache, resourceId *string, region string) (output []*tag) {
	arn := "arn:aws:elasticache:" + region + ":" + *getAwsArn() + ":cluster:" + *resourceId

	input := elasticache.ListTagsForResourceInput{ResourceName: &arn}

	awsTags, err := c.ListTagsForResource(&input)
	if err != nil {
		panic(err)
	}

	for _, awsTag := range awsTags.TagList {
		tag := tag{Key: *awsTag.Key, Value: *awsTag.Value}
		output = append(output, &tag)
	}
	return output
}

func getElasticacheAttributes(attributes []string, ec *elasticache.CacheCluster) map[string]*string {
	output := map[string]*string{
		"Engine":        aws.String(""),
		"EngineVersion": aws.String(""),
	}

	for _, attribute := range attributes {
		switch attribute {
		case "Engine":
			output["Engine"] = ec.Engine
		case "EngineVersion":
			output["EngineVersion"] = ec.EngineVersion
		}
	}
	return output
}

func getElasticacheCloudwatchInfo() *cloudwatchInfo {
	output := cloudwatchInfo{}
	output.DimensionName = aws.String("CacheClusterId")
	output.Namespace = aws.String("AWS/ElastiCache")
	output.CustomDimension = []*tag{}
	return &output
}
