# yet-another-cloudwatch-exporter

![Version: 0.14.0](https://img.shields.io/badge/Version-0.14.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.48.0-alpha](https://img.shields.io/badge/AppVersion-v0.48.0--alpha-informational?style=flat-square)

Yet Another Cloudwatch Exporter

**Homepage:** <https://github.com/nerdswords/yet-another-cloudwatch-exporter>

## Installation

```sh
helm repo add yet-another-cloudwatch-exporter https://nerdswords.github.io/yet-another-cloudwatch-exporter
helm install yet-another-cloudwatch-exporter/yet-another-cloudwatch-exporter
```

## Source Code

* <https://github.com/nerdswords/yet-another-cloudwatch-exporter>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| aws.aws_access_key_id | string | `nil` |  |
| aws.aws_secret_access_key | string | `nil` |  |
| aws.role | string | `nil` |  |
| aws.secret.includesSessionToken | bool | `false` |  |
| aws.secret.name | string | `nil` |  |
| config | string | `"apiVersion: v1alpha1\nsts-region: eu-west-1\ndiscovery:\n  exportedTagsOnMetrics:\n    ec2:\n      - Name\n    ebs:\n      - VolumeId\n  jobs:\n  - type: es\n    regions:\n      - eu-west-1\n    searchTags:\n      - key: type\n        value: ^(easteregg|k8s)$\n    metrics:\n      - name: FreeStorageSpace\n        statistics:\n        - Sum\n        period: 60\n        length: 600\n      - name: ClusterStatus.green\n        statistics:\n        - Minimum\n        period: 60\n        length: 600\n      - name: ClusterStatus.yellow\n        statistics:\n        - Maximum\n        period: 60\n        length: 600\n      - name: ClusterStatus.red\n        statistics:\n        - Maximum\n        period: 60\n        length: 600\n  - type: elb\n    regions:\n      - eu-west-1\n    length: 900\n    delay: 120\n    statistics:\n      - Minimum\n      - Maximum\n      - Sum\n    searchTags:\n      - key: KubernetesCluster\n        value: production-19\n    metrics:\n      - name: HealthyHostCount\n        statistics:\n        - Minimum\n        period: 600\n        length: 600 #(this will be ignored)\n      - name: HTTPCode_Backend_4XX\n        statistics:\n        - Sum\n        period: 60\n        length: 900 #(this will be ignored)\n        delay: 300 #(this will be ignored)\n        nilToZero: true\n      - name: HTTPCode_Backend_5XX\n        period: 60\n  - type: alb\n    regions:\n      - eu-west-1\n    searchTags:\n      - key: kubernetes.io/service-name\n        value: .*\n    metrics:\n      - name: UnHealthyHostCount\n        statistics: [Maximum]\n        period: 60\n        length: 600\n  - type: vpn\n    regions:\n      - eu-west-1\n    searchTags:\n      - key: kubernetes.io/service-name\n        value: .*\n    metrics:\n      - name: TunnelState\n        statistics:\n        - p90\n        period: 60\n        length: 300\n  - type: kinesis\n    regions:\n      - eu-west-1\n    metrics:\n      - name: PutRecords.Success\n        statistics:\n        - Sum\n        period: 60\n        length: 300\n  - type: s3\n    regions:\n      - eu-west-1\n    searchTags:\n      - key: type\n        value: public\n    metrics:\n      - name: NumberOfObjects\n        statistics:\n          - Average\n        period: 86400\n        length: 172800\n      - name: BucketSizeBytes\n        statistics:\n          - Average\n        period: 86400\n        length: 172800\n  - type: ebs\n    regions:\n      - eu-west-1\n    searchTags:\n      - key: type\n        value: public\n    metrics:\n      - name: BurstBalance\n        statistics:\n        - Minimum\n        period: 600\n        length: 600\n        addCloudwatchTimestamp: true\n  - type: kafka\n    regions:\n      - eu-west-1\n    searchTags:\n      - key: env\n        value: dev\n    metrics:\n      - name: BytesOutPerSec\n        statistics:\n        - Average\n        period: 600\n        length: 600\n  - type: appstream\n    regions:\n      - eu-central-1\n    searchTags:\n      - key: saas_monitoring\n        value: true\n    metrics:\n      - name: ActualCapacity\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: AvailableCapacity\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: CapacityUtilization\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: DesiredCapacity\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: InUseCapacity\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: PendingCapacity\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: RunningCapacity\n        statistics:\n          - Average\n        period: 600\n        length: 600\n      - name: InsufficientCapacityError\n        statistics:\n          - Average\n        period: 600\n        length: 600\n  - type: backup\n    regions:\n      - eu-central-1\n    searchTags:\n      - key: saas_monitoring\n        value: true\n    metrics:\n      - name: NumberOfBackupJobsCompleted\n        statistics:\n          - Average\n        period: 600\n        length: 600\nstatic:\n  - namespace: AWS/AutoScaling\n    name: must_be_set\n    regions:\n      - eu-west-1\n    dimensions:\n     - name: AutoScalingGroupName\n       value: Test\n    customTags:\n      - key: CustomTag\n        value: CustomValue\n    metrics:\n      - name: GroupInServiceInstances\n        statistics:\n        - Minimum\n        period: 60\n        length: 300"` |  |
| extraArgs | list | `[]` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/nerdswords/yet-another-cloudwatch-exporter"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| ingress.annotations | object | `{}` |  |
| ingress.className | string | `""` |  |
| ingress.enabled | bool | `false` |  |
| ingress.hosts[0].host | string | `"chart-example.local"` |  |
| ingress.hosts[0].paths[0].path | string | `"/"` |  |
| ingress.hosts[0].paths[0].pathType | string | `"ImplementationSpecific"` |  |
| ingress.tls | list | `[]` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| portName | string | `"http"` |  |
| priorityClassName | string | `nil` |  |
| prometheusRule.enabled | bool | `false` |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext | object | `{}` |  |
| service.port | int | `80` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| serviceMonitor.enabled | bool | `false` |  |
| testConnection | bool | `true` |  |
| tolerations | list | `[]` |  |
