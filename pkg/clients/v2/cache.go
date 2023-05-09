package v2

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/amp"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/storagegateway"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	aws_logging "github.com/aws/smithy-go/logging"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account"
	account_v2 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/account/v2"
	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	cloudwatch_v2 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch/v2"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging"
	tagging_v2 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging/v2"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

type awsRegion = string

type clientCache struct {
	logger    logging.Logger
	clients   map[config.Role]map[awsRegion]*cachedClients
	mu        sync.Mutex
	refreshed bool
	cleared   bool
}

type cachedClients struct {
	awsConfig *aws.Config
	// if we know that this job is only used for static
	// then we don't have to construct as many cached connections
	// later on
	onlyStatic bool
	sts        *sts.Client
	cloudwatch cloudwatch_client.Client
	tagging    tagging.Client
	account    account.Client
}

func NewCache(cfg config.ScrapeConf, fips bool, logger logging.Logger) (clients.Cache, error) {
	var options []func(*aws_config.LoadOptions) error
	options = append(options, aws_config.WithLogger(aws_logging.LoggerFunc(func(classification aws_logging.Classification, format string, v ...interface{}) {
		if classification == aws_logging.Debug && logger.IsDebugEnabled() {
			logger.Debug(fmt.Sprintf(format, v...))
		} else if classification == aws_logging.Warn {
			logger.Warn(fmt.Sprintf(format, v...))
		} else { // AWS logging only supports debug or warn, log everything else as error
			logger.Error(fmt.Errorf("unexected aws error classification: %s", classification), fmt.Sprintf(format, v...))
		}
	})))

	options = append(options, aws_config.WithLogConfigurationWarnings(true))

	endpointURLOverride := os.Getenv("AWS_ENDPOINT_URL")
	if endpointURLOverride != "" {
		options = append(options, aws_config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: endpointURLOverride,
			}, nil
		})))
	}

	if fips {
		options = append(options, aws_config.WithUseFIPSEndpoint(aws.FIPSEndpointStateEnabled))
	}

	c, err := aws_config.LoadDefaultConfig(context.TODO(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to load default aws config: %w", err)
	}

	cache := map[config.Role]map[awsRegion]*cachedClients{}
	for _, discoveryJob := range cfg.Discovery.Jobs {
		for _, role := range discoveryJob.Roles {
			if _, ok := cache[role]; !ok {
				cache[role] = map[awsRegion]*cachedClients{}
			}
			for _, region := range discoveryJob.Regions {
				regionConfig := awsConfigForRegion(role, &c, region, role)
				cache[role][region] = &cachedClients{
					awsConfig:  regionConfig,
					onlyStatic: false,
				}
			}
		}
	}

	for _, customNamespaceJob := range cfg.CustomNamespace {
		for _, role := range customNamespaceJob.Roles {
			if _, ok := cache[role]; !ok {
				cache[role] = map[awsRegion]*cachedClients{}
			}
			for _, region := range customNamespaceJob.Regions {
				// Discovery job client definitions have precedence
				if _, exists := cache[role][region]; !exists {
					regionConfig := awsConfigForRegion(role, &c, region, role)
					cache[role][region] = &cachedClients{
						awsConfig:  regionConfig,
						onlyStatic: false,
					}
				}
			}
		}
	}

	for _, staticJob := range cfg.Static {
		for _, role := range staticJob.Roles {
			if _, ok := cache[role]; !ok {
				cache[role] = map[awsRegion]*cachedClients{}
			}
			for _, region := range staticJob.Regions {
				// Discovery job client definitions have precedence
				if _, exists := cache[role][region]; !exists {
					regionConfig := awsConfigForRegion(role, &c, region, role)
					cache[role][region] = &cachedClients{
						awsConfig:  regionConfig,
						onlyStatic: true,
					}
				}
			}
		}
	}

	return &clientCache{
		logger:  logger,
		clients: cache,
	}, nil
}

