# YACE - yet another cloudwatch exporter

*[EXPERIMENTAL STATE]*  - Not sure if this project makes sense
and/or helps prometheus/aws community. Currently in quick iteration mode
which will probably break things in next versions. Unstable till 1.0.0!

Written without much golang experience. Would love to get feedback :))

# Features
* Stop worrying about your aws IDs - Discovery of ec2, elb, rds, elasticsearch, elasticache
resources automatically through tags
* Filtering tags through regex
* Automatic adding of tag labels to metrics
* One prometheus metric with resource information (e.g. elasticsearch version or elb tags) which is easy groupable via prometheus
* Allows to set nil values of cloudwatch to 0. This allows building elb availability metrics more easily.

## Configuration File

Currently supported aws services:
* es => Elasticsearch
* ec => Elasticache
* ec2 => Elastic compute cloud
* rds => Relation Database Service
* elb => Elastic Load Balancers

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
```

## Contribution
Create Issue, get assigned, write pull request, get it merged, shipped :)

# Thank you
* [Justin Santa Barbara](https://github.com/justinsb) - Told me about aws tags api which simplified a lot - Thanks!
* [Brian Brazil](https://github.com/brian-brazil) - Gave a lot of feedback regarding ux and prometheus lib - Thanks!
