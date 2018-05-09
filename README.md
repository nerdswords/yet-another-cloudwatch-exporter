# [![Docker Image](https://quay.io/repository/invisionag/yet-another-cloudwatch-exporter/status?token=58e4108f-9e6f-44a4-a5fd-0beed543a271 "Docker Repository on Quay")](https://quay.io/repository/invisionag/yet-another-cloudwatch-exporter) YACE - yet another cloudwatch exporter

## Experimental State

Currently in quick iteration mode which will probably break things in next versions.

**Unstable till 1.0.0 - Use with care!**

## Features
* Stop worrying about your AWS IDs - Auto discovery of resources through tags
* Filter monitored resources through regex
* Automatic adding of tag labels to metrics
* Allows to export 0 even if cloudwatch returns nil
* Supported services:
  - es - elasticsearch
  - ec - elasticache
  - ec2 - elastic compute cloud
  - rds - relational database service
  - elb - elastic load balancers

## Image
* `quay.io/invisionag/yet-another-cloudwatch-exporter:x.x.x` e.g. 10.1.3
* Binaries on release page

## Config

Example of config File
```
jobs:
  - discovery:
      region: eu-west-1
      type: "es"
      searchTags:
        - Key: type
          Value: ^(easteregg|k8s)$
    metrics:
      - name: FreeStorageSpace
        statistics: 'Sum'
        period: 600
        length: 60
      - name: ClusterStatus.green
        statistics: 'Minimum'
        period: 600
        length: 60
      - name: ClusterStatus.yellow
        statistics: 'Maximum'
        period: 600
        length: 60
      - name: ClusterStatus.red
        statistics: 'Maximum'
        period: 600
        length: 60
  - discovery:
      type: "elb"
      region: eu-west-1
      searchTags:
        - Key: KubernetesCluster
          Value: production-19
    metrics:
      - name: HealthyHostCount
        statistics: 'Minimum'
        period: 60
        length: 300
      - name: HTTPCode_Backend_4XX
        statistics: 'Sum'
        period: 60
        length: 900
        nilToZero: true
```

## Example metrics
```
# Metrics
aws_ec2_cpuutilization_maximum{name="arn:aws:ec2:eu-west-1:472724724:instance/i-someid"} 57.2916666666667
aws_elb_healthyhostcount_minimum{name="arn:aws:elasticloadbalancing:eu-west-1:472724724:loadbalancer/a815b16g3417211e7738a02fcc13bbf9"} 9
aws_elb_httpcode_backend_4xx_sum{name="arn:aws:elasticloadbalancing:eu-west-1:472724724:loadbalancer/a815b16g3417211e7738a02fcc13bbf9"} 1

# Info helper with tags
aws_elb_info{name="arn:aws:elasticloadbalancing:eu-west-1:472724724:loadbalancer/a815b16g3417211e7738a02fcc13bbf9",tag_KubernetesCluster="production-19",tag_Name="",tag_kubernetes_io_cluster_production_19="owned",tag_kubernetes_io_service_name="nginx-ingress/private-ext"} 0
aws_ec2_info{name="arn:aws:ec2:eu-west-1:472724724:instance/i-someid",tag_Name="jenkins"} 0

# Track cloudwatch requests to calculate costs
yace_cloudwatch_requests_total 168
```

## Example queries

```
# CPUUtilization + Name tag of the instance id - No more instance id needed for monitoring
aws_ec2_cpuutilization_average + on (name) group_left(tag_Name) aws_ec2_info

# Free Storage in Megabytes + tag Type of the elasticsearch cluster
(aws_es_freestoragespace_sum + on (name) group_left(tag_Type) aws_es_info) / 1024

# Add kubernetes / kops tags on 4xx elb metrics
(aws_elb_httpcode_backend_4xx_sum + on (name) group_left(tag_KubernetesCluster,tag_kubernetes_io_service_name) aws_elb_info)

# Availability Metric for ELBs (Sucessfull requests / Total Requests) + k8s service name
# Use nilToZero on all metrics else it won't work
((aws_elb_requestcount_sum - on (name) group_left() aws_elb_httpcode_backend_4xx_sum) - on (name) group_left() aws_elb_httpcode_backend_5xx_sum) + on (name) group_left(tag_kubernetes_io_service_name) aws_elb_info

# Forecast your elasticsearch disk size in 7 days and report metrics with tags type and version
predict_linear(aws_es_freestoragespace_minimum[2d], 86400 * 7) + on (name) group_left(tag_type, tag_version) aws_es_info

# Forecast your cloudwatch costs for next 32 days based on last 10 minutes
# 1.000.000 Requests free
# 0.01 Dollar for 1.000 GetMetricStatistics Api Requests (https://aws.amazon.com/cloudwatch/pricing/)
((increase(yace_cloudwatch_requests_total[10m]) * 6 * 24 * 32) - 100000) / 1000 * 0.01
```

## IAM
These are the currently needed IAM permissions.
```
"tag:getResources",
"cloudwatch:GetMetricStatistics",
```

## Kubernetes Installation
```
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: yace
data:
  config.yml: |-
    ---
    jobs:
      - discovery:
          region: eu-west-1
          type: "ec2"
          searchTags:
            - Key: Name
              Value: jenkins
        metrics:
          - name: CPUUtilization
            statistics: 'Maximum'
            period: 30
            length: 30
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
        image: quay.io/invisionag/yet-another-cloudwatch-exporter:x.x.x # release version as tag
        imagePullPolicy: IfNotPresent
        command:
          - "yace"
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

## Contribute
[Development Setup / Guide](/CONTRIBUTE.md)

# Thank you
* [Justin Santa Barbara](https://github.com/justinsb) - Told me about AWS tags api which simplified a lot - Thanks!
* [Brian Brazil](https://github.com/brian-brazil) - Gave a lot of feedback regarding ux and prometheus lib - Thanks!
