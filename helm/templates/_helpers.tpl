{{/*
Expand the helper functions for the fatidentitycat
*/}}

{{- define "fatidentitycat.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "fatidentitycat.labels" -}}
app: {{ .Chart.Name }}
release: {{ .Release.Name }}
{{- end -}}

{{- define "fatidentitycat.app-image" -}}
{{ .Values.app.image.repository }}:{{ .Values.app.image.tag | default .Chart.AppVersion }}
{{- end -}}
