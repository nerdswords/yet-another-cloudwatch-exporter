# YACE - yet another cloudwatch exporter

*[EXPERIMENTAL STATE]*  - Not sure if this project makes sense
and/or helps prometheus/aws community. Currently in quick iteration mode
which will probably break things in next versions.

Written without much golang experience. Would love to get feedback :))

# Features
* Stop worrying about your aws IDs - Discovery of ec2, elb, rds, elasticsearch, elasticache
resources automatically through tags
* Filtering tags through regex
* Automatic adding of tag labels to metrics
* One prometheus metric with resource information (e.g. elasticsearch version or elb tags) which is easy groupable via prometheus

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
```

## Contribution
Create Issue, get assigned, write pull request, get it merged, shipped :)
