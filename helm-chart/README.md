## Chart Details
This chart will do the following:
* Deploys a YACE (yet-another-cloudwatch-exporter) pod 
* Deploys a configurable configmap hosting the configuration read by the YACE pod

## Installing the Chart
To install the chart with the release name `YACE`:
```bash
$ helm install YACE yet-another-cloudwatch-exporter -n <namespace>
```

### Deleting the Chart
```bash
$ helm delete YACE -n <namespace>
```

## Tip
As this configuration is read as a Configmap, when an update occurs to the configmap the pod must be restarted to pick 
up the new changes. It may be worthwhile installing Reloader (https://github.com/stakater/Reloader ) - this application 
can be configured to automatically restart pods based off of changes to it's Configmap/Secrets.

