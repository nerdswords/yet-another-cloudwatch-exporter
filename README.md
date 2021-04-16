# YACE - yet another cloudwatch exporter [![Docker Image](https://quay.io/repository/invisionag/yet-another-cloudwatch-exporter/status?token=58e4108f-9e6f-44a4-a5fd-0beed543a271 "Docker Repository on Quay")](https://quay.io/repository/invisionag/yet-another-cloudwatch-exporter)

## Project Status

YACE is currently in quick iteration mode. Things will probably break in upcoming versions. However, it has been in production use at InVision AG for a couple of months already.

## Security

### Supported Versions

Only latest version gets security updates. We won't support older versions.

## Reporting a Vulnerability

In case of a vulnerability please directly contact us via mail - security@invision.de

Do not disclose any specifics in github issues! - Thank you.

We will contact you as soon as possible.

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
* Supported services with auto discovery through tags:

  * alb (AWS/ApplicationELB) - Application Load Balancer
  * apigateway (AWS/ApiGateway) - Api Gateway
  * appsync (AWS/AppSync) - AppSync
  * athena (AWS/Athena) - Athena
  * billing (AWS/Billing) - Billing
  * cassandra (AWS/Cassandra) - Cassandra
  * cloudfront (AWS/CloudFront) - Cloud Front
  * docdb (AWS/DocDB) - DocumentDB (with MongoDB compatibility)
  * dynamodb (AWS/DynamoDB) - NoSQL Online Datenbank Service
  * ebs (AWS/EBS) - Elastic Block Storage
  * ec (AWS/Elasticache) - ElastiCache
  * ec2 (AWS/EC2) - Elastic Compute Cloud
  * ec2Spot (AWS/EC2Spot) - Elastic Compute Cloud for Spot Instances
  * ecs-svc (AWS/ECS) - Elastic Container Service (Service Metrics)
  * ecs-containerinsights (ECS/ContainerInsights) - ECS/ContainerInsights (Fargate metrics)
  * efs (AWS/EFS) - Elastic File System
  * elb (AWS/ELB) - Elastic Load Balancer
  * emr (AWS/ElasticMapReduce) - Elastic MapReduce
  * es (AWS/ES) - ElasticSearch
  * fsx (AWS/FSx) - FSx File System
  * gamelift (AWS/GameLift) - GameLift
  * glue (Glue) - AWS Glue Jobs
  * kinesis (AWS/Kinesis) - Kinesis Data Stream
  * nfw (AWS/NetworkFirewall) - Network Firewall
  * ngw (AWS/NATGateway) - Nat Gateway
  * lambda (AWS/Lambda) - Lambda Functions
  * neptune (AWS/Neptune) - Neptune
  * nlb (AWS/NetworkELB) - Network Load Balancer
  * redshift (AWS/Redshift) - Redshift Database
  * rds (AWS/RDS) - Relational Database Service
  * r53r (AWS/Route53Resolver) - Route53 Resolver
  * s3 (AWS/S3) - Object Storage
  * ses (AWS/SES) - Simple Email Service
  * sqs (AWS/SQS) - Simple Queue Service
  * tgw (AWS/TransitGateway) - Transit Gateway
  * vpn (AWS/VPN) - VPN connection
  * asg (AWS/AutoScaling) - Auto Scaling Group
  * kafka (AWS/Kafka) - Managed Apache Kafka
  * firehose (AWS/Firehose) - Managed Streaming Service
  * sns (AWS/SNS) - Simple Notification Service
  * sfn (AWS/States) - Step Functions
  * wafv2 (AWS/WAFV2) - Web Application Firewall v2

## Image

