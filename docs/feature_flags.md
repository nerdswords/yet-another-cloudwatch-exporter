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

