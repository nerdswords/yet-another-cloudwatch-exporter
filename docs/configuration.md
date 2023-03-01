
# Configuration

YACE has two configuration mechanisms, that work side by side. First, some global parameters can be configured through command line arguments. Second, the scraping configuration has to be provided through a YAML file. The configuration file path is passed to the exporter through the `--config.file` command line argument.

## Command Line Options

| Option                 | Description                                                               | Default    |
| ---------------------- | ------------------------------------------------------------------------- | ---------- |
| listen-address         | Network address to listen to                                              | :5000      |
| config.file            | Path to the configuration file                                            | config.yml |
| debug                  | Add verbose logging                                                       | false      |
| fips                   | Use FIPS compliant AWS API                                                | false      |
| cloudwatch-concurrency | Maximum number of concurrent requests to CloudWatch API                   | 5          |
| tag-concurrency        | Maximum number of concurrent requests to Resource Tagging API             | 5          |
| scraping-interval      | Seconds to wait between scraping the AWS metrics                          | 300        |
| metrics-per-query      | Number of metrics made in a single GetMetricsData request                 | 500        |
| labels-snake-case      | Causes labels on metrics to be output in snake case instead of camel case | false      |

## Top level configuration

Below are the top level fields of the YAML configuration file.

| Key             | Description                             |
| --------------- | --------------------------------------- |
| apiVersion      | Configuration file version              |
| sts-region      | Use STS regional endpoint (Optional)    |
| discovery       | Auto-discovery configuration            |
| static          | List of static configurations           |
| customNamespace | List of custom namespace configurations |

## Auto-discovery configuration

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

| Key                       | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| regions                   | List of AWS regions                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| type                      | Cloudwatch service alias ("alb", "ec2", etc) or namespace name ("AWS/EC2", "AWS/S3", etc).                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| length (Default 120)      | How far back to request data for in seconds                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| delay                     | If set it will request metrics up until `current_time - delay`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| roles                     | List of IAM roles to assume (optional)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| searchTags                | List of Key/Value pairs to use for tag filtering (all must match), Value can be a regex.                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| period                    | Statistic period in seconds (General Setting for all metrics in this job)                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| statistics                | List of statistic types, e.g. "Minimum", "Maximum", etc (General Setting for all metrics in this job)                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| roundingPeriod            | Specifies how the current time is rounded before calculating start/end times for CloudWatch GetMetricData requests. This rounding is optimize performance of the CloudWatch request. This setting only makes sense to use if, for example, you specify a very long period (such as 1 day) but want your times rounded to a shorter time (such as 5 minutes).  to For example, a value of 300 will round the current time to the nearest 5 minutes. If not specified, the roundingPeriod defaults to the same value as shortest period in the job. |
| addCloudwatchTimestamp    | Export the metric with the original CloudWatch timestamp (General Setting for all metrics in this job)                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| customTags                | Custom tags to be added as a list of Key/Value pairs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| dimensionNameRequirements | List of metric dimensions to query. Before querying metric values, the total list of metrics will be filtered to only those that contain exactly this list of dimensions. An empty or undefined list results in all dimension combinations being included.                                                                                                                                                                                                                                                                                        |
| metrics                   | List of metric definitions                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |

searchTags example:

```yaml
searchTags:
  - key: env
    value: production
```

## Metric definition

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

## Example of config File

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

[Source: [config_test.yml](../pkg/config/testdata/config_test.yml)]

### Custom Namespace configuration

| Key                    | Description                                                      |
| ---------------------- | ---------------------------------------------------------------- |
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

## Example of config File

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
