# Feature flags

List of features or changes that are disabled by default since they are breaking changes or are considered experimental. Their behavior can change in future releases which will be communicated via the release changelog.

You can enable them using the `-enable-feature` flag with a comma separated list of features. They may be enabled by default in future versions.

## ListMetrics API result processing

`-enable-feature=list-metrics-callback`

Enables processing of ListMetrics API results page-by-page. This seems to reduce memory usage for high values of `CloudWatchAPIConcurrency`.
