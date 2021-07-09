# YACE Helm Chart

## Requirements

- Kubernetes >= 1.17 (not tested on anything lower)
- `helm` >= 3.6.0 (not tested on anything lower)
- Properly configured AWS role; details [here](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html); included is a terraform module that will do all of this for you
    - The role must have this policy for the pod to scrape CloudWatch:

        ```json
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": [
                        "cloudwatch:Get*",
                        "cloudwatch:Describe*",
                        "cloudwatch:List*",
                        "tag:GetResources"
                    ],
                    "Resource": ["*"]
                }
            ]
        }
        ```

## Installation
`helm install yace helm/charts/yace --values values.yaml`; Note: You *must* fill in all values with your own values, or use the included terraform module that will do it all for you, including deploying the chart

To deploy the chart with terraform:

    ```tf
    module "yace" {
      source = path/to/yace"

      region      = "your-region"
      environment = "your-environment"

      cluster_name = "your-eks-cluster-name"
      chart        = "path/to/helm/charts/yace"
    }
    ```

## Usage notes
- Pod is listening on port `5000`; it can be scraped at `/metrics` endpoint
- This helm chart assumes AWS role linked to service account; **it will not work if you do not do it this way**
- The scrape config for prometheus is detailed below:

    ```yaml
    - job_name: 'yace'
      kubernetes_sd_configs:
      - role: pod
      scheme: http
      metrics_path: /metrics
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      authorization:
        credentials_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_container_name, __meta_kubernetes_pod_container_port_number]
        action: keep
        regex: yace;5000
    ```
