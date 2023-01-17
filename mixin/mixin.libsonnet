{
  local config = import './config.libsonnet',
  local util = import './util.libsonnet',
  local mixin = (import './dashboards/all.libsonnet') + config,
  grafanaDashboards+::
    {
      [fname]: util.decorate_dashboard(mixin[fname], tags=['cloudwatch-integration']) + { uid: std.md5(fname) }
      for fname in std.objectFields(mixin)
    },

  prometheusAlerts+:: if std.objectHasAll(mixin, 'prometheusAlerts') then mixin.prometheusAlerts else {},
  prometheusRules+:: if std.objectHasAll(mixin, 'prometheusRules') then mixin.prometheusRules else {},
}