func (c *clientCache) GetCloudwatchClient(region string, role config.Role, concurrencyLimit int) cloudwatch_client.Client {
	if !c.refreshed {
		// if we have not refreshed then we need to lock in case we are accessing concurrently
		c.mu.Lock()
		defer c.mu.Unlock()
	}
	if client := c.clients[role][region].cloudwatch; client != nil {
		return cloudwatch_client.NewLimitedConcurrencyClient(client, concurrencyLimit)
	}
	c.clients[role][region].cloudwatch = cloudwatch_v2.NewClient(c.logger, c.createCloudwatchClient(c.clients[role][region].awsConfig))
	return cloudwatch_client.NewLimitedConcurrencyClient(c.clients[role][region].cloudwatch, concurrencyLimit)
}

func (c *clientCache) GetTaggingClient(region string, role config.Role, concurrencyLimit int) tagging.Client {
	if !c.refreshed {
		// if we have not refreshed then we need to lock in case we are accessing concurrently
		c.mu.Lock()
		defer c.mu.Unlock()
	}
	if client := c.clients[role][region].tagging; client != nil {
		return tagging.NewLimitedConcurrencyClient(client, concurrencyLimit)
	}
	c.clients[role][region].tagging = tagging_v2.NewClient(
		c.logger,
		c.createTaggingClient(c.clients[role][region].awsConfig),
		c.createAutoScalingClient(c.clients[role][region].awsConfig),
		c.createAPIGatewayClient(c.clients[role][region].awsConfig),
		c.createAPIGatewayV2Client(c.clients[role][region].awsConfig),
		c.createEC2Client(c.clients[role][region].awsConfig),
		c.createDMSClient(c.clients[role][region].awsConfig),
		c.createPrometheusClient(c.clients[role][region].awsConfig),
		c.createStorageGatewayClient(c.clients[role][region].awsConfig),
	)
	return tagging.NewLimitedConcurrencyClient(c.clients[role][region].tagging, concurrencyLimit)
}

func (c *clientCache) GetAccountClient(region string, role config.Role) account.Client {
	if !c.refreshed {
		// if we have not refreshed then we need to lock in case we are accessing concurrently
		c.mu.Lock()
		defer c.mu.Unlock()
	}
	if client := c.clients[role][region].account; client != nil {
		return client
	}
	c.clients[role][region].account = account_v2.NewClient(c.logger, c.createStsClient(c.clients[role][region].awsConfig))
	return c.clients[role][region].account
}

func (c *clientCache) Refresh() {
	if c.refreshed {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Avoid double refresh in the event Refresh() is called concurrently
	if c.refreshed {
		return
	}

	for _, regionClients := range c.clients {
		for _, cache := range regionClients {
			cache.cloudwatch = cloudwatch_v2.NewClient(c.logger, c.createCloudwatchClient(cache.awsConfig))
			if cache.onlyStatic {
				continue
			}

			cache.tagging = tagging_v2.NewClient(
				c.logger,
				c.createTaggingClient(cache.awsConfig),
				c.createAutoScalingClient(cache.awsConfig),
				c.createAPIGatewayClient(cache.awsConfig),
				c.createAPIGatewayV2Client(cache.awsConfig),
				c.createEC2Client(cache.awsConfig),
				c.createDMSClient(cache.awsConfig),
				c.createPrometheusClient(cache.awsConfig),
				c.createStorageGatewayClient(cache.awsConfig),
			)

			cache.account = account_v2.NewClient(c.logger, c.createStsClient(cache.awsConfig))
		}
	}

	c.refreshed = true
	c.cleared = false
}

func (c *clientCache) Clear() {
	if c.cleared {
		return
	}
	// Prevent concurrent reads/write if clear is called during execution
	c.mu.Lock()
	defer c.mu.Unlock()
	// Avoid double clear in the event Refresh() is called concurrently
	if c.cleared {
		return
	}

	for _, regions := range c.clients {
		for _, cache := range regions {
			cache.sts = nil
			cache.cloudwatch = nil
			cache.account = nil
			cache.tagging = nil
		}
	}

	c.refreshed = false
	c.cleared = true
}

func (c *clientCache) createCloudwatchClient(regionConfig *aws.Config) *cloudwatch.Client {
	return cloudwatch.NewFromConfig(*regionConfig, func(options *cloudwatch.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}
		options.Retryer = retry.NewStandard(func(options *retry.StandardOptions) {
			options.MaxAttempts = 5
			options.MaxBackoff = 3 * time.Second

			// existing settings
			// TODO how to tell the difference between throttle and non-throttle errors now?
			//	NumMaxRetries: 5
			//	//MaxThrottleDelay and MinThrottleDelay used for throttle errors
			//	MaxThrottleDelay: 10 * time.Second
			//	MinThrottleDelay: 1 * time.Second
			//	// For other errors
			//	MaxRetryDelay: 3 * time.Second
			//	MinRetryDelay: 1 * time.Second
		})
	})
}

