package main

import (
	"github.com/aws/aws-sdk-go/aws"
)

type serviceFilter struct {
	Alias            string
	ResourceFilters  []*string
	DimensionRegexps []*string
}

type serviceConfig map[string]serviceFilter

func (sm serviceConfig) getService(serviceType string) *serviceFilter {
	if _, ok := sm[serviceType]; !ok {
		for _, sc := range sm {
			if sc.Alias == serviceType {
				return &sc
			}
		}
	}
	return nil
}

var (
	supportedServices = serviceConfig{
		"AWS/ApplicationELB": {
			Alias: "elb-application",
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
			Alias: "apigateway",
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
			Alias: "appsync",
			ResourceFilters: []*string{
				aws.String("appsync"),
			},
			DimensionRegexps: []*string{
				aws.String("apis/(?P<GraphQLAPIId>[^/]+)"),
			},
		},
		"AWS/AutoScaling": {
			Alias: "autoscaling",
			DimensionRegexps: []*string{
				aws.String("autoScalingGroupName/(?P<AutoScalingGroupName>[^/]+)"),
			},
		},
		"AWS/Billing": {
			Alias: "billing",
		},
		"AWS/CloudFront": {
			Alias: "cloudfront",
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
			Alias: "docdb",
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
			Alias: "dynamodb",
			ResourceFilters: []*string{
				aws.String("dynamodb:table"),
			},
			DimensionRegexps: []*string{
				aws.String(":table/(?P<TableName>[^/]+)"),
			},
		},
		"AWS/EBS": {
			Alias: "ebs",
			ResourceFilters: []*string{
				aws.String("ec2:volume"),
			},
			DimensionRegexps: []*string{
				aws.String("volume/(?P<VolumeId>[^/]+)"),
			},
		},
		"AWS/ElastiCache": {
			Alias: "elasticache",
			ResourceFilters: []*string{
				aws.String("elasticache:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster:(?P<CacheClusterId>[^/]+)"),
			},
		},
		"AWS/EC2": {
			Alias: "ec2",
			ResourceFilters: []*string{
				aws.String("ec2:instance"),
			},
			DimensionRegexps: []*string{
				aws.String("instance/(?P<InstanceId>[^/]+)"),
			},
		},
		"AWS/EC2Spot": {
			Alias: "ec2-spot",
			DimensionRegexps: []*string{
				aws.String("(?P<FleetRequestId>.*)"),
			},
		},
		"AWS/ECS": {
			Alias: "ecs",
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
			Alias: "ecs-containerinsights",
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
			Alias: "efs",
			ResourceFilters: []*string{
				aws.String("elasticfilesystem:file-system"),
			},
			DimensionRegexps: []*string{
				aws.String("file-system/(?P<FileSystemId>[^/]+)"),
			},
		},
		"AWS/ELB": {
			Alias: "elb",
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer"),
			},
			DimensionRegexps: []*string{
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		},
		"AWS/ElasticMapReduce": {
			Alias: "elasticmapreduce",
			ResourceFilters: []*string{
				aws.String("elasticmapreduce:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<JobFlowId>[^/]+)"),
			},
		},
		"AWS/ES": {
			Alias: "elasticsearch",
			ResourceFilters: []*string{
				aws.String("es:domain"),
			},
			DimensionRegexps: []*string{
				aws.String(":domain/(?P<DomainName>[^/]+)"),
			},
		},
		"AWS/Firehose": {
			Alias: "firehose",
			ResourceFilters: []*string{
				aws.String("firehose"),
			},
			DimensionRegexps: []*string{
				aws.String(":deliverystream/(?P<DeliveryStreamName>[^/]+)"),
			},
		},
		"AWS/FSx": {
			Alias: "fsx",
			ResourceFilters: []*string{
				aws.String("fsx:file-system"),
			},
			DimensionRegexps: []*string{
				aws.String("file-system/(?P<FileSystemId>[^/]+)"),
			},
		},
		"AWS/GameLift": {
			Alias: "gamelift",
			ResourceFilters: []*string{
				aws.String("gamelift"),
			},
			DimensionRegexps: []*string{
				aws.String(":fleet/(?P<FleetId>[^/]+)"),
			},
		},
		"Glue": {
			Alias: "glue",
			ResourceFilters: []*string{
				aws.String("glue:job"),
			},
			DimensionRegexps: []*string{
				aws.String(":job/(?P<JobName>[^/]+)"),
			},
		},
		"AWS/IoT": {
			Alias: "iot",
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
			Alias: "kafka",
			ResourceFilters: []*string{
				aws.String("kafka:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster/(?P<Cluster_Name>[^/]+)"),
			},
		},
		"AWS/Kinesis": {
			Alias: "kinesis",
			ResourceFilters: []*string{
				aws.String("kinesis:stream"),
			},
			DimensionRegexps: []*string{
				aws.String(":stream/(?P<StreamName>[^/]+)"),
			},
		},
		"AWS/Lambda": {
			Alias: "lambda",
			ResourceFilters: []*string{
				aws.String("lambda:function"),
			},
			DimensionRegexps: []*string{
				aws.String(":function:(?P<FunctionName>[^/]+)"),
			},
		},
		"AWS/NATGateway": {
			Alias: "natgateway",
			ResourceFilters: []*string{
				aws.String("ec2:natgateway"),
			},
			DimensionRegexps: []*string{
				aws.String("natgateway/(?P<NatGatewayId>[^/]+)"),
			},
		},
		"AWS/NetworkELB": {
			Alias: "elb-network",
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
			Alias: "rds",
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
			Alias: "redshift",
			ResourceFilters: []*string{
				aws.String("redshift:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster:(?P<ClusterIdentifier>[^/]+)"),
			},
		},
		"AWS/Route53Resolver": {
			Alias: "route53-resolver",
			ResourceFilters: []*string{
				aws.String("route53resolver"),
			},
			DimensionRegexps: []*string{
				aws.String(":resolver-endpoint/(?P<EndpointId>[^/]+)"),
			},
		},
		"AWS/S3": {
			Alias: "s3",
			ResourceFilters: []*string{
				aws.String("s3"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<BucketName>[^:]+)$"),
			},
		},
		"AWS/SES": {
			Alias: "ses",
		},
		"AWS/States": {
			Alias: "stepfunctions",
			ResourceFilters: []*string{
				aws.String("states"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<StateMachineArn>.*)"),
			},
		},
		"AWS/SNS": {
			Alias: "sns",
			ResourceFilters: []*string{
				aws.String("sns"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<TopicName>[^:]+)$"),
			},
		},
		"AWS/SQS": {
			Alias: "sqs",
			ResourceFilters: []*string{
				aws.String("sqs"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<QueueName>[^:]+)$"),
			},
		},
		"AWS/TransitGateway": {
			Alias: "transitgateway",
			ResourceFilters: []*string{
				aws.String("ec2:transit-gateway"),
			},
			DimensionRegexps: []*string{
				aws.String(":transit-gateway/(?P<TransitGateway>[^/]+)"),
				aws.String(":transit-gateway-attachment/(?P<TransitGateway>[^/]+)/(?P<TransitGatewayAttachment>[^/]+)"),
			},
		},
		"AWS/VPN": {
			Alias: "vpn",
			ResourceFilters: []*string{
				aws.String("ec2:vpn-connection"),
			},
			DimensionRegexps: []*string{
				aws.String(":vpn-connection/(?P<VpnId>[^/]+)"),
			},
		},
		"AWS/WAFV2": {
			Alias: "waf",
			ResourceFilters: []*string{
				aws.String("wafv2"),
			},
			DimensionRegexps: []*string{
				aws.String("/webacl/(?P<WebACL>[^/]+)"),
			},
		},
	}
)
