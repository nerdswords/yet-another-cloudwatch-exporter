# 0.25.0-alpha

- *BREAKING CHANGE* Use NaN as default if AWS returns nil (arnitolog)
- Add autodiscovery for AWS/EC2Spot (singhjagmohan1000)
- Add autodiscovery for DocumentDB (haarchri)
- Add autodiscovery for GameLift (jp)
- Added support for fips compliant endpoints (smcavallo)

# 0.24.0-alpha

- Add API Gateway IAM info to README (Botono)
- Fix sorting of datapoints, add test util functions (Botono)
- Fix missing DataPoints and improve yace in various ways (vishalraina)
- Added Github action file to basic validation of incoming PR (vishalraina)
- Fix info metrics missing (goya)
- Add rds db clusters (goya)
- Fix missing labels (goya)

# 0.23.0-alpha

- Add sampleCount statistics (udhos)
- Add WAFv2 support (mksh)

# 0.22.0-alpha

- Fix alb issues (reddoggad)
- Add nlb support (reddoggad)

# 0.21.0-alpha

- Big tidy up of code, remove old methods and refactor used ones (jylitalo)
- Fix crashes where labels are not collected correctly (rrusso1982)
- Fix pointer bug causing metrics to be missing (jylitalo)
- Allow more then 25 apigateways to be discovered (udhos)

# 0.20.0-alpha

- Add api-gateway support (smcavallo)
- Improve metrics validation (jylitalo)
- Fix metrics with '<', '>' chars

# 0.19.1-alpha

- Remove error during build

# 0.19.0-alpha
Wow what a release. Thanks to all contributors. This is
our biggest release and it made me a lot of fun to see all those
contributions. From small doc changes (love those) to major rewrites
of big components or new complex features. Thanks!

* *BREAKING CHANGE* Add support for multiple roleArns (jylitalo)
```yaml
# Before
---
discovery:
  jobs:
  - type: rds
    roleArn: "arn:aws:iam::123456789012:role/Prometheus"
# After
discovery:
  jobs:
  - type: rds
    roleArns:
    - "arn:aws:iam::123456789012:role/Prometheus"
```
* Upgrade golang from 1.12 to 1.14
* Major linting of code and improving global code quality. (jylitalo)
* Improve logging (jylitalo)
* Add config validation. (daviddetorres)
* Added support for tags with '@' char included (afroschauer )
* Added Transit Gateway Attachment Metrics (rhys-evans)
* Fix information gathering if no data is retrieved by cloudwatch (daviddetorres)
* Improve docs (calvinbui)
* Add redshift support (smcavallo)
* Allow easier configuration through adding period / addCloudwatchTimestamp setting additionally
  to job level. (rrusso1982)
* Add initial unit tests (smcavallo)
* Add new configuration to allow snake case labels (rrusso1982)
* Fix complex metric dimension bug (rrusso1982)
* Upgrade golang packages (smcavallo)
* Set up correct partition for ASG for AWS China and GovCloud Regions (smcavallo)
* Add ability to set custom tags to discovery job metrics (goya)

# 0.18.0-alpha
* *BREAKING CHANGE* Add support for multiple regions (goya)
```yaml
# Before
---
discovery:
  jobs:
  - type: rds
    region: eu-west-1
# After
discovery:
  jobs:
  - type: rds
    regions:
    - eu-west-1
```
* Fix missing alb target group metrics (abhi4890 )
* Added support for step functions (smcavallo)

# 0.17.0-alpha
* Added support for sns / firehose (rhys-evans)
* Added support for fsx / appsync (arnitolog)

# 0.16.0-alpha
* Hugh rewrite: Decouple scraping and serving metrics. Thanks so much daviddetorres!
* *BREAKING CHANGE* Decoupled scraping and set scraping interval to 5 minutes.
```
The flag 'decoupled-scraping' makes the exporter to scrape Cloudwatch metrics in background in fixed intervals, in stead of each time that the '/metrics' endpoint is fetched. This protects from the abuse of API requests that can cause extra billing in AWS account. This flag is activated by default.

If the flag 'decoupled-scraping' is activated, the flag 'scraping-interval' defines the seconds between scrapes. Its default value is 300.
```
* Hugh rewrite: Rewrite of metric gathering to reduce API Limit problems. Thanks so much daviddetorres!
* Improvment of ALB data gathering and filtering (daviddetorres)
* Detect and fix bug after merge (deanrock)
* Add cloudfront support (mentos1386)

# 0.15.0-alpha
* Fixed docker run command in README.md (daviddetorres)
* Added support for Nat Gateway / Transit Gateway / Route 53 Resolver (j-nix)
* Added support for ECS/ContainerInsights (daviddetorres)
* Fix pagination for getMetricList (eminugurkenar)

# 0.14.7-alpha
* Change logging to json format (bheight-Zymergen)

# 0.14.6-alpha
* Add support for kafka (eminugurkenar)
* Add structured json logging (bheight-Zymergen)
* Increase code readability (bheight-Zymergen)
* Fix ecs scraping bug (rabunkosar-dd)
* Fix aws cloudwatch period bug (rabunkosar-dd)

# 0.14.5-alpha
* Fix sts api calls without specifying a region (nhinds)
* Update aws-sdk to v1.25.21 (nhinds)

# 0.14.4-alpha
* Fix github actions (nhinds)
* Update aws-sdk-go (deanrock)
* Avoid appending to a shared dimensions variable from inside a loop (nhinds)
* Remove hardcoded StorageType dimension from S3 metric (nhinds)

# 0.14.3-alpha
* Fix problems and crashes with ALBs and ELBs (Deepak1100)

# 0.14.2-alpha
* **BREAKING** Changing user in Docker image to be non root to adhere to potential security requirements. (whitlekx)
* Fix prometheus metric bug with new services with '-' e.g. ecs-svc.

# 0.14.1-alpha
* Was accidentally with code from 01.14.0-alpha released.

# 0.14.0-alpha
* **BREAKING** Default command in Dockerfile is changed to yace. This removes the need to add yace as command.
```yaml
# Before
        command:
          - "yace"
          - "--config.file=/tmp/config.yml"
# After
        args:
          - "--config.file=/tmp/config.yml"
```
* Add support for Elastic MapReduce (nhinds)
* Add support for SQS - (alext)
* Add support for ECS Services as ecs-svc
* Add support for NLB
* Add retries to cloudwatch api calls (Deepak1100)
* Fix dimension labels for static jobs (alext)

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

