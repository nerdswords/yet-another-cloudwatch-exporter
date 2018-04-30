package main

import (
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

func createDatabaseSession(region string) *rds.RDS {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	return rds.New(sess, &aws.Config{Region: aws.String(region)})
}

func describeDatabases(discovery discovery) (resources awsResources) {
	c := createDatabaseSession(discovery.Region)

	params := &rds.DescribeDBInstancesInput{}

	resp, err := c.DescribeDBInstances(params)
	if err != nil {
		panic(err)
	}

	for _, i := range resp.DBInstances {
		resource := awsResource{Id: i.DBInstanceIdentifier}
		resource.Service = aws.String("rds")
		resource.Tags = getDatabaseTags(c, i.DBInstanceArn)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources.Resources = append(resources.Resources, &resource)
		}
	}

	resources.CloudwatchInfo = getRdsCloudwatchInfo()

	return resources
}

func getDatabaseTags(c *rds.RDS, arn *string) (output []*tag) {
	input := rds.ListTagsForResourceInput{ResourceName: arn}

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

func getRdsCloudwatchInfo() *cloudwatchInfo {
	output := cloudwatchInfo{}
	output.DimensionName = aws.String("DBInstanceIdentifier")
	output.Namespace = aws.String("AWS/RDS")
	output.CustomDimension = []*tag{}
	return &output
}