* `quay.io/invisionag/yet-another-cloudwatch-exporter:x.x.x` e.g. 0.5.0
* See [Releases](https://github.com/ivx/yet-another-cloudwatch-exporter/releases) for binaries

## Configuration

### Command Line Options

| Option               | Description                                                                       |
| -------------------- | --------------------------------------------------------------------------------- |
| labels-snake-case    | Causes labels on metrics to be output in snake case instead of camel case         |
| floating-time-window | Use a floating start/end time window instead of rounding times to 5 min intervals |

### Top level configuration

| Key       | Description                   |
| --------- | ----------------------------- |
| discovery | Auto-discovery configuration  |
| static    | List of static configurations |

### Auto-discovery configuration

| Key                   | Description                                       |
| --------------------- | ------------------------------------------------- |
| exportedTagsOnMetrics | List of tags per service to export to all metrics |
| jobs                  | List of auto-discovery jobs                       |

exportedTagsOnMetrics example:

```yaml
exportedTagsOnMetrics:
  ec2:
    - Name
    - type
```

Note: Only [tagged resources](https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html) are discovered. 

### Auto-discovery job

| Key                    | Description                                                                                              |
| ---------------------- | -------------------------------------------------------------------------------------------------------- |
| regions                | List of AWS regions                                                                                      |
| type                   | Cloudwatch service alias ("alb", "ec2", etc) or namespace name ("AWS/EC2", "AWS/S3", etc).                                                |
| length (Default 120)   | How far back to request data for in seconds                                                              |
| delay                  | If set it will request metrics up until `current_time - delay`                                           |
| roleArns               | List of IAM roles to assume (optional)                                                                   |
| searchTags             | List of Key/Value pairs to use for tag filtering (all must match), Value can be a regex.                 |
| period                 | Statistic period in seconds (General Setting for all metrics in this job)                                |
| addCloudwatchTimestamp | Export the metric with the original CloudWatch timestamp (General Setting for all metrics in this job)   |
| customTags             | Custom tags to be added as a list of Key/Value pairs                                                     |
| metrics                | List of metric definitions                                                                               |

searchTags example:

```yaml
searchTags:
  - key: env
    value: production
```

### Metric definition

| Key                    | Description                                                                             |
| ---------------------- | --------------------------------------------------------------------------------------- |
| name                   | CloudWatch metric name                                                                  |
| statistics             | List of statistic types, e.g. "Minimum", "Maximum", etc.                                |
| period                 | Statistic period in seconds (Overrides job level setting)                               |
| length                 | How far back to request data for in seconds(for static jobs)                            |
| delay                  | If set it will request metrics up until `current_time - delay`(for static jobs)         |
| nilToZero              | Return 0 value if Cloudwatch returns no metrics at all. By default NaN will be reported |
| addCloudwatchTimestamp | Export the metric with the original CloudWatch timestamp (Overrides job level setting)  |

* Available statistics: Maximum, Minimum, Sum, SampleCount, Average, pXX.
* **Watch out using `addCloudwatchTimestamp` for sparse metrics, e.g from S3, since Prometheus won't scrape metrics containing timestamps older than 2-3 hours**
* **Setting Inheritance: Some settings at the job level are overridden by settings at the metric level.  This allows for a specific setting to override a
general setting.  The currently inherited settings are period, and addCloudwatchTimestamp**

### Static configuration

| Key        | Description                                                |
| ---------- | ---------------------------------------------------------- |
| regions    | List of AWS regions                                        |
| roleArns   | List of IAM roles to assume                                |
| namespace  | CloudWatch namespace                                       |
| name       | Must be set with multiple block definitions per namespace  |
| customTags | Custom tags to be added as a list of Key/Value pairs       |
| dimensions | CloudWatch metric dimensions as a list of Name/Value pairs |
| metrics    | List of metric definitions                                 |

### Example of config File

```yaml
discovery:
  exportedTagsOnMetrics:
    ec2:
      - Name
    ebs:
      - VolumeId
  jobs:
  - type: es
    regions:
      - eu-west-1
    searchTags:
      - key: type
        value: ^(easteregg|k8s)$
    metrics:
      - name: FreeStorageSpace
        statistics:
        - Sum
        period: 600
        length: 60
      - name: ClusterStatus.green
        statistics:
        - Minimum
        period: 600
        length: 60
      - name: ClusterStatus.yellow
        statistics:
        - Maximum
        period: 600
        length: 60
      - name: ClusterStatus.red
        statistics:
        - Maximum
        period: 600
        length: 60
  - type: elb
    regions:
      - eu-west-1
    length: 900
    delay: 120
    searchTags:
      - key: KubernetesCluster
        value: production-19
    metrics:
      - name: HealthyHostCount
        statistics:
        - Minimum
        period: 600
        length: 600 #(this will be ignored)
      - name: HTTPCode_Backend_4XX
        statistics:
        - Sum
        period: 60
        length: 900 #(this will be ignored)
        delay: 300 #(this will be ignored)
        nilToZero: true
  - type: alb
    regions:
      - eu-west-1
    searchTags:
      - key: kubernetes.io/service-name
        value: .*
    metrics:
      - name: UnHealthyHostCount
        statistics: [Maximum]
        period: 60
        length: 600
  - type: vpn
    regions:
      - eu-west-1
    searchTags:
      - key: kubernetes.io/service-name
        value: .*
    metrics:
      - name: TunnelState
        statistics:
        - p90
        period: 60
        length: 300
  - type: kinesis
    regions:
      - eu-west-1
    metrics:
      - name: PutRecords.Success
        statistics:
        - Sum
        period: 60
        length: 300
  - type: s3
    regions:
      - eu-west-1
    searchTags:
      - key: type
        value: public
    metrics:
      - name: NumberOfObjects
        statistics:
          - Average
        period: 86400
        length: 172800
      - name: BucketSizeBytes
        statistics:
          - Average
        period: 86400
        length: 172800
  - type: ebs
    regions:
      - eu-west-1
    searchTags:
      - key: type
        value: public
    metrics:
      - name: BurstBalance
        statistics:
        - Minimum
        period: 600
        length: 600
        addCloudwatchTimestamp: true
  - type: kafka
    regions:
      - eu-west-1
    searchTags:
      - key: env
        value: dev
    metrics:
      - name: BytesOutPerSec
        statistics:
        - Average
        period: 600
        length: 600
static:
  - namespace: AWS/AutoScaling
    name: must_be_set
    regions:
      - eu-west-1
    dimensions:
     - name: AutoScalingGroupName
       value: Test
    customTags:
      - key: CustomTag
        value: CustomValue
    metrics:
      - name: GroupInServiceInstances
        statistics:
        - Minimum
        period: 60
        length: 300
```

[Source: [config_test.yml](config_test.yml)]

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

# Availability Metric for ELBs (Sucessfull requests / Total Requests) + k8s service name
# Use nilToZero on all metrics else it won't work
((aws_elb_request_count_sum - on (name) group_left() aws_elb_httpcode_backend_4_xx_sum) - on (name) group_left() aws_elb_httpcode_backend_5_xx_sum) + on (name) group_left(tag_kubernetes_io_service_name) aws_elb_info

# Forecast your elasticsearch disk size in 7 days and report metrics with tags type and version
predict_linear(aws_es_free_storage_space_minimum[2d], 86400 * 7) + on (name) group_left(tag_type, tag_version) aws_es_info

# Forecast your cloudwatch costs for next 32 days based on last 10 minutes
# 1.000.000 Requests free
# 0.01 Dollar for 1.000 GetMetricStatistics Api Requests (https://aws.amazon.com/cloudwatch/pricing/)
((increase(yace_cloudwatch_requests_total[10m]) * 6 * 24 * 32) - 100000) / 1000 * 0.01
```

## IAM

The following IAM permissions are required for YACE to work.

```json
"tag:GetResources",
"cloudwatch:GetMetricData",
"cloudwatch:GetMetricStatistics",
"cloudwatch:ListMetrics"
```

The following IAM permissions are required for the transit gateway attachment (twga) metrics to work.
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

## Running locally

```shell
docker run -d --rm -v $PWD/credentials:/exporter/.aws/credentials -v $PWD/config.yml:/tmp/config.yml \
-p 5000:5000 --name yace quay.io/invisionag/yet-another-cloudwatch-exporter:vx.xx.x # release version as tag - Do not forget the version 'v'

```

## Kubernetes Installation

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: yace
data:
  config.yml: |-
    ---
    # Start of config file
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: yace
spec:
  replicas: 1
  template:
    metadata:
      labels:
        name: yace
    spec:
      containers:
      - name: yace
        image: quay.io/invisionag/yet-another-cloudwatch-exporter:vx.x.x # release version as tag - Do not forget the version 'v'
        imagePullPolicy: IfNotPresent
        args:
          - "--config.file=/tmp/config.yml"
        ports:
        - name: app
          containerPort: 5000
        volumeMounts:
        - name: config-volume
          mountPath: /tmp
      volumes:
      - name: config-volume
        configMap:
          name: yace
```
## Options
### RoleArns

Multiple roleArns are useful, when you are monitoring multi-account setup, where all accounts are using same AWS services. For example, you are running yace in monitoring account and you have number of accounts (for example newspapers, radio and television) running ECS clusters. Each account gives yace permissions to assume local IAM role, which has all the necessary permissions for Cloudwatch metrics. On this kind of setup, you could simply list:
```yaml
  jobs:
    - type: ecs-svc
      regions:
        - eu-north-1
      roleArns:
        - "arn:aws:iam::111111111111:role/prometheus" # newspaper
        - "arn:aws:iam:2222222222222:role/prometheus" # radio
        - "arn:aws:iam:3333333333333:role/prometheus" # television
      metrics:
        - name: MemoryReservation
          statistics:
            - Average
            - Minimum
            - Maximum
          period: 600
          length: 600
```

### Requests concurrency
The flags 'cloudwatch-concurrency' and 'tag-concurrency' define the number of concurrent request to cloudwatch metrics and tags. Their default value is 5.

Setting a higher value makes faster scraping times but can incur in throttling and the blocking of the API.

### Decoupled scraping
The flag 'decoupled-scraping' makes the exporter to scrape Cloudwatch metrics in background in fixed intervals, in stead of each time that the '/metrics' endpoint is fetched. This protects from the abuse of API requests that can cause extra billing in AWS account. This flag is activated by default.

If the flag 'decoupled-scraping' is activated, the flag 'scraping-interval' defines the seconds between scrapes. Its default value is 300.

## Troubleshooting / Debugging

### Help my metrics are intermittent

* Please, try out a bigger length e.g. for elb try out a length of 600 and a period of 600. Then test how low you can
go without losing data. ELB metrics on AWS are written every 5 minutes (300) in default.

### My metrics only show new values after 5 minutes

* Please, try to set a lower value for the 'scraping-interval' flag or set the 'decoupled-scraping' to false.

## Contribute

[Development Setup / Guide](/CONTRIBUTE.md)

# Thank you

* [Justin Santa Barbara](https://github.com/justinsb) - For telling me about AWS tags api which simplified a lot - Thanks!
* [Brian Brazil](https://github.com/brian-brazil) - Who gave a lot of feedback regarding UX and prometheus lib - Thanks!
