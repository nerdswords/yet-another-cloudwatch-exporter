{
  decorate_dashboard(dashboard, tags, refresh='30s', timeFrom='now-30m')::
    dashboard {
      editable: false,
      id: null,  // If id is set the grafana client will try to update instead of create
      tags: tags,
      refresh: refresh,
      time: {
        from: timeFrom,
        to: 'now',
      },
      templating: {
        list+: [
          if std.objectHas(t, 'query') && t.query == 'prometheus' then t { regex: '(?!grafanacloud-usage|grafanacloud-ml-metrics).+' } else t
          for t in dashboard.templating.list
        ],
      },
    },
}
