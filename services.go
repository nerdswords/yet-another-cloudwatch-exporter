package main

import (
	"github.com/aws/aws-sdk-go/aws"
)

type serviceFilter struct {
	Namespace        string
	Alias            string
	ResourceFilters  []*string
	DimensionRegexps []*string
}

type serviceConfig []serviceFilter

func (sc serviceConfig) getService(serviceType string) *serviceFilter {
	for _, sf := range sc {
		if sf.Alias == serviceType || sf.Namespace == serviceType {
			return &sf
		}
	}
	return nil
}

var (
	supportedServices = serviceConfig{
		{
			Namespace: "AWS/ApplicationELB",
			Alias:     "elb-application",
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer/app"),
				aws.String("elasticloadbalancing:targetgroup"),
			},
			DimensionRegexps: []*string{
				aws.String(":(?P<TargetGroup>targetgroup/.+)"),
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		}, {
			Namespace: "AWS/ApiGateway",
			Alias:     "apigateway",
			ResourceFilters: []*string{
				aws.String("apigateway"),
			},
			DimensionRegexps: []*string{
				aws.String("apis/(?P<ApiName>[^/]+)$"),
				aws.String("apis/(?P<ApiName>[^/]+)/stages/(?P<Stage>[^/]+)$"),
				aws.String("apis/(?P<ApiName>[^/]+)/resources/(?P<Resource>[^/]+)$"),
				aws.String("apis/(?P<ApiName>[^/]+)/resources/(?P<Resource>[^/]+)/methods/(?P<Method>[^/]+)$"),
			},
		}, {
			Namespace: "AWS/AppSync",
			Alias:     "appsync",
			ResourceFilters: []*string{
				aws.String("appsync"),
			},
			DimensionRegexps: []*string{
				aws.String("apis/(?P<GraphQLAPIId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/AutoScaling",
			Alias:     "autoscaling",
			DimensionRegexps: []*string{
				aws.String("autoScalingGroupName/(?P<AutoScalingGroupName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Billing",
			Alias:     "billing",
		}, {
			Namespace: "AWS/CloudFront",
			Alias:     "cloudfront",
			ResourceFilters: []*string{
				aws.String("cloudfront:distribution"),
			},
			DimensionRegexps: []*string{
				aws.String("distribution/(?P<DistributionId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Cognito",
			ResourceFilters: []*string{
				aws.String("cognito-idp:userpool"),
			},
			DimensionRegexps: []*string{
				aws.String("userpool/(?P<UserPool>[^/]+)"),
			},
		}, {
			Namespace: "AWS/DocDB",
			Alias:     "docdb",
			ResourceFilters: []*string{
				aws.String("rds:db"),
				aws.String("rds:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster:(?P<DBClusterIdentifier>[^/]+)"),
				aws.String("db:(?P<DBInstanceIdentifier>[^/]+)"),
			},
		}, {
			Namespace: "AWS/DynamoDB",
			Alias:     "dynamodb",
			ResourceFilters: []*string{
				aws.String("dynamodb:table"),
			},
			DimensionRegexps: []*string{
				aws.String(":table/(?P<TableName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/EBS",
			Alias:     "ebs",
			ResourceFilters: []*string{
				aws.String("ec2:volume"),
			},
			DimensionRegexps: []*string{
				aws.String("volume/(?P<VolumeId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/ElastiCache",
			Alias:     "elasticache",
			ResourceFilters: []*string{
				aws.String("elasticache:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster:(?P<CacheClusterId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/EC2",
			Alias:     "ec2",
			ResourceFilters: []*string{
				aws.String("ec2:instance"),
			},
			DimensionRegexps: []*string{
				aws.String("instance/(?P<InstanceId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/EC2Spot",
			Alias:     "ec2-spot",
			DimensionRegexps: []*string{
				aws.String("(?P<FleetRequestId>.*)"),
			},
		}, {
			Namespace: "AWS/ECS",
			Alias:     "ecs",
			ResourceFilters: []*string{
				aws.String("ecs:cluster"),
				aws.String("ecs:service"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<ClusterName>[^/]+)"),
				aws.String("service/(?P<ClusterName>[^/]+)/([^/]+)"),
			},
		}, {
			Namespace: "ECS/ContainerInsights",
			Alias:     "ecs-containerinsights",
			ResourceFilters: []*string{
				aws.String("ecs:cluster"),
				aws.String("ecs:service"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<ClusterName>[^/]+)"),
				aws.String("service/(?P<ClusterName>[^/]+)/([^/]+)"),
			},
		}, {
			Namespace: "AWS/EFS",
			Alias:     "efs",
			ResourceFilters: []*string{
				aws.String("elasticfilesystem:file-system"),
			},
			DimensionRegexps: []*string{
				aws.String("file-system/(?P<FileSystemId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/ELB",
			Alias:     "elb",
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer"),
			},
			DimensionRegexps: []*string{
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		}, {
			Namespace: "AWS/ElasticMapReduce",
			Alias:     "elasticmapreduce",
			ResourceFilters: []*string{
				aws.String("elasticmapreduce:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<JobFlowId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/ES",
			Alias:     "elasticsearch",
			ResourceFilters: []*string{
				aws.String("es:domain"),
			},
			DimensionRegexps: []*string{
				aws.String(":domain/(?P<DomainName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Firehose",
			Alias:     "firehose",
			ResourceFilters: []*string{
				aws.String("firehose"),
			},
			DimensionRegexps: []*string{
				aws.String(":deliverystream/(?P<DeliveryStreamName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/FSx",
			Alias:     "fsx",
			ResourceFilters: []*string{
				aws.String("fsx:file-system"),
			},
			DimensionRegexps: []*string{
				aws.String("file-system/(?P<FileSystemId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/GameLift",
			Alias:     "gamelift",
			ResourceFilters: []*string{
				aws.String("gamelift"),
			},
			DimensionRegexps: []*string{
				aws.String(":fleet/(?P<FleetId>[^/]+)"),
			},
		}, {
			Namespace: "Glue",
			Alias:     "glue",
			ResourceFilters: []*string{
				aws.String("glue:job"),
			},
			DimensionRegexps: []*string{
				aws.String(":job/(?P<JobName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/IoT",
			Alias:     "iot",
			ResourceFilters: []*string{
				aws.String("iot:rule"),
				aws.String("iot:provisioningtemplate"),
			},
			DimensionRegexps: []*string{
				aws.String(":rule/(?P<RuleName>[^/]+)"),
				aws.String(":provisioningtemplate/(?P<TemplateName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Kafka",
			Alias:     "kafka",
			ResourceFilters: []*string{
				aws.String("kafka:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster/(?P<Cluster_Name>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Kinesis",
			Alias:     "kinesis",
			ResourceFilters: []*string{
				aws.String("kinesis:stream"),
			},
			DimensionRegexps: []*string{
				aws.String(":stream/(?P<StreamName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Lambda",
			Alias:     "lambda",
			ResourceFilters: []*string{
				aws.String("lambda:function"),
			},
			DimensionRegexps: []*string{
				aws.String(":function:(?P<FunctionName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/NATGateway",
			Alias:     "natgateway",
			ResourceFilters: []*string{
				aws.String("ec2:natgateway"),
			},
			DimensionRegexps: []*string{
				aws.String("natgateway/(?P<NatGatewayId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/NetworkELB",
			Alias:     "elb-network",
			ResourceFilters: []*string{
				aws.String("elasticloadbalancing:loadbalancer/net"),
				aws.String("elasticloadbalancing:targetgroup"),
			},
			DimensionRegexps: []*string{
				aws.String(":(?P<TargetGroup>targetgroup/.+)"),
				aws.String(":loadbalancer/(?P<LoadBalancer>.+)$"),
			},
		}, {
			Namespace: "AWS/RDS",
			Alias:     "rds",
			ResourceFilters: []*string{
				aws.String("rds:db"),
				aws.String("rds:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster:(?P<DBClusterIdentifier>[^/]+)"),
				aws.String(":db:(?P<DBInstanceIdentifier>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Redshift",
			Alias:     "redshift",
			ResourceFilters: []*string{
				aws.String("redshift:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster:(?P<ClusterIdentifier>[^/]+)"),
			},
		}, {
			Namespace: "AWS/Route53Resolver",
			Alias:     "route53-resolver",
			ResourceFilters: []*string{
				aws.String("route53resolver"),
			},
			DimensionRegexps: []*string{
				aws.String(":resolver-endpoint/(?P<EndpointId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/S3",
			Alias:     "s3",
			ResourceFilters: []*string{
				aws.String("s3"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<BucketName>[^:]+)$"),
			},
		}, {
			Namespace: "AWS/SES",
			Alias:     "ses",
		}, {
			Namespace: "AWS/States",
			Alias:     "stepfunctions",
			ResourceFilters: []*string{
				aws.String("states"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<StateMachineArn>.*)"),
			},
		}, {
			Namespace: "AWS/SNS",
			Alias:     "sns",
			ResourceFilters: []*string{
				aws.String("sns"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<TopicName>[^:]+)$"),
			},
		}, {
			Namespace: "AWS/SQS",
			Alias:     "sqs",
			ResourceFilters: []*string{
				aws.String("sqs"),
			},
			DimensionRegexps: []*string{
				aws.String("(?P<QueueName>[^:]+)$"),
			},
		}, {
			Namespace: "AWS/TransitGateway",
			Alias:     "transitgateway",
			ResourceFilters: []*string{
				aws.String("ec2:transit-gateway"),
			},
			DimensionRegexps: []*string{
				aws.String(":transit-gateway/(?P<TransitGateway>[^/]+)"),
				aws.String(":transit-gateway-attachment/(?P<TransitGateway>[^/]+)/(?P<TransitGatewayAttachment>[^/]+)"),
			},
		}, {
			Namespace: "AWS/VPN",
			Alias:     "vpn",
			ResourceFilters: []*string{
				aws.String("ec2:vpn-connection"),
			},
			DimensionRegexps: []*string{
				aws.String(":vpn-connection/(?P<VpnId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/WAFV2",
			Alias:     "waf",
			ResourceFilters: []*string{
				aws.String("wafv2"),
			},
			DimensionRegexps: []*string{
				aws.String("/webacl/(?P<WebACL>[^/]+)"),
			},
		},
	}
)
