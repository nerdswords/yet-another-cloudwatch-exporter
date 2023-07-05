# CloudWatch Mixin

This is a Prometheus [Monitoring Mixin](https://monitoring.mixins.dev/) that comes with pre-defined dashboards.

It can be installed e.g. with [Grizzly](https://grafana.github.io/grizzly).

First, install [jsonnet-bundler](https://github.com/jsonnet-bundler/jsonnet-bundler) with

```
go install -a github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest
```

Then install all the dependencies of this mixin:

```
jb install
```

Finally, install `Grizzly` and apply the mixin to your Grafana instance:

```
go install github.com/grafana/grizzly/cmd/grr@latest
grr apply mixin.libsonnet
```
