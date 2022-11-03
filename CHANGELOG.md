# 0.42.0-alpha

* Resolve logging issue (@datsabk)
* MediaTailor - Correct dimension regex for MT (@scott-mccracken)
* Helm chart update for optional test-connection pod (@nikosmeds)
* Helm chart update to set priorityClassName (@nikosmeds)
* Bump github.com/aws/aws-sdk-go from 1.44.122 to 1.44.127
* Bump github.com/urfave/cli/v2 from 2.20.3 to 2.23.0

# 0.41.0-alpha

* Clean up unused variables. (@cristiangreco)
* Fix typo: sts-endpoint should be sts-region. (@cristiangreco)
* Enabled Managed prometheus metrics (@datsabk)
* Add support for AWS Kafka Connect (@cgowthaman)
* Import CloudWatch mixin. (@jeschkies)
* main.go refactoring: define cmd action as a separate func. (@cristiangreco)
* Add support for EMR Serverless (@cgowthaman)

# 0.40.0-alpha
* Fix typo in Charts.yml (@yasharne)
* Subcommand `verify-config` actually validates the config file. (@cristiangreco)
* Add dimensions regex for AmazonMQ. (@cristiangreco)
* Fix metrics with additional dimensions being not being scraped. (@cristiangreco)
* Remove unused code, add test for RemoveDuplicateMetrics. (@cristiangreco)
* Bump github.com/sirupsen/logrus
* Bump github.com/urfave/cli/v2
* Bump github.com/aws/aws-sdk-go
* Bump actions/setup-python

# 0.39.0-alpha
* Improve code quality and unblock this release (cristiangreco)
* Add helm chart (vkobets)
* Fix DX metrics (paulojmdias)
* Fix searchTags and bad dimension name (femiagbabiaka)
* Handle empty list in filter metric tests (mtt88)
* Add AWS Elemental MediaTailor support (scott-mccracken)
* Support storagegateway metrics (sedan07)
* Filter api gateway resources to skip "stages" (ch4rms)
* Bump aws-sdk, urfave/cli, prometheus/client_golang

# 0.38.0-alpha

