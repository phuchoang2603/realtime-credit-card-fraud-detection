{{- define "payment-gateway.selector" -}}
app.kubernetes.io/name: {{ . }}
payment-gateway/release: {{ . }}
{{- end -}}
