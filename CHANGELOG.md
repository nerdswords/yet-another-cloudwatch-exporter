# 0.4.0

* BREAKING: Use list as statistics config option for less api calls
* Start to track changes in CHANGELOG.md
* Increase speed, not only each jobs threaded but now each metric
* Add s3 support
* Fix potential race condition during cloudwatch access
* Fix bug ignoring period in cloudwatch config
* Use interfaces for aws access and prepare code for unit tests
* Implement minimum, average, maximum, sum for cloudwatch api
* Implement way to handle multiple data returned by cloudwatch
* Update go dependencies
