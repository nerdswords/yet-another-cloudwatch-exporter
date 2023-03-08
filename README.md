# YACE - yet another cloudwatch exporter

YACE, or `yet another cloudwatch exporter`, is a [Prometheus exporter](https://prometheus.io/docs/instrumenting/exporters/#exporters-and-integrations) for [AWS CloudWatch](http://aws.amazon.com/cloudwatch/) metrics. It is written in Go and uses the official AWS SDK.

The project was originally created by Thomas Peitz while working at InVision.de, then later moved outside of the company repo. Read the full rebranding story [here](https://medium.com/@IT_Supertramp/reorganizing-yace-79d7149b9584).

## Alternatives

Consider using the official [CloudWatch Exporter](https://github.com/prometheus/cloudwatch_exporter) if you prefer a Java implementation.


## Project Status

While YACE is at version less than 1.0.0, expect that any new release might introduce breaking changes. We'll document changes in [CHANGELOG.md](CHANGELOG.md).

Where feasible, features will be deprecated instead of being immediately changed or removed. This means that YACE will continue to work but might log warning messages. Expect deprecated features to be permanently changed/removed within the next 2/3 releases.

## Security

Read more how to report a security vulnerability in [SECURITY.md](SECURITY.md).

### Supported Versions

Only the latest version gets security updates. We won't support older versions.

## Features

* Stop worrying about your AWS IDs - Auto discovery of resources via tags
* Structured JSON logging
* Filter monitored resources via regex
* Automatic adding of tag labels to metrics
* Automatic adding of dimension labels to metrics
* Allows to export 0 even if CloudWatch returns nil
* Allows exports metrics with CloudWatch timestamps (disabled by default)
* Static metrics support for all cloudwatch metrics without auto discovery
* Pull data from multiple AWS accounts using cross-account roles
* Can be used as a library in an external application
* Support the scraping of custom namespaces metrics with the CloudWatch Dimensions.
* Supported services with auto discovery through tags:

  * acm (AWS/CertificateManager) - Certificate Manager
  * acm-pca (AWS/ACMPrivateCA) - ACM Private CA
  * airflow (AmazonMWAA) - Managed Apache Airflow
  * mwaa (AWS/MWAA) - Managed Apache Airflow (Container, queue, and database metrics)
  * alb (AWS/ApplicationELB) - Application Load Balancer
  * apigateway (AWS/ApiGateway) - API Gateway
  * appstream (AWS/AppStream) - AppStream
  * appsync (AWS/AppSync) - AppSync
  * amp (AWS/Prometheus) - Managed Service for Prometheus
  * aoss (AWS/AOSS) - OpenSearch Serverless
  * athena (AWS/Athena) - Athena
  * backup (AWS/Backup) - Backup
  * beanstalk (AWS/ElasticBeanstalk) - Elastic Beanstalk
  * billing (AWS/Billing) - Billing
  * cassandra (AWS/Cassandra) - Cassandra
  * cloudfront (AWS/CloudFront) - Cloud Front
  * cognito-idp (AWS/Cognito) - Cognito
  * dms (AWS/DMS) - Database Migration Service
  * docdb (AWS/DocDB) - DocumentDB (with MongoDB compatibility)
  * dx (AWS/DX) - Direct Connect
  * dynamodb (AWS/DynamoDB) - NoSQL Key-Value Database
  * ebs (AWS/EBS) - Elastic Block Storage
  * ec (AWS/Elasticache) - ElastiCache
  * ec2 (AWS/EC2) - Elastic Compute Cloud
  * ec2Spot (AWS/EC2Spot) - Elastic Compute Cloud for Spot Instances
  * ecs-svc (AWS/ECS) - Elastic Container Service (Service Metrics)
  * ecs-containerinsights (ECS/ContainerInsights) - ECS/ContainerInsights (Fargate metrics)
  * efs (AWS/EFS) - Elastic File System
  * elb (AWS/ELB) - Elastic Load Balancer
  * emr (AWS/ElasticMapReduce) - Elastic MapReduce
  * emr-serverless (AWS/EMRServerless) - Amazon EMR Serverless
  * es (AWS/ES) - ElasticSearch
  * fsx (AWS/FSx) - FSx File System
  * gamelift (AWS/GameLift) - GameLift
  * ga (AWS/GlobalAccelerator) - AWS Global Accelerator
  * glue (Glue) - AWS Glue Jobs
  * iot (AWS/IoT) - IoT
  * kafkaconnect (AWS/KafkaConnect) - AWS MSK Connectors
  * kinesis (AWS/Kinesis) - Kinesis Data Stream
  * nfw (AWS/NetworkFirewall) - Network Firewall
  * ngw (AWS/NATGateway) - NAT Gateway
  * lambda (AWS/Lambda) - Lambda Functions
  * mediaconnect (AWS/MediaConnect) - AWS Elemental MediaConnect
  * medialive (AWS/MediaLive) - AWS Elemental MediaLive
  * mediatailor (AWS/MediaTailor) - AWS Elemental MediaTailor
  * mq (AWS/AmazonMQ) - Managed Message Broker Service
  * neptune (AWS/Neptune) - Neptune
  * nlb (AWS/NetworkELB) - Network Load Balancer
  * vpc-endpoint (AWS/PrivateLinkEndpoints) - VPC Endpoint
  * vpc-endpoint-service (AWS/PrivateLinkServices) - VPC Endpoint Service
  * redshift (AWS/Redshift) - Redshift Database
  * rds (AWS/RDS) - Relational Database Service
  * route53 (AWS/Route53) - Route53 Health Checks
  * route53-resolver (AWS/Route53Resolver) - Route53 Resolver
  * s3 (AWS/S3) - Object Storage
  * ses (AWS/SES) - Simple Email Service
  * shield (AWS/DDoSProtection) - Distributed Denial of Service (DDoS) protection service
  * sqs (AWS/SQS) - Simple Queue Service
  * storagegateway (AWS/StorageGateway) - On-premises access to cloud storage
  * tgw (AWS/TransitGateway) - Transit Gateway
  * vpn (AWS/VPN) - VPN connection
  * asg (AWS/AutoScaling) - Auto Scaling Group
  * kafka (AWS/Kafka) - Managed Apache Kafka
  * firehose (AWS/Firehose) - Managed Streaming Service
  * sns (AWS/SNS) - Simple Notification Service
  * sfn (AWS/States) - Step Functions
  * wafv2 (AWS/WAFV2) - Web Application Firewall v2
  * workspaces (AWS/WorkSpaces) - Workspaces

## Installing and running

Refer to the [installing guide](docs/install.md).

## Authentication

The exporter will need to be running in an environment which has access to AWS. The exporter uses the [AWS SDK for Go](https://aws.github.io/aws-sdk-go-v2/docs/getting-started/) and supports providing authentication via [AWS's default credential chain](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials). Regardless of the method used to acquire the credentials, some permissions are needed for the exporter to work.

As a quick start, the following IAM policy can be used to grant the permissions for all YACE's features to work. If some of the features are not needed, the policy can be adjusted accordingly, as described in [this section](#required-iam-permissions).

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Stmt1674249227793",
      "Action": [
        "tag:GetResources",
        "cloudwatch:GetMetricData",
        "cloudwatch:GetMetricStatistics",
        "cloudwatch:ListMetrics",
        "ec2:DescribeTags",
        "ec2:DescribeInstances",
        "ec2:DescribeRegions",
        "ec2:DescribeTransitGateway*",
        "apigateway:GET",
        "dms:DescribeReplicationInstances",
        "dms:DescribeReplicationTasks"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

If running YACE inside an AWS EC2 instance, the exporter will automatically attempt to assume the associated IAM Role. If this is undesirable behavior turn off the use the metadata endpoint by setting the environment variable `AWS_EC2_METADATA_DISABLED=true`.

## Configuration

Refer to the [configuration](docs/configuration.md) docs.

## Metrics Examples

```text
### Metrics with exportedTagsOnMetrics
aws_ec2_cpuutilization_maximum{dimension_InstanceId="i-someid", name="arn:aws:ec2:eu-west-1:472724724:instance/i-someid", tag_Name="jenkins"} 57.2916666666667

### Info helper with tags
aws_elb_info{name="arn:aws:elasticloadbalancing:eu-west-1:472724724:loadbalancer/a815b16g3417211e7738a02fcc13bbf9",tag_KubernetesCluster="production-19",tag_Name="",tag_kubernetes_io_cluster_production_19="owned",tag_kubernetes_io_service_name="nginx-ingress/private-ext",region="eu-west-1"} 0
aws_ec2_info{name="arn:aws:ec2:eu-west-1:472724724:instance/i-someid",tag_Name="jenkins"} 0

### Track cloudwatch requests to calculate costs
yace_cloudwatch_requests_total 168
```

## Query Examples without exportedTagsOnMetrics

```text
# CPUUtilization + Name tag of the instance id - No more instance id needed for monitoring
aws_ec2_cpuutilization_average + on (name) group_left(tag_Name) aws_ec2_info

# Free Storage in Megabytes + tag Type of the elasticsearch cluster
(aws_es_free_storage_space_sum + on (name) group_left(tag_Type) aws_es_info) / 1024

# Add kubernetes / kops tags on 4xx elb metrics
(aws_elb_httpcode_backend_4_xx_sum + on (name) group_left(tag_KubernetesCluster,tag_kubernetes_io_service_name) aws_elb_info)

# Availability Metric for ELBs (Successful requests / Total Requests) + k8s service name
# Use nilToZero on all metrics else it won't work
((aws_elb_request_count_sum - on (name) group_left() aws_elb_httpcode_backend_4_xx_sum) - on (name) group_left() aws_elb_httpcode_backend_5_xx_sum) + on (name) group_left(tag_kubernetes_io_service_name) aws_elb_info

# Forecast your elasticsearch disk size in 7 days and report metrics with tags type and version
predict_linear(aws_es_free_storage_space_minimum[2d], 86400 * 7) + on (name) group_left(tag_type, tag_version) aws_es_info

# Forecast your cloudwatch costs for next 32 days based on last 10 minutes
# 1.000.000 Requests free
# 0.01 Dollar for 1.000 GetMetricStatistics Api Requests (https://aws.amazon.com/cloudwatch/pricing/)
((increase(yace_cloudwatch_requests_total[10m]) * 6 * 24 * 32) - 100000) / 1000 * 0.01
```

## Required IAM permissions

The following IAM permissions are required for YACE to work.

```json
"tag:GetResources",
"cloudwatch:GetMetricData",
"cloudwatch:GetMetricStatistics",
"cloudwatch:ListMetrics"
```

The following IAM permissions are required for the transit gateway attachment (tgwa) metrics to work.

```json
"ec2:DescribeTags",
"ec2:DescribeInstances",
"ec2:DescribeRegions",
"ec2:DescribeTransitGateway*"
```

The following IAM permission is required to discover tagged API Gateway REST APIs:

```json
"apigateway:GET"
```

The following IAM permissions are required to discover tagged Database Migration Service (DMS) replication instances and tasks:

```json
"dms:DescribeReplicationInstances",
"dms:DescribeReplicationTasks"
```

## Override AWS endpoint urls
to support local testing all AWS urls can be overridden with by setting an environment variable `AWS_ENDPOINT_URL`
```shell
docker run -d --rm -v $PWD/credentials:/exporter/.aws/credentials -v $PWD/config.yml:/tmp/config.yml \
-e AWS_ENDPOINT_URL=http://localhost:4766 -p 5000:5000 --name yace ghcr.io/nerdswords/yet-another-cloudwatch-exporter:vx.xx.x # release version as tag - Do not forget the version 'v'
```

## Options
### RoleArns

Multiple roleArns are useful, when you are monitoring multi-account setup, where all accounts are using same AWS services. For example, you are running yace in monitoring account and you have number of accounts (for example newspapers, radio and television) running ECS clusters. Each account gives yace permissions to assume local IAM role, which has all the necessary permissions for Cloudwatch metrics. On this kind of setup, you could simply list:
```yaml
apiVersion: v1alpha1
sts-region: eu-west-1
discovery:
  jobs:
    - type: ecs-svc
      regions:
        - eu-north-1
      roles:
        - roleArn: "arn:aws:iam::1111111111111:role/prometheus" # newspaper
        - roleArn: "arn:aws:iam::2222222222222:role/prometheus" # radio
        - roleArn: "arn:aws:iam::3333333333333:role/prometheus" # television
      metrics:
        - name: MemoryReservation
          statistics:
            - Average
            - Minimum
            - Maximum
          period: 600
          length: 600
```

Additionally, if the IAM role you want to assume requires an [External ID](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html?icmpid=docs_iam_console) you can specify it this way:

```yaml
  roles:
    - roleArn: "arn:aws:iam::1111111111111:role/prometheus"
      externalId: "shared-external-identifier"
```

### Requests concurrency
The flags 'cloudwatch-concurrency' and 'tag-concurrency' define the number of concurrent request to cloudwatch metrics and tags. Their default value is 5.

Setting a higher value makes faster scraping times but can incur in throttling and the blocking of the API.

### Decoupled scraping
The exporter scraped cloudwatch metrics in the background in fixed interval.
This protects from the abuse of API requests that can cause extra billing in AWS account.

The flag 'scraping-interval' defines the seconds between scrapes.
The default value is 300.

## Using YACE as a Go library

It is possible to embed YACE in to an external application. This mode might be useful to you if you would like to scrape on demand or run in a stateless manner.

The entrypoint to use YACE as a library is the `UpdateMetrics` func in [update.go](./pkg/exporter.go#L35) which requires,
- `config`: this is the struct representation of the configuration defined in [Top Level Configuration](#top-level-configuration)
- `registry`: any prometheus compatible registry where scraped AWS metrics will be written
- `metricsPerQuery`: controls the same behavior defined by the CLI flag `metrics-per-query`
- `labelsSnakeCase`: controls the same behavior defined by the CLI flag `labels-snake-case`
- `cloudwatchSemaphore`/`tagSemaphore`: adjusts the concurrency of requests as defined by [Requests concurrency](#requests-concurrency). Pass in a different length channel to adjust behavior
- `cache`
  - Any implementation of the [SessionCache Interface](./pkg/session/sessions.go#L41)
  - `session.NewSessionCache(config, <fips value>)` would be the default
  - `<fips value>` is defined by the `fips` CLI flag
- `observedMetricLabels`
  - Prometheus requires that all metrics exported with the same key have the same labels
  - This map will track all labels observed and ensure they are exported on all metrics with the same key in the provided `registry`
  - You should provide the same instance of this map if you intend to re-use the `registry` between calls
- `logger`
  - Any implementation of the [Logger Interface](./pkg/logging/logger.go#L13)
  - `logging.NewLogger(logrus.StandardLogger())` is an acceptable default

The update definition also includes an exported slice of [Metrics](./pkg/exporter.go#L18) which includes AWS API call metrics. These can be registered with the provided `registry` if you want them
included in the AWS scrape results. If you are using multiple instances of `registry` it might make more sense to register these metrics in the application using YACE as a library to better
track them over the lifetime of the application.

## Troubleshooting / Debugging

### Help my metrics are intermittent

* Please, try out a bigger length e.g. for elb try out a length of 600 and a period of 600. Then test how low you can
go without losing data. ELB metrics on AWS are written every 5 minutes (300) in default.

### My metrics only show new values after 5 minutes

* Please, try to set a lower value for the 'scraping-interval' flag or set the 'decoupled-scraping' to false.

## Contribute

[Development Setup / Guide](/CONTRIBUTE.md)

## Thank you

* [Justin Santa Barbara](https://github.com/justinsb) - For telling me about AWS tags api which simplified a lot - Thanks!
* [Brian Brazil](https://github.com/brian-brazil) - Who gave a lot of feedback regarding UX and prometheus lib - Thanks!
