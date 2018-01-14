---
title: Service Overview
---

{{ $defaultIngressHost := .DefaultIngressHost }}

# Ingresses

{{ range $ns, $ingresses := .Ingresses -}}
{{- range $ingress := $ingresses }}
* {{ $ingress.Name }}
{{- range $rule := $ingress.Spec.Rules }}
{{- range $rulePath := $rule.HTTP.Paths }}
  * [{{ $rulePath.Backend.ServiceName }}:{{ $rulePath.Backend.ServicePort }}]({{if $ingress.Spec.TLS }}https{{else}}http{{end}}://{{if $rule.Host }}{{ $rule.Host }}{{ else }}{{ $defaultIngressHost }}{{end}}{{ $rulePath.Path }})
{{- end -}}
{{- end -}}
{{- end -}}
{{ end }}


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