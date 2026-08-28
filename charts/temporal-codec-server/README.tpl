{{ template "chart.header" . }}
{{ template "chart.deprecationWarning" . }}

{{ template "chart.badgesSection" . }}

{{ template "chart.description" . }}

## Installing the Chart

```shell
helm upgrade \
  --cleanup-on-fail \
  --create-namespace \
  --install \
  --namespace codec \
  --reset-then-reuse-values \
  --wait \
  temporal-codec-server oci://ghcr.io/mrsimonemms/charts/{{ template "chart.name" . }}
```

{{ template "chart.homepageLine" . }}

{{ template "chart.maintainersSection" . }}

{{ template "chart.sourcesSection" . }}

{{ template "chart.requirementsSection" . }}

{{ template "chart.valuesSection" . }}

{{ template "helm-docs.versionFooter" . }}
