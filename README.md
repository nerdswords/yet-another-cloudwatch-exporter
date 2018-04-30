# YACE - yet another cloudwatch exporter

*[EXPERIMENTAL STATE]*  - Not sure if this project makes sense
and/or helps prometheus/aws community. Currently in quick iterrations mode
which will probably break things in next versions.

Written without much golang experience. Would love to get feedback :))

## Features
* Auto Discovery of aws resources through tags
* Abstract aws things a little bit to make it easier for users
* Allow easy adding of tag labels to instances

## Why did you not add instance labels to every metric?
This was my first thought but acutally i learned it is an anti-pattern.
This metrics are build for easy grouping. Here is an example:

```Example of grouping query```
```

## Configuration File

Currently supported aws services:
* es => Elasticsearch
* ec => Elasticache
* ec2 => Elastic compute cloud
* rds => Relation Database Service
* elb => Elastic Load Balancers

```
jobs:
  - name: elasticsearch
    discovery:
      region: eu-west-1
      type: "es"
      searchTags:
        - Key: type
          Value: ^(iwfm|k8s)$
      exportedTags:
        - type
      exportedAttributes:
        - DedicatedMasterCount
        - ElasticsearchVersion
        - InstanceCount
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

### Exported Attributes
Currently these possibilites are implemented:

Elasticsearch:
* DedicatedMasterCount
* ElasticsearchVersion
* InstanceCount
* VolumeSize

Elasticache:
* Engine
* EngineVersion
