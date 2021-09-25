package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ResourceFunc func(tagsInterface, *Job, string) ([]*taggedResource, error)

type FilterFunc func(tagsInterface, []*taggedResource) ([]*taggedResource, error)

type serviceFilter struct {
	Namespace        string
	Alias            string
	IgnoreLength     bool
	ResourceFilters  []*string
	DimensionRegexps []*string
	ResourceFunc     ResourceFunc
	FilterFunc       FilterFunc
}

type serviceConfig []serviceFilter

func (sc serviceConfig) GetService(serviceType string) *serviceFilter {
	for _, sf := range sc {
		if sf.Alias == serviceType || sf.Namespace == serviceType {
			return &sf
		}
	}
	return nil
}

var (
	SupportedServices = serviceConfig{
		{
			Namespace: "AWS/CertificateManager",
			Alias:     "acm",
			ResourceFilters: []*string{
				aws.String("acm:certificate"),
			},
		},
		{
			Namespace: "AWS/ApplicationELB",
			Alias:     "alb",
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
			FilterFunc: func(iface tagsInterface, inputResources []*taggedResource) (outputResources []*taggedResource, err error) {
				ctx := context.Background()
				apiGatewayAPICounter.Inc()
				var limit int64 = 500 // max number of results per page. default=25, max=500
				const maxPages = 10
				input := apigateway.GetRestApisInput{Limit: &limit}
				output := apigateway.GetRestApisOutput{}
				var pageNum int
				err = iface.apiGatewayClient.GetRestApisPagesWithContext(ctx, &input, func(page *apigateway.GetRestApisOutput, lastPage bool) bool {
					pageNum++
					output.Items = append(output.Items, page.Items...)
					return pageNum <= maxPages
				})
				for _, resource := range inputResources {
					for i, gw := range output.Items {
						if strings.Contains(resource.ARN, *gw.Id) {
							r := resource
							r.ARN = strings.ReplaceAll(resource.ARN, *gw.Id, *gw.Name)
							outputResources = append(outputResources, r)
							output.Items = append(output.Items[:i], output.Items[i+1:]...)
							break
						}
					}
				}
				return outputResources, err
			},
		}, {
			Namespace: "AWS/AmazonMQ",
			Alias:     "mq",
			ResourceFilters: []*string{
				aws.String("mq"),
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
			Namespace: "AWS/Athena",
			Alias:     "athena",
			ResourceFilters: []*string{
				aws.String("athena"),
			},
			DimensionRegexps: []*string{
				aws.String("athena/(?P<WorkGroup>[^/]+)"),
			},
		},
		{
			Namespace: "AWS/AutoScaling",
			Alias:     "asg",
			DimensionRegexps: []*string{
				aws.String("autoScalingGroupName/(?P<AutoScalingGroupName>[^/]+)"),
			},
			ResourceFunc: func(iface tagsInterface, job *Job, region string) (resources []*taggedResource, err error) {
				ctx := context.Background()
				pageNum := 0
				return resources, iface.asgClient.DescribeAutoScalingGroupsPagesWithContext(ctx, &autoscaling.DescribeAutoScalingGroupsInput{},
					func(page *autoscaling.DescribeAutoScalingGroupsOutput, more bool) bool {
						pageNum++
						autoScalingAPICounter.Inc()

						for _, asg := range page.AutoScalingGroups {
							resource := taggedResource{
								ARN:       aws.StringValue(asg.AutoScalingGroupARN),
								Namespace: job.Type,
								Region:    region,
							}

							for _, t := range asg.Tags {
								resource.Tags = append(resource.Tags, Tag{Key: *t.Key, Value: *t.Value})
							}

							if resource.filterThroughTags(job.SearchTags) {
								resources = append(resources, &resource)
							}
						}
						return pageNum < 100
					},
				)
			},
		}, {
			Namespace: "AWS/ElasticBeanstalk",
			Alias:     "beanstalk",
		},
		{
			Namespace:    "AWS/Billing",
			Alias:        "billing",
			IgnoreLength: true,
		}, {
			Namespace: "AWS/Cassandra",
			Alias:     "cassandra",
			ResourceFilters: []*string{
				aws.String("cassandra"),
			},
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
			Alias:     "cognito-idp",
			ResourceFilters: []*string{
				aws.String("cognito-idp:userpool"),
			},
			DimensionRegexps: []*string{
				aws.String("userpool/(?P<UserPool>[^/]+)"),
			},
		}, {
			Namespace: "AWS/DMS",
			Alias:     "dms",
			ResourceFilters: []*string{
				aws.String("dms"),
			},
			DimensionRegexps: []*string{
				aws.String("rep:(?P<ReplicationInstanceIdentifier>[^/]+)"),
			},
		}, {
			Namespace: "AWS/DDoSProtection",
			Alias:     "shield",
			ResourceFilters: []*string{
				aws.String("shield:protection"),
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
			Alias:     "ec",
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
			Alias:     "ec2Spot",
			DimensionRegexps: []*string{
				aws.String("(?P<FleetRequestId>.*)"),
			},
			ResourceFunc: func(iface tagsInterface, job *Job, region string) (resources []*taggedResource, err error) {
				ctx := context.Background()
				pageNum := 0
				return resources, iface.ec2Client.DescribeSpotFleetRequestsPagesWithContext(ctx, &ec2.DescribeSpotFleetRequestsInput{},
					func(page *ec2.DescribeSpotFleetRequestsOutput, more bool) bool {
						pageNum++
						ec2APICounter.Inc()

						for _, ec2Spot := range page.SpotFleetRequestConfigs {
							resource := taggedResource{
								ARN:       aws.StringValue(ec2Spot.SpotFleetRequestId),
								Namespace: job.Type,
								Region:    region,
							}

							for _, t := range ec2Spot.Tags {
								resource.Tags = append(resource.Tags, Tag{Key: *t.Key, Value: *t.Value})
							}

							if resource.filterThroughTags(job.SearchTags) {
								resources = append(resources, &resource)
							}
						}
						return pageNum < 100
					},
				)
			},
		}, {
			Namespace: "AWS/ECS",
			Alias:     "ecs-svc",
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
				aws.String(":loadbalancer/(?P<LoadBalancerName>.+)$"),
			},
		}, {
			Namespace: "AWS/ElasticMapReduce",
			Alias:     "emr",
			ResourceFilters: []*string{
				aws.String("elasticmapreduce:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String("cluster/(?P<JobFlowId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/ES",
			Alias:     "es",
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
			Namespace: "AWS/Neptune",
			Alias:     "neptune",
			ResourceFilters: []*string{
				aws.String("rds:db"),
				aws.String("rds:cluster"),
			},
			DimensionRegexps: []*string{
				aws.String(":cluster:(?P<DBClusterIdentifier>[^/]+)"),
				aws.String(":db:(?P<DBInstanceIdentifier>[^/]+)"),
			},
		}, {
			Namespace: "AWS/NetworkFirewall",
			Alias:     "nfw",
			ResourceFilters: []*string{
				aws.String("network-firewall:firewall"),
			},
			DimensionRegexps: []*string{
				aws.String("firewall/(?P<FirewallName>[^/]+)"),
			},
		}, {
			Namespace: "AWS/NATGateway",
			Alias:     "ngw",
			ResourceFilters: []*string{
				aws.String("ec2:natgateway"),
			},
			DimensionRegexps: []*string{
				aws.String("natgateway/(?P<NatGatewayId>[^/]+)"),
			},
		}, {
			Namespace: "AWS/NetworkELB",
			Alias:     "nlb",
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
			Alias:     "r53r",
			ResourceFilters: []*string{
				aws.String("route53resolver"),
			},
			DimensionRegexps: []*string{
				aws.String(":resolver-endpoint/(?P<EndpointId>[^/]+)"),
			},
		}, {
			Namespace:    "AWS/S3",
			Alias:        "s3",
			IgnoreLength: true,
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
			Alias:     "sfn",
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
			Alias:     "tgw",
			ResourceFilters: []*string{
				aws.String("ec2:transit-gateway"),
			},
			DimensionRegexps: []*string{
				aws.String(":transit-gateway/(?P<TransitGateway>[^/]+)"),
				aws.String("(?P<TransitGateway>[^/]+)/(?P<TransitGatewayAttachment>[^/]+)"),
			},
			ResourceFunc: func(iface tagsInterface, job *Job, region string) (resources []*taggedResource, err error) {
				ctx := context.Background()
				pageNum := 0
				return resources, iface.ec2Client.DescribeTransitGatewayAttachmentsPagesWithContext(ctx, &ec2.DescribeTransitGatewayAttachmentsInput{},
					func(page *ec2.DescribeTransitGatewayAttachmentsOutput, more bool) bool {
						pageNum++
						ec2APICounter.Inc()

						for _, tgwa := range page.TransitGatewayAttachments {
							resource := taggedResource{
								ARN:       fmt.Sprintf("%s/%s", *tgwa.TransitGatewayId, *tgwa.TransitGatewayAttachmentId),
								Namespace: job.Type,
								Region:    region,
							}

							for _, t := range tgwa.Tags {
								resource.Tags = append(resource.Tags, Tag{Key: *t.Key, Value: *t.Value})
							}

							if resource.filterThroughTags(job.SearchTags) {
								resources = append(resources, &resource)
							}
						}
						return pageNum < 100
					},
				)
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
			Alias:     "wafv2",
			ResourceFilters: []*string{
				aws.String("wafv2"),
			},
			DimensionRegexps: []*string{
				aws.String("/webacl/(?P<WebACL>[^/]+)"),
			},
		}, {
			Namespace: "AWS/WorkSpaces",
			Alias:     "workspaces",
			ResourceFilters: []*string{
				aws.String("workspaces:workspace"),
				aws.String("workspaces:directory"),
			},
			DimensionRegexps: []*string{
				aws.String(":workspace/(?P<WorkspaceId>[^/]+)$"),
				aws.String(":directory/(?P<DirectoryId>[^/]+)$"),
			},
		},
	}
)
