{{/*
Expand the name of the chart.
*/}}
{{- define "kafeventconsumer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kafeventconsumer.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kafeventconsumer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kafeventconsumer.labels" -}}
helm.sh/chart: {{ include "kafeventconsumer.chart" . }}
{{ include "kafeventconsumer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.global.environment }}
app.kubernetes.io/environment: {{ .Values.global.environment }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kafeventconsumer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kafeventconsumer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kafeventconsumer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kafeventconsumer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate namespace
*/}}
{{- define "kafeventconsumer.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace -}}
{{- end }}
