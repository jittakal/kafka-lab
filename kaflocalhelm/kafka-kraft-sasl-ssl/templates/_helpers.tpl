{{/*
Expand the name of the chart.
*/}}
{{- define "kafka-kraft-sasl-ssl.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kafka-kraft-sasl-ssl.fullname" -}}
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
{{- define "kafka-kraft-sasl-ssl.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kafka-kraft-sasl-ssl.labels" -}}
helm.sh/chart: {{ include "kafka-kraft-sasl-ssl.chart" . }}
{{ include "kafka-kraft-sasl-ssl.selectorLabels" . }}
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
{{- define "kafka-kraft-sasl-ssl.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kafka-kraft-sasl-ssl.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kafka-kraft-sasl-ssl.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kafka-kraft-sasl-ssl.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Kafka server name
*/}}
{{- define "kafka-kraft-sasl-ssl.server.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-server
{{- end }}

{{/*
Kafka data volume name
*/}}
{{- define "kafka-kraft-sasl-ssl.data.volume.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-data
{{- end }}

{{/*
Kafka secrets volume name
*/}}
{{- define "kafka-kraft-sasl-ssl.secrets.volume.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-secrets
{{- end }}

{{/*
Kafka client secrets volume name
*/}}
{{- define "kafka-kraft-sasl-ssl.client.secrets.volume.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-client-secrets
{{- end }}

{{/*
Kafka config volume name
*/}}
{{- define "kafka-kraft-sasl-ssl.config.volume.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-config
{{- end }}

{{/*
Kafka shared data volume name
*/}}
{{- define "kafka-kraft-sasl-ssl.shared.data.volume.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-shared-data
{{- end }}

{{/*
Kafka TLS certificates volume name
*/}}
{{- define "kafka-kraft-sasl-ssl.tls.certificates.volume.name" -}}
{{ include "kafka-kraft-sasl-ssl.fullname" . }}-tls-certificates
{{- end }}

{{/*
Generate controller quorum voters for KRaft mode
Format: 1@kafka-0.kafka-hs.namespace.svc.cluster.local:29092,2@kafka-1...
*/}}
{{- define "kafka-kraft-sasl-ssl.controller.quorum.voters" -}}
{{- $replicationCount := int .Values.replicaCount -}}
{{- $controllerPort := int .Values.controller.service.port -}}
{{- $fullname := include "kafka-kraft-sasl-ssl.fullname" . -}}
{{- $namespace := .Release.Namespace -}}
{{- $voters := list -}}
{{- range $i := until $replicationCount }}
{{- $voters = append $voters (printf "%d@%s-%d.%s-hs.%s.svc.cluster.local:%d" (add $i 1) $fullname $i $fullname $namespace $controllerPort) -}}
{{- end -}}
{{- join "," $voters -}}
{{- end }}

{{/*
Generate headless service CNAME
*/}}
{{- define "kafka-kraft-sasl-ssl.cname" -}}
{{- $fullname := include "kafka-kraft-sasl-ssl.fullname" . -}}
{{- $namespace := .Release.Namespace -}}
{{- printf "%s-hs.%s.svc.cluster.local" $fullname $namespace -}}
{{- end }}

{{/*
Generate DNS names for TLS certificates
*/}}
{{- define "kafka-kraft-sasl-ssl.dnsnames" -}}
{{- $replicationCount := int .Values.replicaCount -}}
{{- $fullname := include "kafka-kraft-sasl-ssl.fullname" . -}}
{{- $namespace := .Release.Namespace -}}
{{- $dnsNames := list -}}
{{- $dnsNames = append $dnsNames (printf "%s-hs.%s.svc.cluster.local" $fullname $namespace) -}}
{{- $dnsNames = append $dnsNames (printf "*.%s-hs.%s.svc.cluster.local" $fullname $namespace) -}}
{{- range $i := until $replicationCount }}
{{- $dnsNames = append $dnsNames (printf "%s-%d.%s-hs.%s.svc.cluster.local" $fullname $i $fullname $namespace) -}}
{{- end -}}
{{- $dnsNames = append $dnsNames "localhost" -}}
{{- range $dnsName := $dnsNames }}
- {{ $dnsName | quote }}
{{- end -}}
{{- end }}

{{/*
Generate advertised listeners for Kafka brokers
*/}}
{{- define "kafka-kraft-sasl-ssl.advertised.listeners" -}}
{{- $fullname := include "kafka-kraft-sasl-ssl.fullname" . -}}
{{- $namespace := .Release.Namespace -}}
{{- $brokerPort := int .Values.broker.service.port -}}
{{- $hostPort := int .Values.broker.service.hostPort -}}
{{- printf "SSL-INTERNAL://$(POD_NAME).%s-hs.%s.svc.cluster.local:%d,SSL://localhost:%d" $fullname $namespace $brokerPort $hostPort -}}
{{- end }}

{{/*
Generate namespace
*/}}
{{- define "kafka-kraft-sasl-ssl.namespace" -}}
{{- default .Release.Namespace .Values.global.namespace -}}
{{- end }}

{{/*
JVM Heap Options
*/}}
{{- define "kafka-kraft-sasl-ssl.jvm.heap" -}}
{{- if .Values.resources.limits.memory -}}
{{- $memory := .Values.resources.limits.memory -}}
{{- $memoryGi := regexFind "[0-9]+" $memory | int -}}
{{- $heapGi := div (mul $memoryGi 75) 100 -}}
{{- printf "-Xmx%dG -Xms%dG" $heapGi $heapGi -}}
{{- else -}}
-Xmx2G -Xms2G
{{- end -}}
{{- end }}
