# Feature flags

List of features or changes that are disabled by default since they are breaking changes or are considered experimental. Their behavior can change in future releases which will be communicated via the release changelog.

You can enable them using the `-enable-feature` flag with a comma separated list of features. They may be enabled by default in future versions.

## New resource matching algorithm

`-enable-feature=encoding-resource-associator`

Enabled the new resource matching algorithm, introduced in [#833](https://github.com/nerdswords/yet-another-cloudwatch-exporter/pull/833). The new algorithm fixes some bugs YACE has on discovery jobs, when matching a resource discovered through `resourcetaggingapi` with the corresponding metric, based on the dimensions.
