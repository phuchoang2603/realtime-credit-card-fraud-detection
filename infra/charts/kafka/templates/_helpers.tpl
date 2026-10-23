{{- define "kafka.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kafka.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "kafka.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "kafka.image" -}}
{{- $root := index . 0 -}}
{{- $image := index . 1 -}}
{{- $tagOverride := "" -}}
{{- if gt (len .) 2 -}}
{{- $tagOverride = index . 2 | default "" -}}
{{- end -}}
{{- if $root.Values.global.imageRegistry -}}
{{- $tag := $tagOverride | default $root.Values.global.imageTag | default "" -}}
{{- if $tag -}}
{{- printf "%s/%s:%s" $root.Values.global.imageRegistry $image $tag -}}
{{- else -}}
{{- printf "%s/%s" $root.Values.global.imageRegistry $image -}}
{{- end -}}
{{- else -}}
{{- $image -}}
{{- end -}}
{{- end -}}
