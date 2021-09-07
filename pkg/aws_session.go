package exporter

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/sts"

	log "github.com/sirupsen/logrus"
)

// originally from aws_tags.go
func createSession(role Role, config *aws.Config) *session.Session {
	sess, err := session.NewSession(config)
	config.CredentialsChainVerboseErrors = aws.Bool(true)
	if err != nil {
		log.Fatalf("Failed to create session due to %v", err)
	}
	if role.RoleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, role.RoleArn, func(p *stscreds.AssumeRoleProvider) {
			if role.ExternalID != "" {
				p.ExternalID = aws.String(role.ExternalID)
			}
		})
	}
	return sess
}

// originally from aws_tags.go
func createAPIGatewaySession(region *string, role Role, fips bool) apigatewayiface.APIGatewayAPI {
	maxApiGatewaygAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxApiGatewaygAPIRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/apigateway.html
		endpoint := fmt.Sprintf("https://apigateway-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}
	return apigateway.New(createSession(role, config), config)
}

// originally from aws_tags.go
func createASGSession(region *string, role Role, fips bool) autoscalingiface.AutoScalingAPI {
	maxAutoScalingAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAutoScalingAPIRetries}
	if fips {
		// ToDo: Autoscaling does not have a FIPS endpoint
		// https://docs.aws.amazon.com/general/latest/gr/autoscaling_region.html
		// endpoint := fmt.Sprintf("https://autoscaling-plans-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}
	return autoscaling.New(createSession(role, config), config)
}

// originally from aws_cloudwatch.go
func createCloudwatchSession(region *string, role Role, fips bool) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region:                        aws.String(*region),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	}))

	maxCloudwatchRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxCloudwatchRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/cw_region.html
		endpoint := fmt.Sprintf("https://monitoring-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
	}

	if role.RoleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, role.RoleArn, func(p *stscreds.AssumeRoleProvider) {
			if role.ExternalID != "" {
				p.ExternalID = aws.String(role.ExternalID)
			}
		})
	}

	return cloudwatch.New(sess, config)
}

// originally from aws_tags.go
func createEC2Session(region *string, role Role, fips bool) ec2iface.EC2API {
	maxEC2APIRetries := 10
	config := &aws.Config{Region: region, MaxRetries: &maxEC2APIRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/ec2-service.html
		endpoint := fmt.Sprintf("https://ec2-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}
	return ec2.New(createSession(role, config), config)
}

// originally from aws_cloudwatch.go
func createStsSession(role Role) *sts.STS {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	}))
	maxStsRetries := 5
	config := &aws.Config{MaxRetries: &maxStsRetries}
	if log.IsLevelEnabled(log.DebugLevel) {
		config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
	}
	if role.RoleArn != "" {
		config.Credentials = stscreds.NewCredentials(sess, role.RoleArn, func(p *stscreds.AssumeRoleProvider) {
			if role.ExternalID != "" {
				p.ExternalID = aws.String(role.ExternalID)
			}
		})
	}
	return sts.New(sess, config)
}

// originally from aws_tags.go
func createTagSession(region *string, role Role, fips bool) *r.ResourceGroupsTaggingAPI {
	maxResourceGroupTaggingRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxResourceGroupTaggingRetries}
	if fips {
		// ToDo: Resource Groups Tagging API does not have FIPS compliant endpoints
		// https://docs.aws.amazon.com/general/latest/gr/arg.html
		// endpoint := fmt.Sprintf("https://tagging-fips.%s.amazonaws.com", *region)
		// config.Endpoint = aws.String(endpoint)
	}
	return r.New(createSession(role, config), config)
}