func (c *clientCache) createTaggingClient(regionConfig *aws.Config) *resourcegroupstaggingapi.Client {
	return resourcegroupstaggingapi.NewFromConfig(*regionConfig, func(options *resourcegroupstaggingapi.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createAutoScalingClient(assumedConfig *aws.Config) *autoscaling.Client {
	return autoscaling.NewFromConfig(*assumedConfig, func(options *autoscaling.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createEC2Client(assumedConfig *aws.Config) *ec2.Client {
	return ec2.NewFromConfig(*assumedConfig, func(options *ec2.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createDMSClient(assumedConfig *aws.Config) *databasemigrationservice.Client {
	return databasemigrationservice.NewFromConfig(*assumedConfig, func(options *databasemigrationservice.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createAPIGatewayClient(assumedConfig *aws.Config) *apigateway.Client {
	return apigateway.NewFromConfig(*assumedConfig, func(options *apigateway.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createAPIGatewayV2Client(assumedConfig *aws.Config) *apigatewayv2.Client {
	return apigatewayv2.NewFromConfig(*assumedConfig, func(options *apigatewayv2.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createStorageGatewayClient(assumedConfig *aws.Config) *storagegateway.Client {
	return storagegateway.NewFromConfig(*assumedConfig, func(options *storagegateway.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createPrometheusClient(assumedConfig *aws.Config) *amp.Client {
	return amp.NewFromConfig(*assumedConfig, func(options *amp.Options) {
		if c.logger.IsDebugEnabled() {
			options.ClientLogMode = aws.LogRequestWithBody | aws.LogResponseWithBody
		}

		options.RetryMaxAttempts = 5
	})
}

func (c *clientCache) createStsClient(awsConfig *aws.Config) *sts.Client {
	//TODO need to use regional sts setting here
	return sts.NewFromConfig(*awsConfig, func(options *sts.Options) {
		options.RetryMaxAttempts = 5
	})
}

var defaultRole = config.Role{}

func awsConfigForRegion(r config.Role, c *aws.Config, region awsRegion, role config.Role) *aws.Config {
	regionalSts := sts.NewFromConfig(*c, func(options *sts.Options) {
		options.Region = region
	})
	if r == defaultRole {
		// We are not using delegated access so return the original config and regional sts
		return c
	}

	// based on https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/stscreds#hdr-Assume_Role
	// found via https://github.com/aws/aws-sdk-go-v2/issues/1382
	credentials := stscreds.NewAssumeRoleProvider(regionalSts, role.RoleArn, func(options *stscreds.AssumeRoleOptions) {
		if role.ExternalID != "" {
			options.ExternalID = aws.String(role.ExternalID)
		}
	})

	delegatedConfig := c.Copy()
	delegatedConfig.Region = region
	delegatedConfig.Credentials = aws.NewCredentialsCache(credentials)

	return &delegatedConfig
}
