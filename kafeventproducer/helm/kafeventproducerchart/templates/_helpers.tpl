{{/*
Expand the name of the chart.
*/}}
{{- define "kafeventproducer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "kafeventproducer.fullname" -}}
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
{{- define "kafeventproducer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kafeventproducer.labels" -}}
helm.sh/chart: {{ include "kafeventproducer.chart" . }}
{{ include "kafeventproducer.selectorLabels" . }}
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
{{- define "kafeventproducer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kafeventproducer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kafeventproducer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kafeventproducer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate namespace
*/}}
{{- define "kafeventproducer.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace -}}
{{- end }}
