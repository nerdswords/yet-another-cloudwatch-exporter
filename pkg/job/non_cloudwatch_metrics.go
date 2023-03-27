package job

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/apitagging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/promutil"
	"regexp"
)

type additionalAWSMetric struct {
	additionalAWSMetrics func(resources []*model.TaggedResource, api apitagging.TagsInterface) ([]*promutil.PrometheusMetric, error)
}

var additionalAWSMetrics = map[string]additionalAWSMetric{
	"AWS/DynamoDB": {
		additionalAWSMetrics: func(resources []*model.TaggedResource, api apitagging.TagsInterface) ([]*promutil.PrometheusMetric, error) {
			additionalMetrics := make([]*promutil.PrometheusMetric, 0)
			for _, resource := range resources {
				re := regexp.MustCompile(":table/(.*)")
				match := re.FindStringSubmatch(resource.ARN)
				describeTableOutput, err := api.DynamoDBClient.DescribeTable(&dynamodb.DescribeTableInput{
					TableName: aws.String(match[1]),
				})
				if err != nil {
					return nil, err
				}
				labels := make(map[string]string)

				labels["name"] = resource.ARN
				labels["region"] = resource.Region
				accountIDSearchString := regexp.MustCompile(".*:(.*):table\\/.*")
				labels["account_id"] = accountIDSearchString.FindStringSubmatch(resource.ARN)[1]

				tableNameSearchString := regexp.MustCompile(".*:table\\/(.*)")
				labels["tableName"] = tableNameSearchString.FindStringSubmatch(resource.ARN)[1]

				itemCount := float64(*describeTableOutput.Table.ItemCount)
				p := promutil.PrometheusMetric{
					Name:   aws.String("aws_dynamodb_item_count"),
					Labels: labels,
					Value:  &itemCount,
				}
				additionalMetrics = append(additionalMetrics, &p)
			}
			return additionalMetrics, nil
		},
	},
}

func scrapeAdditionalMetrics(job *config.Job, resources []*model.TaggedResource, api apitagging.TagsInterface) ([]*promutil.PrometheusMetric, error) {

	additionalMetrics := make([]*promutil.PrometheusMetric, 0)
	svc := config.SupportedServices.GetService(job.Type)
	if ext, ok := additionalAWSMetrics[svc.Namespace]; ok {
		if ext.additionalAWSMetrics != nil {
			output, err := ext.additionalAWSMetrics(resources, api)
			if err != nil {
				return nil, err
			}
			additionalMetrics = append(additionalMetrics, output...)
		}
	}
	return additionalMetrics, nil
}
