{{- define "gateway-service.labels" -}}
app.kubernetes.io/name: {{ .Values.name }}
payment-gateway/release: {{ .Values.name }}
{{- end -}}

{{- define "gateway-service.selector" -}}
app.kubernetes.io/name: {{ .Values.name }}
payment-gateway/release: {{ .Values.name }}
{{- end -}}
