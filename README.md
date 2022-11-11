# YACE - yet another cloudwatch exporter

## What is this organisation?

[Medium Article about rebranding yace](https://medium.com/@IT_Supertramp/reorganizing-yace-79d7149b9584)

## Project Status

YACE is currently in quick iteration mode. Things will probably break in upcoming versions. However, it has been in production use at InVision AG for a couple of months already.

## Security

### Supported Versions

Only latest version gets security updates. We won't support older versions.

## Reporting a Vulnerability

In case of a vulnerability please directly contact us via mail - security@nerdswords.de

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
* Can be used as a library in an external application
* Support the scraping of custom namespaces metrics with the CloudWatch Dimensions.
* Supported services with auto discovery through tags:

  * acm (AWS/CertificateManager) - Certificate Manager
  * airflow (AmazonMWAA) - Managed Apache Airflow
  * alb (AWS/ApplicationELB) - Application Load Balancer
  * apigateway (AWS/ApiGateway) - API Gateway
  * appstream (AWS/AppStream) - AppStream
  * appsync (AWS/AppSync) - AppSync
  * amp (AWS/Prometheus) - Managed Service for Prometheus
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

## Image

* `ghcr.io/nerdswords/yet-another-cloudwatch-exporter:x.x.x` e.g. 0.5.0
* See [Releases](https://github.com/nerdswords/yet-another-cloudwatch-exporter/releases) for binaries

## Configuration

### Command Line Options

| Option               | Description                                                                       |
| -------------------- | --------------------------------------------------------------------------------- |
| labels-snake-case    | Causes labels on metrics to be output in snake case instead of camel case         |

### Top level configuration

| Key          | Description                                  |
|--------------|----------------------------------------------|
| apiVersion   | Configuration file version                   |
| sts-region   | Use STS regional endpoint (Optional)         |
| discovery    | Auto-discovery configuration                 |
| static       | List of static configurations                |
| customNamespace | List of custom namespace configurations        |

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
| type                   | Cloudwatch service alias ("alb", "ec2", etc) or namespace name ("AWS/EC2", "AWS/S3", etc).               |
| length (Default 120)   | How far back to request data for in seconds                                                              |
| delay                  | If set it will request metrics up until `current_time - delay`                                           |
| roles                  | List of IAM roles to assume (optional)                                                                   |
| searchTags             | List of Key/Value pairs to use for tag filtering (all must match), Value can be a regex.                 |
| period                 | Statistic period in seconds (General Setting for all metrics in this job)                                |
| statistics             | List of statistic types, e.g. "Minimum", "Maximum", etc (General Setting for all metrics in this job)    |
| roundingPeriod         | Specifies how the current time is rounded before calculating start/end times for CloudWatch GetMetricData requests. This rounding is optimize performance of the CloudWatch request. This setting only makes sense to use if, for example, you specify a very long period (such as 1 day) but want your times rounded to a shorter time (such as 5 minutes).  to For example, a value of 300 will round the current time to the nearest 5 minutes. If not specified, the roundingPeriod defaults to the same value as shortest period in the job.                     |
| addCloudwatchTimestamp | Export the metric with the original CloudWatch timestamp (General Setting for all metrics in this job)   |
| customTags             | Custom tags to be added as a list of Key/Value pairs                                                     |
| dimensionNameRequirements | List of metric dimensions to query. Before querying metric values, the total list of metrics will be filtered to only those that contain exactly this list of dimensions. An empty or undefined list results in all dimension combinations being included. |
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
| roles      | List of IAM roles to assume                                |
| namespace  | CloudWatch namespace                                       |
| name       | Must be set with multiple block definitions per namespace  |
| customTags | Custom tags to be added as a list of Key/Value pairs       |
| dimensions | CloudWatch metric dimensions as a list of Name/Value pairs |
| metrics    | List of metric definitions                                 |

### Example of config File

```yaml
apiVersion: v1alpha1
sts-region: eu-west-1
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
        period: 60
        length: 600
      - name: ClusterStatus.green
        statistics:
        - Minimum
        period: 60
        length: 600
      - name: ClusterStatus.yellow
        statistics:
        - Maximum
        period: 60
        length: 600
      - name: ClusterStatus.red
        statistics:
        - Maximum
        period: 60
        length: 600
  - type: elb
    regions:
      - eu-west-1
    length: 900
    delay: 120
    statistics:
      - Minimum
      - Maximum
      - Sum
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
      - name: HTTPCode_Backend_5XX
        period: 60
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
  - type: appstream
    regions:
      - eu-central-1
    searchTags:
      - key: saas_monitoring
        value: true
    metrics:
      - name: ActualCapacity
        statistics:
          - Average
        period: 600
        length: 600
      - name: AvailableCapacity
        statistics:
          - Average
        period: 600
        length: 600
      - name: CapacityUtilization
        statistics:
          - Average
        period: 600
        length: 600
      - name: DesiredCapacity
        statistics:
          - Average
        period: 600
        length: 600
      - name: InUseCapacity
        statistics:
          - Average
        period: 600
        length: 600
      - name: PendingCapacity
        statistics:
          - Average
        period: 600
        length: 600
      - name: RunningCapacity
        statistics:
          - Average
        period: 600
        length: 600
      - name: InsufficientCapacityError
        statistics:
          - Average
        period: 600
        length: 600
  - type: backup
    regions:
      - eu-central-1
    searchTags:
      - key: saas_monitoring
        value: true
    metrics:
      - name: NumberOfBackupJobsCompleted
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

[Source: [config_test.yml](pkg/testdata/config_test.yml)]

### Custom Namespace configuration

| Key                    | Description                                                      |
|------------------------| -----------------------------------------------------------------|
| regions                | List of AWS regions                                              |
| name                   | the name of your rule. It will be added as a label in Prometheus |
| namespace              | The Custom CloudWatch namespace                                  |
| roles                  | Roles that the exporter will assume                              |
| metrics                | List of metric definitions                                       |
| statistics             | default value for statistics                                     |
| nilToZero              | default value for nilToZero                                      |
| period                 | default value for period                                         |
| length                 | default value for length                                         |
| delay                  | default value for delay                                          |
| addCloudwatchTimestamp | default value for addCloudwatchTimestamp                         |

### Example of config File

```yaml
apiVersion: v1alpha1
sts-region: eu-west-1
customNamespace:
  - name: customEC2Metrics
    namespace: CustomEC2Metrics
    regions:
      - us-east-1
    metrics:
      - name: cpu_usage_idle
        statistics:
          - Average
        period: 300
        length: 300
        nilToZero: true
      - name: disk_free
        statistics:
          - Average
        period: 300
        length: 300
        nilToZero: true
```

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

## IAM

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

## EC2 and STS Assume Role
YACE will automatically attempt to assume the role associated with a machine within EC2. If this is undesirable behavior turn off the use of the use of metadata endpoint by setting the environment variable `AWS_EC2_METADATA_DISABLED=true`.

## Running locally

```shell
docker run -d --rm -v $PWD/credentials:/exporter/.aws/credentials -v $PWD/config.yml:/tmp/config.yml \
-p 5000:5000 --name yace ghcr.io/nerdswords/yet-another-cloudwatch-exporter:vx.xx.x # release version as tag - Do not forget the version 'v'

```


## Override AWS endpoint urls
to support local testing all AWS urls can be overridden with by setting an environment variable `AWS_ENDPOINT_URL`
```shell
docker run -d --rm -v $PWD/credentials:/exporter/.aws/credentials -v $PWD/config.yml:/tmp/config.yml \
-e AWS_ENDPOINT_URL=http://localhost:4766 -p 5000:5000 --name yace ghcr.io/nerdswords/yet-another-cloudwatch-exporter:vx.xx.x # release version as tag - Do not forget the version 'v'

```

## Kubernetes Installation
### Install with HELM
* [README](charts/yet-another-cloudwatch-exporter/README.md)


### Install with manifests
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
apiVersion: apps/v1
kind: Deployment
metadata:
  name: yace
spec:
  replicas: 1
  selector:
    matchLabels:
      name: yace
  template:
    metadata:
      labels:
        name: yace
    spec:
      containers:
      - name: yace
        image: ghcr.io/nerdswords/yet-another-cloudwatch-exporter:vx.x.x # release version as tag - Do not forget the version 'v'
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

### Embedding YACE as a library in an external application
It is possible to embed YACE in to an external application. This mode might be useful to you if you would like to scrape on demand or run in a stateless manner.

The entrypoint to use YACE as a library is the `UpdateMetrics` func in [update.go](./pkg/update.go#L15) which requires,
- `config`: this is the struct representation of the configuration defined in [Top Level Configuration](#top-level-configuration)
- `registry`: any prometheus compatible registry where scraped AWS metrics will be written
- `metricsPerQuery`: controls the same behavior defined by the CLI flag `metrics-per-query`
- `labelsSnakeCase`: controls the same behavior defined by the CLI flag `labels-snake-case`
- `cloudwatchSemaphore`/`tagSemaphore`: adjusts the concurrency of requests as defined by [Requests concurrency](#requests-concurrency). Pass in a different length channel to adjust behavior
- `cache`
  - Any implementation of the [SessionCache Interface](./pkg/sessions.go#L34)
  - `exporter.NewSessionCache(config, <fips value>)` would be the default
  - `<fips value>` is defined by the `fips` CLI flag
- `observedMetricLabels`
  - Prometheus requires that all metrics exported with the same key have the same labels
  - This map will track all labels observed and ensure they are exported on all metrics with the same key in the provided `registry`
  - You should provide the same instance of this map if you intend to re-use the `registry` between calls
- `logger`
  - Any implementation of the [Logger Interface](./pkg/update.go#L50)
  - `exporter.NewLogrusLogger(log.StandardLogger())` is an acceptable default

The update definition also includes an exported slice of [Metrics](./pkg/update.go#L11) which includes AWS API call metrics. These can be registered with the provided `registry` if you want them
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

# Thank you

* [Justin Santa Barbara](https://github.com/justinsb) - For telling me about AWS tags api which simplified a lot - Thanks!
* [Brian Brazil](https://github.com/brian-brazil) - Who gave a lot of feedback regarding UX and prometheus lib - Thanks!
