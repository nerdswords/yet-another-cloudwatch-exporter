package main

import (
	"github.com/ivx/yet-another-cloudwatch-exporter/discovery/ec2spot"
	"github.com/ivx/yet-another-cloudwatch-exporter/discovery/autoscaling"
)

type serviceConfig struct {
	Resources  []string
	Dimensions []string
}

var (
	supportedServices = map[string]serviceConfig{
		"AWS/ApplicationELB": {
			ResourceFilters: []string{
				"elasticloadbalancing:loadbalancer/app", 
				"elasticloadbalancing:targetgroup",
			},
			DimensionRegexps: []string{
				":(?P<TargetGroup>targetgroup/.+)",
				":loadbalancer/(?P<LoadBalancer>.+)$",
			},
		},
		"AWS/ApiGateway": {
			ResourceFilters: []string{"apigateway"},
			DimensionRegexps: []string{
				"apis/(?P<ApiName>[^/]+)$",
				"apis/(?P<ApiName>[^/]+)/stages/(?P<Stage>[^/]+)$",
				"apis/(?P<ApiName>[^/]+)/resources/(?P<Resource>[^/]+)$",
				"apis/(?P<ApiName>[^/]+)/resources/(?P<Resource>[^/]+)/methods/(?P<Method[^/]+>)$",
			},
		},
		"AWS/AppSync": {
			ResourceFilters: []string{"appsync"},
			DimensionRegexps: []string{
				"apis/(?P<GraphQLAPIId>[^/]+)",
			},
		},
		"AWS/AutoScaling": {
			DimensionRegexps: []string{
				"autoScalingGroupName/(?P<AutoScalingGroupName>[^/]+)",
			},
			ResourceDiscovery: autoscaling.getAutoscalingGroups
			
		},
		"AWS/Billing": {},
		"AWS/CloudFront": {
			ResourceFilterss: []string{
				"cloudfront:distribution",
			},
			DimensionRegexps: []string{
				"distribution/(?P<DistributionId>[^/]+)",
			},
		},
		"AWS/Cognito": {
			ResourceFilters: []string{"cognito-idp:userpool"},
			DimensionRegexps: []string{
				"userpool/(?P<UserPool>[^/]+)",
			},
		},
		"AWS/DocDB": {
			ResourceFilters: []string{"rds:db", "rds:cluster"},
			DimensionRegexps: []string{
				"cluster:(?P<DBClusterIdentifier>[^/]+)",
				"db:(?P<DBInstanceIdentifier>[^/]+)",
			},
		},
		"AWS/DynamoDB": {
			ResourceFilters: []string{"dynamodb:table"},
			DimensionRegexps: []string{
				":table/(?P<TableName>[^/]+)",
			},
		},
		"AWS/EBS": {
			ResourceFilters: []string{"ec2:volume"},
			DimensionRegexps: []string{
				"volume/(?P<VolumeId>[^/]+)",
			},
		},
		"AWS/ElastiCache": {
			ResourceFilters: []string{"elasticache:cluster"},
			DimensionRegexps: []string{
				"cluster:(?P<CacheClusterId>[^/]+)",
			},
		},
		"AWS/EC2": {
			ResourceFilters: []string{"ec2:instance"},
			DimensionRegexps: []string{
				"instance/(?P<InstanceId>[^/]+)",
			},
		},
		"AWS/EC2Spot": {
			ResourceDiscovery: ec2spot.getSpotFleetRequests,
			DimensionRegexps: []string{
				"(?P<FleetRequestId>.*)",
			},
		},
		"AWS/ECS": {
			ResourceFilters: []string{
				"ecs:cluster", 
				"ecs:service",
			},
			DImensionRegexps: []string{
				"cluster/(?P<ClusterName>[^/]+)",
				"service/(?P<ClusterName>[^/]+)/([^/]+)",
			},
		},
		"ECS/ContainerInsights": {
			ResourceFilters: []string{
				"ecs:cluster", 
				"ecs:service",
			},
			DImensionRegexps: []string{
				"cluster/(?P<ClusterName>[^/]+)",
				"service/(?P<ClusterName>[^/]+)/([^/]+)",
			},
		},
		"AWS/EFS": {
			ResourceFilters: []string{
				"elasticfilesystem:file-system",
			},
			DImensionRegexps: []string{
				"file-system/(?P<FileSystemId>[^/]+)",
			},
		},
		"AWS/ELB": {
			ResourceFilters: []string{
				"elasticloadbalancing:loadbalancer",
			},
			DImensionRegexps: []string{
				":loadbalancer/(?P<LoadBalancer>.+)$",
			},
		},
		"AWS/ElasticMapReduce": {
			ResourceFilters: []string{
				"elasticmapreduce:cluster",
			},
			DImensionRegexps: []string{
				"cluster/(?P<JobFlowId>[^/]+)",
			},
		},
		"AWS/ES": {
			ResourceFilters: []string{
				"es:domain",
			},
			DImensionRegexps: []string{
				":domain/(?P<DomainName>[^/]+)",
			},
		},
		"AWS/Firehose": {
			ResourceFilters: []string{
				"firehose",
			},
			DImensionRegexps: []string{
				":deliverystream/(?P<DeliveryStreamName>[^/]+)",
			},
		},
		"AWS/FSx": {
			ResourceFilters: []string{
				"fsx:file-system",
			},
			DImensionRegexps: []string{
				"file-system/(?P<FileSystemId>[^/]+)",
			},
		},
		"AWS/GameLift": {
			ResourceFilters: []string{
				"gamelift",
			},
			DImensionRegexps: []string{
				":fleet/(?P<FleetId>[^/]+)",
			},
		},
		"Glue": {
			ResourceFilters: []string{
				"glue:job"
			},
			DImensionRegexps: []string{
				":job/(?P<JobName>[^/]+)",
			},
		},
		"AWS/IoT": {
			ResourceFilters: []string{},
		},
		"AWS/Kafka": {
			ResourceFilters: []string{
				"kafka:cluster",
			},
			DImensionRegexps: []string{
				":cluster/(?P<Cluster Name>[^/]+)",
			},
		},
		"AWS/Kinesis": {
			ResourceFilters: []string{
				"kinesis:stream",
			},
			DImensionRegexps: []string{
				":stream/(?P<StreamName>[^/]+)",
			},
		},
		"AWS/Lambda": {
			ResourceFilters: []string{
				"lambda:function",
			},
			DImensionRegexps: []string{
				":function:(?P<FunctionName>[^/]+)",
			},
		},
		"AWS/NATGateway": {
			ResourceFilters: []string{
				"ec2:natgateway",
			},
			DImensionRegexps: []string{
				"natgateway/(?P<NatGatewayId>[^/]+)",
			},
		},
		"AWS/NetworkELB": {
			ResourceFilters: []string{
				"elasticloadbalancing:loadbalancer/net", 
				"elasticloadbalancing:targetgroup",
			},
			DImensionRegexps: []string{
				":(?P<TargetGroup>targetgroup/.+)",
				":loadbalancer/(?P<LoadBalancer>.+)$",
			},
		},
		"AWS/RDS": {
			ResourceFilters: []string{
				"rds:db", 
				"rds:cluster",
			},
			DImensionRegexps: []string{
				":cluster:(?P<DBClusterIdentifier>[^/]+)",
				":db:(?P<DBInstanceIdentifier>[^/]+)",
			},
		},
		"AWS/Redshift": {
			ResourceFilters: []string{
				"redshift:cluster",
			},
			DImensionRegexps: []string{
				":cluster:(?P<ClusterIdentifier>[^/]+)",
			},
		},
		"AWS/Route53Resolver": {
			ResourceFilters: []string{
				"route53resolver",
			},
			DImensionRegexps: []string{
				":resolver-endpoint/(?P<EndpointId>[^/]+)",
			},
		},
		"AWS/S3": {
			ResourceFilters: []string{
				"s3",
			},
			DImensionRegexps: []string{
				"(?P<BucketName>[^:]+)$",
			},
		},
		"AWS/SES": {},
		"AWS/States": {
			ResourceFilters: []string{
				"states",
			},
			DImensionRegexps: []string{
				"(?P<StateMachineArn>.*)",
			},
		},
		"AWS/SNS": {
			ResourceFilters: []string{
				"sns",
			},
			DImensionRegexps: []string{
				"(?P<TopicName>[^:]+)$",
			},
		},
		"AWS/SQS": {
			ResourceFilters: []string{
				"sqs",
			},
			DImensionRegexps: []string{
				"(?P<QueueName>[^:]+)$",
			},
		},
		"AWS/TransitGateway": {
			ResourceFilters: []string{
				"ec2:transit-gateway",
			},
			DImensionRegexps: []string{
				":transit-gateway/(?P<TransitGateway>[^/]+)",
			},
		},
		"AWS/VPN": {
			ResourceFilters: []string{
				"ec2:vpn-connection",
			},
			DImensionRegexps: []string{
				":vpn-connection/(?P<VpnId>[^/]+)",
			},
		},
		"AWS/WAFV2": {
			ResourceFilters: []string{
				"wafv2",
			},
			DImensionRegexps: []string{
				"/webacl/(?P<WebACL>[^/]+)",
			},
		},
	}
)