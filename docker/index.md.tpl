---
title: Service Overview
---

# Services
{{ range $ns, $svcs := .Services }}

## Namespace {{ $ns }}
{{ range $svc := $svcs }}
* {{ $svc.Name }}
{{- range $port := $svc.Spec.Ports }}
{{- if $port.Name }}
  * [{{ $port.Name }}]({{if eq $port.Port 443 }}https{{ else }}http{{end}}://{{ $svc.Spec.ClusterIP }}:{{ $port.Port }})
{{- else }}
  * [default]({{if eq $port.Port 443 }}https{{ else }}http{{end}}://{{ $svc.Spec.ClusterIP }}:{{ $port.Port }})
{{- end -}}
{{- end -}}
{{ end }}
{{ end }}