# Purpose of the clients package
The goal of this package is to abstract away as much of the aws sdk implementation details as possible. Maintaining this level
of abstraction allows us to swap between the [AWS SDK for Go v1](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/welcome.html) and [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/)
without needing to make large changes to the metric scraping process.

The folder structure is intended to isolate common interfaces away from v1 and v2 to specific implementations

```
/clients: cache interface required by the library entry point
/clients/v1
/clients/v2
/clients/account: yace specific account interface required to lookup aws account fino
/clients/account/v1
/clients/account/v2
/clients/cloudwatch: yace specific cloudwatch interface required to gather cloudwatch metrics data
/clients/cloudwatch/v1
/clients/cloudwatch/v2
/clients/tagging: yace specific tagging interface used to discover resources via a wide set of aws apis
/clients/tagging/v1
/clients/tagging/v2
```

sdk v1 and v2 are just different go implementations on top of AWS's defined APIs. They do not call different AWS APIs merely
swap between different abstractions on top of those APIs. Since the abstractions are largely different but the APIs are the same
the differences in implementation are largely superficial.

# Ensuring v1 and v2 stay in sync
Since we are in a transition state from sdk v1 to sdk v2 it's very important that we maintain as much feature parity as possible
between the two implementations. Factory/client interface changes are relatively easy to keep in sync since changes need to be
made to v1 and v2 implementations. Other areas are not quite so obvious such as

## /clients/tagging/v_/filters.go serviceFilters

`serviceFilters` are extra definitions for how to lookup or filter resources for certain cloudwatch namespaces which cannot be done
using only tag data alone.

Due to the major differences in how aws sdk v1 and v2 behave, it is not easy to have both share an implementation of `serviceFilter`.
Any changes which are made to a serviceFilter implementation in on version MUST be ported to the other version. This includes,

* Adding a service filter implementation for a new service
* Modifying the behavior of a `ResourceFunc`
* Modifying the behavior of a `FilterFunc`

If this becomes a large burden it might be possible to create a shared abstraction which is more independent of client version

## /clients/cloudwatch/v_/input.go

The functions in these two files govern the API requests made to CloudWatch it is imperative that the inputs for requests to v1
and APIs v2 stay in sync at all times. 
