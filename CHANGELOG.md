# 0.13.7
* Add region as exported label to metrics

# 0.13.6
* Fix errors with "=" in tags (cdchris12)
* Add curl to container for easy debugging (cdchris12)

# 0.13.5-alpha
* Limit concurrency of aws calls

# 0.13.4
* Add Autoscaling group support (wjam)
* Fix strange AWS namespace bug for static exports (AWS/EC2/API)
* Add warning if metric length of less than 300s is configured / Interminent metrics

# 0.13.3
* Fix ALB problems. Target Group metrics are now exported as aws_albtg
```
aws_albtg_request_count_sum{dimension_LoadBalancer="app/Test-ALB/fec38de4cf0cacb1",dimension_TargetGroup="targetgroup/Test/708ecba11979327b",name="arn:aws:elasticloadbalancing:eu-west-1:237935892384916:targetgroup/Test/708dcba119793234"} 0
```

# 0.13.2
* CI problem

# 0.13.1-alpha
* **BREAKING** For some metrics `cloudwatch:ListMetrics` iam permissions are needed. Please update your role!
* **BREAKING** Add 'v' to indicate it is a version number in docker tag / version output
```
# Before
 image: quay.io/invisionag/yet-another-cloudwatch-exporter:0.13.0
# After
 image: quay.io/invisionag/yet-another-cloudwatch-exporter:v0.13.0
```
* Use golang 1.12.0 to build
* Use goreleaser to release
* Update aws dependencies
* Use github actions as CI
* Migrate dependency management to golang modules

# 0.13.0-alpha
* **BREAKING** For some metrics `cloudwatch:ListMetrics` iam permissions are needed. Please update your role!
* **BREAKING** As adding cloudwatch timestamp breaks some metrics I decided to not set it as default anymore.
This should make it easier for new users to have fun with this project.
It fixes for some users `non-histogram and non-summary metrics should not have "_sum" suffix` bug.
```yaml
# Before
  metrics:
    - name: FreeStorageSpace
      disableTimestamp: true
# After
  metrics:
    - name: FreeStorageSpace

# Before
  metrics:
    - name: FreeStorageSpace
# After
  metrics:
    - name: FreeStorageSpace
      useCloudwatchTimestamp: true
```
* Add ability to specify additional dimensions on discovery jobs e.g. for BucketSizeBytes metrics on S3 (abuchananTW)
* Fix incorrect dimension value in case of alb in discovery config (GeeksWine)
* Add CLI command to debug output
* Add DynamoDB support

# 0.12.0 / 0.12.0-alpha
* **BREAKING** Add the exact timestamps from CloudWatch to the exporter Prometheus metrics (LeePorte)
* Add a new option `disableTimestamp` to not include a timestamp for a specific metric (it can be useful for sparse metrics, e.g. from S3) (LeePorte)
* Add support for kinesis (AndrewChubatiuk)

# 0.11.0
* **BREAKING** Add snake_case to prometheus metrics (sanchezpaco)
```yaml
# Before
aws_elb_requestcount_sum
# After
aws_elb_request_count_sum
```

* Add optional delay setting to scraping (Deepak1100)
```yaml
period: 60
length: 900
delay: 300
``` 

# 0.10.0
* Reduce usage of listMetrics calls (nhinds)
* Add support of iam roles (nhinds)
* Add optional roleArn setting, which allows scraping with different roles e.g. pull data from mulitple AWS accounts using cross-acount roles (nhinds)
```yaml
    metrics:
      - name: FreeStorageSpace
        roleArn: xxx
        statistics:
        - 'Sum'
        period: 600
        length: 60
```

# 0.9.0
* Add lambda support (nhinds)
* Fix support for listing multiple statistics per metric (nhinds)
* Add tag labels on metrics for easy querying (nhinds)
```
# Before
aws_ec2_cpuutilization_average + on (name) group_left(tag_Name) aws_ec2_info

# After, now name tags are on metrics and no grouping needed
aws_ec2_cpuutilization_average
```

* **BREAKING** Change config syntax. Now you can define tags which are exported as labels on metrics.
Before:

```yaml
discovery:
  - region: eu-west-1
    type: "es"
    searchTags:
      - Key: type
        Value: ^(easteregg|k8s)$
    metrics:
      - name: FreeStorageSpace
        statistics:
        - 'Sum'
        period: 600
        length: 60
```

New Syntax with optional exportedTagsOnMetrics:
```yaml
discovery:
  exportedTagsOnMetrics:
    ec2:
      - Name
  jobs:
    - region: eu-west-1
      type: "es"
      searchTags:
        - Key: type
          Value: ^(easteregg|k8s)$
      metrics:
        - name: FreeStorageSpace
          statistics:
          - 'Sum'
          period: 600
          length: 60
```

# 0.8.0
* Added VPN connection metrics (AndrewChubatiuk)
* Added ExtendedStatistics / percentiles (linefeedse)
* Added Average Statistic (AndrewChubatiuk)

# 0.7.0-alpha
* ALB Support (linefeedse)
* Custom lables for static metrics

Example
```yaml
static:
  - namespace: AWS/AutoScaling
    region: eu-west-1
    dimensions:
     - name: AutoScalingGroupName
       value: Test
    customTags:
      - Key: CustomTag
        Value: CustomValue
    metrics:
      - name: GroupInServiceInstances
        statistics:
        - 'Minimum'
        period: 60
        length: 300
```

# 0.6.1
* Sanitize colons in tags (linefeedse)

# 0.6.0 / 0.6.0-alpha
* **BREAKING**: Period/length uses now seconds instead of minutes
* **BREAKING**: Config file uses new syntax to support static
* Support of --debug flag which outputs some dev debug informations
* Support of metrics who are not included in tags api (e.g. autoscaling metrics)

Before
```yaml
jobs:
  - discovery:
      region: eu-west-1
      metrics:
        - name: HealthyHostCount
          statistics:
          - 'Minimum'
          period: 60
          length: 300
```

New Syntax:
```yaml
discovery:
  - region: eu-west-1
    type: elb
    searchTags:
      - Key: KubernetesCluster
        Value: production
    metrics:
      - name: HealthyHostCount
        statistics:
        - 'Minimum'
        period: 60
        length: 300
static:
  - namespace: AWS/AutoScaling
    region: eu-west-1
    dimensions:
     - name: AutoScalingGroupName
       value: Test
    metrics:
      - name: GroupInServiceInstances
        statistics:
        - 'Minimum'
        period: 60
        length: 300
```

# 0.5.0
* Support of EFS - Elastic File System
* Support of EBS - Elastic Block Storage

# 0.4.0
* **BREAKING**: Config file uses list as statistics config option,
this should reduce api calls for more than one statistics.

Before:
```yaml
jobs:
  - discovery:
    metrics:
        statistics: 'Maximum'
```
After:
```yaml
jobs:
  - discovery:
    metrics:
        statistics:
        - 'Maximum'
```
* Start to track changes in CHANGELOG.md
* Better error handling (discordianfish)
* Increase speed, not only each jobs threaded but now each metric
* Add s3 support
* Fix potential race condition during cloudwatch access
* Fix bug ignoring period in cloudwatch config
* Use interfaces for aws access and prepare code for unit tests
* Implement minimum, average, maximum, sum for cloudwatch api
* Implement way to handle multiple data returned by cloudwatch
* Update go dependencies