* Set max page size for tagging API requests (#617)
* Build with Go 1.18

# 0.37.0-alpha
* New config `dimensionNameRequirements` allows autodiscovery jobs to only
  fetch metrics that include specified dimensions (jutley)
* Update deps

# 0.36.2-alpha
* Cost Reduction - Use less API requests if no tagged resources are found (cristiangreco)
* Update deps

# 0.36.1-alpha
* Use structured logs for logging interface (kgeckhart)

# 0.36.0-alpha

* *BREAKING CHANGE FOR LIBRARY USERS* Major refactoring of usage of logging library (kgeckhart)
* Minor update of deps and security patches (urfave/cli/v2, golangci/golangci-lint-action, github.com/prometheus/client_golang, github.com/stretchr/testify, github.com/aws/aws-sdk-go
* Updates of Readme (markwallsgrove)

# 0.35.0-alpha
* Update dependencies
* Improve / Document way how to use the exporter as external library (kgeckhart)
* Refactor label consistency (kgeckhart)
* Add suppot for vpc-endpoint (AWS/PrivateLinkEndpoints) (aleslash)
* Add support for vpc-endpoint-service (AWS/PrivateLinkServices) (aleslash)

# 0.34.0-alpha
* Update dependencies
* Add weekly dependabot updates (jylitalo)
* Add support for regional sts endpoints (matt-mercer)
* Add multi-arch docker build (charlie-haley)

New services
* Add global accelerator support (charlie-haley)
* Add AppStream support (jhuesemann)
* Add Managed Apache Airflow support (sdenham)
* Add KinesisAnalytics support (gumpt)

Bug Fixes
* Fix targetgroup arn lookup (domcyrus)
* Fix WorkGroup Dimension are not showing in Athena Metrics (sahajavidya)
* Improve regex performance (kgeckhart)
* Fix prometheus reload causing a goroutine leak (gumpt / cristiangreco)

Docs
* Added help for new contributors (aleslash)

# 0.33.0-alpha
* Add /healthz route which allows to deploy more secure with helm (aleslash)
* Read DMS replication instance identifier from the DMS API (nhinds)

# 0.32.0-alpha
* [BREAKING] Fix the calculation of start and end times for GetMetricData (csquire)
```
floating-time-window is now replaced with roundingPeriod

Specifies how the current time is rounded before calculating start/end times for CloudWatch GetMetricData requests. This rounding is optimize performance of the CloudWatch request. This setting only makes sense to use if, for example, you specify a very long period (such as 1 day) but want your times rounded to a shorter time (such as 5 minutes). to For example, a value of 300 will round the current time to the nearest 5 minutes. If not specified, the roundingPeriod defaults to the same value as shortest period in the job.
```
* Improve testing / linting (cristiangreco)
* Verify cli parameters and improve cli parsing (a0s)
* Allow to configure yace cli parameters via env variables (a0s)
* Improve error handling of cloudwatch (matthewnolf)
* Add support for directconnect and route53 health checks
* Improve throttling handling to AWS APIs (anilkun)
* Add issue templates to improve support (NickLarsenNZ)
* Allow setting default values for statistics (surminus)
* Fix apigateway method and resouce dimension bug (aleslash)

Thanks a lot to all contributors! - Lovely to see so much efforts especially in testing
to get this project more and more stable. - I know we are far away from a nice tested
code base but we are improving in the right direction and I really love to see all
of your efforts there. It is really appreciated from my side.

I just contacted AWS to get some open source credits so we can build some kind of
end to end tests. This shoud allow us to find tricky bugs earlier and not only when we ship
things.

Love to all of you, Thomas!

# 0.31.0-alpha
* [BREAKING] Decoupled scraping is now default. Removed code which allowed to use scraper without it.
```
# Those flags are just ignored
-decoupled-scraping=false
-decoupled-scraping=true
```
* [BREAKING] Small timeframes of scraping can be used again now. In the past yace decided the scraping
  interval based on config. This magic was removed for simplicity.
```
# In the past this would have in some cases still set --scraping-interval 600
--scraping-interval 10
# Now it really would scrape every 10 seconds which could introduce big API costs. So please watch
# your API requests!
--scraping-interval 10
```
* Fix problems with start/endtime of scrapes (klarrio-dlamb)
* Add support for Database Migration Service metrics
* Allow to hotreload config via /reload (antoniomerlin)

# 0.30.1-alpha
* *SECURITY* Fix issue with building binaries. Please update to mitigate (https://nvd.nist.gov/vuln/detail/CVE-2020-14039)
* Thanks jeason81 for reporting this security incident!

# 0.30.0-alpha
* *BREAKING* Introduce new version field to config file (jylitalo)
```
# Before
discovery:
  jobs:
# After
apiVersion: v1alpha1
discovery:
  jobs:
```
* [BUG] Fix issues with nilToZero (eminugurkenar)
* [BUG] Fix race condition setting end time for discovery jobs (cristiangreco)
* Simplify session creation code (jylitalo)
* Major improvement of aws discovery code (jylitalo)
* Major rewrite of the async scraping logic (rabunkosar-dd)
* Add support for AWS/ElasticBeanstalk (andyzasl)
* Upgrade golang to 1.17
* Upgrade golang libraries to newest versions

# 0.29.0-alpha
Okay, private things settled. We have a new organisation for
the project. Lets boost it and get the open PRs merged!
This version is like 0.28.0-alpha but docker images hosted on ghcr.io
and published via new github organisation nerdswords. Find
details [here](https://medium.com/@IT_Supertramp/reorganizing-yace-79d7149b9584).

Thanks to all there waiting and using the product! :)

- *BREAKING CHANGE* Using a new docker registry / organisation:
```yaml
# Before
quay.io/invisionag/yet-another-cloudwatch-exporter:v0.29.0-alpha
# Now
ghcr.io/nerdswords/yet-another-cloudwatch-exporter:v0.29.0-alpha
```

# 0.28.0-alpha
Sorry folks, I currently struggle a little bit
to get things merged fast due to a lot of private
stuff. Really appreciate all your PRs and
hope to get the bigger ones (which are sadly
still not merged yet) into next release.

Really appreciate any person working on this
project! - Have a nice day :)

- *BREAKING CHANGE* Added support for specifying an External ID with IAM role Arns (cristiangreco)
```yaml
# Before
discovery:
  jobs:
  - type: rds
    roleArns:
    - "arn:aws:iam::123456789012:role/Prometheus"
# After
discovery:
  jobs:
  - type: rds
    roles:
    - roleArn: "arn:aws:iam::123456789012:role/Prometheus"
      externalId: "shared-external-identifier" # optional
```
- Add alias for AWS/Cognito service (tohjustin)
- Fix logic in dimensions for Transit Gateway Attachments (rhys-evans)
- Fix bug with scraping intervals (boazreicher)
- Support arm64 builds (alias-dev)
- Fix IgnoreLength logic (dctrwatson)
- Simplify code base (jylitalo)
- Simplify k8s deployments for new users (mahmoud-abdelhafez)
- Handle metrics with '%' in their name (darora)
- Fix classic elb name (nhinds)
- Skip metrics in edge cases (arvidsnet)

Freshly shipped new integrations:
- Certificate Manager (mksh)
- WorkSpaces (kl4w)
- DDoSProtection / Shield (arvidsnet)

# 0.27.0-alpha

- Make exporter a library. (jeschkies)
- Add CLI option to validate config file (zswanson)
- Fix multidimensional static metric (nmiculinic)
- Fix scrapes running in EKS fail after first scrape (rrusso1982)
- Fix Docker build (jeschkies)
- Allow to use this project in China (insectme)
- Fix error retrieving kafka metrics (friedrichg)

Freshly integrated:
- Add AWS/NetworkFirewall (rhys-evans)
- Add AWS/Cassandra (bjhaid)
- Add AWS/AmazonMQ (saez0pub)
- Add AWS/Athena (haarchri)
- Add AWS/Neptune (benjaminaaron)

Thanks to doc fixes: calvinbui

# 0.26.1-alpha / 0.26.2-alpha / 0.26.3-alpha

- Fix CI issue

# 0.26.0-alpha

- *BREAKING CHANGE* Removed a need to use static dimensions in dynamic jobs in cases, when they cannot be parsed from ARNs (AndrewChubatiuk)
    ```
      # Before
      metrics:
      - name: NumberOfObjects
        statistics:
          - Average
        additionalDimensions:
          - name: StorageType
            value: AllStorageTypes
      # After
      metrics:
      - name: NumberOfObjects
        statistics:
          - Average
    ```
* *BREAKING CHANGE* Use small case for searchTags config option (AndrewChubatiuk)
    ```
    # Before
    searchTags:
    - Key: type
      Value: public
    # After
    searchTags:
    - key: type
      value: public
      ```
* *BREAKING CHANGE* CloudFront renamed from `cf` to `cloudfront`
    ```
    # Before
    - type: cf
    # After
    - type: cloudfront
      ```

- Added regular expressions to parse dimensions from resources (AndrewChubatiuk)
- Added option to use floating time windows (zqad)
- Added CLI option to validate config file (zswanson)
- Added AWS network Firewall (rhys-evans)
- Fixed multidimensional static metric (nmiculinic)
- Tidy up code (jylitalo)

# 0.25.0-alpha

- *BREAKING CHANGE* Use NaN as default if AWS returns nil (arnitolog)
- Add autodiscovery for AWS/EC2Spot (singhjagmohan1000)
- Add autodiscovery for DocumentDB (haarchri)
- Add autodiscovery for GameLift (jp)
- Added support for fips compliant endpoints (smcavallo)
- Update deps and build with golang 1.15 (smcavallo)

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
