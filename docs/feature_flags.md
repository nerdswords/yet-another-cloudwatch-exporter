# Feature flags

List of features or changes that are disabled by default since they are breaking changes or are considered experimental. Their behavior can change in future releases which will be communicated via the release changelog.

You can enable them using the `-enable-feature` flag with a comma separated list of features. They may be enabled by default in future versions.

## New "associator" algorithm

`-enable-feature=max-dimensions-associator`

Enable a new version of the resource-matching algorithm for discovery jobs.
The associator is the component that matches the output of the `ListMetrics` API response (metrics names and dimensions) to the output of the `GetResources` API response (list of tagged resources).
The new algorithm is intended to fix some odd behaviour where metrics are assigned to the wrong resource name (e.g. this has been reported to happen with ECS and GlobalAccelerator).
Additionally, for some services (e.g. DMS) the default algorithm was reporting metrics where it shouldn't (untagged services appearing just because their ARN would match the auto-discovery regex). This shouldn't happen anymore.

## ListMetrics API result processing

`-enable-feature=list-metrics-callback`

Enables processing of ListMetrics API results page-by-page. This seems to reduce memory usage for high values of `CloudWatchAPIConcurrency`.

## AWS SDK v2

`-enable-feature=aws-sdk-v2`

Uses the v2 version of the aws sdk for go. The sdk v2 version was released in Jan 2021 and is marketed to come with large performance gains. This version offers a drastically different
interface and should be compatible with sdk v2. 

## Always return info metrics

`-enable-feature=always-return-info-metrics`

Return info metrics even if there are no CloudWatch metrics for the resource. This is useful if you want to get a complete picture of your estate, for example if you have some resources which have not yet been used.
