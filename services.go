package main

import (
	"github.com/aws/aws-sdk-go/aws"
)

type serviceConfig struct {
	ResourceFilters  []*string
	DimensionRegexps []*string
}

var (
	supportedServices = map[string]serviceConfig{
		"AWS/ApplicationELB": {
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer/app"),
				aws.String("elasticloadbalancing:targetgroup"),
			},
			DimensionRegexps: []*string{
				aws.String(":(?P<TargetGroup>targetgroup/.+)"),
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		},
		"AWS/ApiGateway": {
			ResourceFilters: []*string{
				aws.String("apigateway"),
			},
			DimensionRegexps: []*string{
				aws.String("apis/(?P<ApiName>[^/]+)$"),
				aws.String("apis/(?P<ApiName>[^/]+)/stages/(?P<Stage>[^/]+)$"),
				aws.String("apis/(?P<ApiName>[^/]+)/resources/(?P<Resource>[^/]+)$"),
				aws.String("apis/(?P<ApiName>[^/]+)/resources/(?P<Resource>[^/]+)/methods/(?P<Method>[^/]+)$"),
			},
		},
		"AWS/AppSync": {
			ResourceFilters: []*string{
				aws.String("appsync"),
			},
			DimensionRegexps: []*string{
				aws.String("apis/(?P<GraphQLAPIId>[^/]+)"),
			},
		},
		"AWS/AutoScaling": {
			DimensionRegexps: []*string{
				aws.String("autoScalingGroupName/(?P<AutoScalingGroupName>[^/]+)"),
			},
		},
		"AWS/Billing": {},
		"AWS/CloudFront": {
			ResourceFilters: []*string{
				aws.String("cloudfront:distribution"),
			},
			DimensionRegexps: []*string{
				aws.String("distribution/(?P<DistributionId>[^/]+)"),
			},
		},
		"AWS/Cognito": {
			ResourceFilters: []*string{
				aws.String("cognito-idp:userpool"),
			},
			DimensionRegexps: []*string{
				aws.String("userpool/(?P<UserPool>[^/]+)"),
			},
		},
		"AWS/DocDB": {
			ResourceFilters: []*string{
				aws.String("rds:db"),
				aws.String("rds:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster:(?P<DBClusterIdentifier>[^/]+)"),
				aws.String("db:(?P<DBInstanceIdentifier>[^/]+)"),
			},
		},
		"AWS/DynamoDB": {
			ResourceFilters: []*string{
				aws.String("dynamodb:table"),
			},
			DimensionRegexps: []*string{
				aws.String(":table/(?P<TableName>[^/]+)"),
			},
		},
		"AWS/EBS": {
			ResourceFilters: []*string{
				aws.String("ec2:volume"),
			},
			DimensionRegexps: []*string{
				aws.String("volume/(?P<VolumeId>[^/]+)"),
			},
		},
		"AWS/ElastiCache": {
			ResourceFilters: []*string{
				aws.String("elasticache:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster:(?P<CacheClusterId>[^/]+)"),
			},
		},
		"AWS/EC2": {
			ResourceFilters: []*string{
				aws.String("ec2:instance"),
			},
			DimensionRegexps: []*string{
				aws.String("instance/(?P<InstanceId>[^/]+)"),
			},
		},
		"AWS/EC2Spot": {
			DimensionRegexps: []*string{
				aws.String("(?P<FleetRequestId>.*)"),
			},
		},
		"AWS/ECS": {
			ResourceFilters: []*string{
				aws.String("ecs:cluster"),
				aws.String("ecs:service"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<ClusterName>[^/]+)"),
				aws.String("service/(?P<ClusterName>[^/]+)/([^/]+)"),
			},
		},
		"ECS/ContainerInsights": {
			ResourceFilters: []*string{
				aws.String("ecs:cluster"),
				aws.String("ecs:service"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<ClusterName>[^/]+)"),
				aws.String("service/(?P<ClusterName>[^/]+)/([^/]+)"),
			},
		},
		"AWS/EFS": {
			ResourceFilters: []*string{
				aws.String("elasticfilesystem:file-system"),
			},
			DimensionRegexps: []*string{
				aws.String("file-system/(?P<FileSystemId>[^/]+)"),
			},
		},
		"AWS/ELB": {
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer"),
			},
			DimensionRegexps: []*string{
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		},
		"AWS/ElasticMapReduce": {
			ResourceFilters: []*string{
				aws.String("elasticmapreduce:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<JobFlowId>[^/]+)"),
			},
		},
		"AWS/ES": {
			ResourceFilters: []*string{
				aws.String("es:domain"),
			},
			DimensionRegexps: []*string{
				aws.String(":domain/(?P<DomainName>[^/]+)"),
			},
		},
		"AWS/Firehose": {
			ResourceFilters: []*string{
				aws.String("firehose"),
			},
			DimensionRegexps: []*string{
				aws.String(":deliverystream/(?P<DeliveryStreamName>[^/]+)"),
			},
		},
		"AWS/FSx": {
			ResourceFilters: []*string{
				aws.String("fsx:file-system"),
			},
			DimensionRegexps: []*string{
				aws.String("file-system/(?P<FileSystemId>[^/]+)"),
			},
		},
		"AWS/GameLift": {
			ResourceFilters: []*string{
				aws.String("gamelift"),
			},
			DimensionRegexps: []*string{
				aws.String(":fleet/(?P<FleetId>[^/]+)"),
			},
		},
		"Glue": {
			ResourceFilters: []*string{
				aws.String("glue:job"),
			},
			DimensionRegexps: []*string{
				aws.String(":job/(?P<JobName>[^/]+)"),
			},
		},
		"AWS/IoT": {
			ResourceFilters: []*string{
				aws.String("iot:rule"),
				aws.String("iot:provisioningtemplate"),
			},
			DimensionRegexps: []*string{
				aws.String(":rule/(?P<RuleName>[^/]+)"),
				aws.String(":provisioningtemplate/(?P<TemplateName>[^/]+)"),
			},
		},
		"AWS/Kafka": {
			ResourceFilters: []*string{
				aws.String("kafka:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster/(?P<Cluster_Name>[^/]+)"),
			},
		},
		"AWS/Kinesis": {
			ResourceFilters: []*string{
				aws.String("kinesis:stream"),
			},
			DimensionRegexps: []*string{
				aws.String(":stream/(?P<StreamName>[^/]+)"),
			},
		},
		"AWS/Lambda": {
			ResourceFilters: []*string{
				aws.String("lambda:function"),
			},
			DimensionRegexps: []*string{
				aws.String(":function:(?P<FunctionName>[^/]+)"),
			},
		},
		"AWS/NATGateway": {
			ResourceFilters: []*string{
				aws.String("ec2:natgateway"),
			},
			DimensionRegexps: []*string{
				aws.String("natgateway/(?P<NatGatewayId>[^/]+)"),
			},
		},
		"AWS/NetworkELB": {
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer/net"),
				aws.String("elasticloadbalancing:targetgroup"),
			},
			DimensionRegexps: []*string{
				aws.String(":(?P<TargetGroup>targetgroup/.+)"),
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		},
		"AWS/RDS": {
			ResourceFilters: []*string{
				aws.String("rds:db"),
				aws.String("rds:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster:(?P<DBClusterIdentifier>[^/]+)"),
				aws.String(":db:(?P<DBInstanceIdentifier>[^/]+)"),
			},
		},
		"AWS/Redshift": {
			ResourceFilters: []*string{
				aws.String("redshift:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster:(?P<ClusterIdentifier>[^/]+)"),
			},
		},
		"AWS/Route53Resolver": {
			ResourceFilters: []*string{
				aws.String("route53resolver"),
			},
			DimensionRegexps: []*string{
				aws.String(":resolver-endpoint/(?P<EndpointId>[^/]+)"),
			},
		},
		"AWS/S3": {
			ResourceFilters: []*string{
				aws.String("s3"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<BucketName>[^:]+)$"),
			},
		},
		"AWS/SES": {},
		"AWS/States": {
			ResourceFilters: []*string{
				aws.String("states"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<StateMachineArn>.*)"),
			},
		},
		"AWS/SNS": {
			ResourceFilters: []*string{
				aws.String("sns"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<TopicName>[^:]+)$"),
			},
		},
		"AWS/SQS": {
			ResourceFilters: []*string{
				aws.String("sqs"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<QueueName>[^:]+)$"),
			},
		},
		"AWS/TransitGateway": {
			ResourceFilters: []*string{
				aws.String("ec2:transit-gateway"),
			},
			DimensionRegexps: []*string{
				aws.String(":transit-gateway/(?P<TransitGateway>[^/]+)"),
				aws.String(":transit-gateway-attachment/(?P<TransitGateway>[^/]+)/(?P<TransitGatewayAttachment>[^/]+)"),
			},
		},
		"AWS/VPN": {
			ResourceFilters: []*string{
				aws.String("ec2:vpn-connection"),
			},
			DimensionRegexps: []*string{
				aws.String(":vpn-connection/(?P<VpnId>[^/]+)"),
			},
		},
		"AWS/WAFV2": {
			ResourceFilters: []*string{
				aws.String("wafv2"),
			},
			DimensionRegexps: []*string{
				aws.String("/webacl/(?P<WebACL>[^/]+)"),
			},
		},
	}
)
