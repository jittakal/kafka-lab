{{/*
Expand the name of the chart.
*/}}
{{- define "kafeventstore.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kafeventstore.fullname" -}}
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
{{- define "kafeventstore.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kafeventstore.labels" -}}
helm.sh/chart: {{ include "kafeventstore.chart" . }}
{{ include "kafeventstore.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.global.environment }}
app.kubernetes.io/environment: {{ .Values.global.environment }}
{{- end }}
{{- if .Values.global.cloudProvider }}
app.kubernetes.io/cloud-provider: {{ .Values.global.cloudProvider }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kafeventstore.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kafeventstore.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kafeventstore.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kafeventstore.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the ConfigMap to use
*/}}
{{- define "kafeventstore.configMapName" -}}
{{- if .Values.configMap.existingConfigMap }}
{{- .Values.configMap.existingConfigMap }}
{{- else }}
{{- include "kafeventstore.fullname" . }}
{{- end }}
{{- end }}

{{/*
Get Kafka credentials secret name
*/}}
{{- define "kafeventstore.kafkaSecretName" -}}
{{- if .Values.secrets.kafka.existingSecret }}
{{- .Values.secrets.kafka.existingSecret }}
{{- else }}
{{- printf "%s-kafka" (include "kafeventstore.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Get AWS credentials secret name
*/}}
{{- define "kafeventstore.awsSecretName" -}}
{{- if .Values.secrets.aws.existingSecret }}
{{- .Values.secrets.aws.existingSecret }}
{{- else }}
{{- printf "%s-aws" (include "kafeventstore.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Get Azure credentials secret name
*/}}
{{- define "kafeventstore.azureSecretName" -}}
{{- if .Values.secrets.azure.existingSecret }}
{{- .Values.secrets.azure.existingSecret }}
{{- else }}
{{- printf "%s-azure" (include "kafeventstore.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Get GCS credentials secret name
*/}}
{{- define "kafeventstore.gcsSecretName" -}}
{{- if .Values.secrets.gcs.existingSecret }}
{{- .Values.secrets.gcs.existingSecret }}
{{- else }}
{{- printf "%s-gcs" (include "kafeventstore.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Get PVC name
*/}}
{{- define "kafeventstore.pvcName" -}}
{{- if .Values.persistence.existingClaim }}
{{- .Values.persistence.existingClaim }}
{{- else }}
{{- include "kafeventstore.fullname" . }}
{{- end }}
{{- end }}

{{/*
Get PV name
*/}}
{{- define "kafeventstore.pvName" -}}
{{- printf "%s-pv" (include "kafeventstore.fullname" .) }}
{{- end }}

{{/*
Return the appropriate apiVersion for HPA
*/}}
{{- define "kafeventstore.hpa.apiVersion" -}}
{{- if semverCompare ">=1.23-0" .Capabilities.KubeVersion.GitVersion -}}
autoscaling/v2
{{- else -}}
autoscaling/v2beta2
{{- end -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for PodDisruptionBudget
*/}}
{{- define "kafeventstore.pdb.apiVersion" -}}
{{- if semverCompare ">=1.21-0" .Capabilities.KubeVersion.GitVersion -}}
policy/v1
{{- else -}}
policy/v1beta1
{{- end -}}
{{- end -}}

{{/*
Return cloud provider specific annotations for service account
*/}}
{{- define "kafeventstore.serviceAccountAnnotations" -}}
{{- if eq .Values.global.cloudProvider "aws" }}
eks.amazonaws.com/role-arn: {{ .Values.serviceAccount.annotations.roleArn | default "" | quote }}
{{- else if eq .Values.global.cloudProvider "azure" }}
azure.workload.identity/client-id: {{ .Values.serviceAccount.annotations.clientId | default "" | quote }}
azure.workload.identity/tenant-id: {{ .Values.serviceAccount.annotations.tenantId | default "" | quote }}
{{- else if eq .Values.global.cloudProvider "gcp" }}
iam.gke.io/gcp-service-account: {{ .Values.serviceAccount.annotations.gcpServiceAccount | default "" | quote }}
{{- end }}
{{- with .Values.serviceAccount.annotations }}
{{- toYaml . }}
{{- end }}
{{- end }}

{{/*
Validate storage backend configuration
*/}}
{{- define "kafeventstore.validateStorage" -}}
{{- $backend := .Values.config.storage.backend }}
{{- if not (or (eq $backend "s3") (eq $backend "azure") (eq $backend "gcs") (eq $backend "file")) }}
{{- fail "config.storage.backend must be one of: s3, azure, gcs, file" }}
{{- end }}
{{- if and (eq $backend "file") (not .Values.persistence.enabled) }}
{{- fail "persistence.enabled must be true when using file storage backend" }}
{{- end }}
{{- end }}

{{/*
Get image with tag
*/}}
{{- define "kafeventstore.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Return the storage class name
*/}}
{{- define "kafeventstore.storageClass" -}}
{{- if .Values.persistence.storageClass }}
{{- .Values.persistence.storageClass }}
{{- else if eq .Values.global.cloudProvider "aws" }}
gp3
{{- else if eq .Values.global.cloudProvider "azure" }}
managed-premium
{{- else if eq .Values.global.cloudProvider "gcp" }}
standard-rwo
{{- else }}
{{- "" }}
{{- end }}
{{- end }}
