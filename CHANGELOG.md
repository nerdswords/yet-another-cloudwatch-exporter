# 0.10.0
* Reduce usage of listMetrics calls
* Add support of iam roles
* Add optional roleArn setting, which allows scraping with different roles e.g. pull data from mulitple AWS accounts using cross-acount roles
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
* Add lambda support
* Fix support for listing multiple statistics per metric
* Add tag labels on metrics for easy querying
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
* Added VPN connection metrics
* Added ExtendedStatistics (percentiles)
* Added Average Statistic

# 0.7.0-alpha
* ALB Support
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
* Sanitize colons in tags

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
* Increase speed, not only each jobs threaded but now each metric
* Add s3 support
* Fix potential race condition during cloudwatch access
* Fix bug ignoring period in cloudwatch config
* Use interfaces for aws access and prepare code for unit tests
* Implement minimum, average, maximum, sum for cloudwatch api
* Implement way to handle multiple data returned by cloudwatch
* Update go dependencies
