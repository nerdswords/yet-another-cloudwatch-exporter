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

func describeDatabases(discovery discovery) (resources []*resourceWrapper) {
	c := createDatabaseSession("eu-west-1")

	params := &rds.DescribeDBInstancesInput{}

	resp, err := c.DescribeDBInstances(params)
	if err != nil {
		panic(err)
	}

	for _, i := range resp.DBInstances {
		resource := resourceWrapper{Id: i.DBInstanceIdentifier}
		resource.Service = aws.String("rds")
		resource.Tags = getDatabaseTags(i.DBInstanceArn)
		if resource.filterThroughTags(discovery.SearchTags) {
			resources = append(resources, &resource)
		}
	}

	return resources
}

func getDatabaseTags(arn *string) (output []*searchTag) {
	c := createDatabaseSession("eu-west-1")

	input := rds.ListTagsForResourceInput{ResourceName: arn}

	awsTags, err := c.ListTagsForResource(&input)
	if err != nil {
		panic(err)
	}

	for _, awsTag := range awsTags.TagList {
		tag := searchTag{Key: *awsTag.Key, Value: *awsTag.Value}
		output = append(output, &tag)
	}
	return output
}
