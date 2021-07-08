{{/*
Expand the name of the chart.
*/}}
{{- define "firefly.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "firefly.fullname" -}}
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
{{- define "firefly.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "firefly.coreLabels" -}}
helm.sh/chart: {{ include "firefly.chart" . }}
{{ include "firefly.coreSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kuberentes.io/part-of: {{ .Chart.Name }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "firefly.dataexchangeLabels" -}}
helm.sh/chart: {{ include "firefly.chart" . }}
{{ include "firefly.dataexchangeSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kuberentes.io/part-of: {{ .Chart.Name }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "firefly.coreSelectorLabels" -}}
app.kubernetes.io/name: {{ include "firefly.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: core
{{- end }}

{{/*
Selector labels
*/}}
{{- define "firefly.dataexchangeSelectorLabels" -}}
app.kubernetes.io/name: {{ include "firefly.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: dx
{{- end }}

{{- define "firefly.dataexchangeP2PHost" -}}
{{- if .Values.dataexchange.ingress.enabled }}
{{- index .Values.dataexchange.ingress.hosts 0 }}
{{- else }}
{{- printf "%s-dx.%s.svc:%d" (include "firefly.fullname" .) .Release.Namespace (.Values.dataexchange.service.p2pPort | int64) }}
{{- end }}
{{- end }}

{{- define "firefly.coreConfig" -}}
{{- if .Values.config.debugEnabled }}
log:
  level: debug
debug:
  port: {{ .Values.core.service.debugPort }}
{{- end }}
http:
  port: {{ .Values.core.service.httpPort }}
  address: 0.0.0.0
admin:
  port:  {{ .Values.core.service.adminPort }}
  address: 0.0.0.0
  enabled: {{ .Values.config.adminEnabled }}
  preinit: {{ and .Values.config.adminEnabled .Values.config.preInit }}
ui:
  path: ./frontend
org:
  name: {{ .Values.config.organizationName }}
  identity: {{ .Values.config.organizationIdentity }}
{{- if .Values.config.blockchainOverride }}
blockchain:
  {{- toYaml (tpl .Values.config.blockchainOverride .) | nindent 2 }}
{{- else if .Values.config.ethconnectUrl }}
blockchain:
  type: ethereum
  ethereum:
    ethconnect:
      url: {{ tpl .Values.config.ethconnectUrl . }}
      instance: {{ .Values.config.fireflyContractAddress }}
      topic: "0" # TODO should likely be configurable
      {{- if and .Values.config.ethconnectUsername .Values.config.ethconnectPassword }}
      auth:
        username: {{ .Values.config.ethconnectUsername | quote }}
        password: {{ .Values.config.ethconnectPassword | quote }}
      {{- end }}
      {{- if .Values.config.ethconnectPrefixShort }}
      prefixShort: {{ .Values.config.ethconnectPrefixShort }}
      {{- end }}
      {{- if .Values.config.ethconnectPrefixLong }}
      prefixLong: {{ .Values.config.ethconnectPrefixLong }}
      {{- end }}
{{- end }}
{{- if .Values.config.databaseOverride }}
database:
  {{- toYaml (tpl .Values.config.databaseOverride .) | nindent 2 }}
{{- else if .Values.config.postgresUrl }}
database:
  type: postgres
  postgres:
    url: {{ tpl .Values.config.postgresUrl . }}
    migrations:
      auto: {{ .Values.config.postgresAutomigrate }}
{{- end }}
{{- if .Values.config.publicstorageOverride }}
publicstorage:
  {{- toYaml (tpl .Values.config.publicstorageOverride .) | nindent 2 }}
{{- else if and .Values.config.ipfsApiUrl .Values.config.ipfsGatewayUrl }}
publicstorage:
  type: ipfs
  ipfs:
    api:
      url: {{ tpl .Values.config.ipfsApiUrl . }}
      {{- if and .Values.config.ipfsApiUsername .Values.config.ipfsApiPassword }}
      auth:
        username: {{ .Values.config.ipfsApiUsername |quote }}
        password:  {{ .Values.config.ipfsApiPassword | quote }}
      {{- end }}
    gateway:
      url: {{ tpl .Values.config.ipfsGatewayUrl . }}
      {{- if and .Values.config.ipfsGatewayUsername .Values.config.ipfsGatewayPassword }}
      auth:
        username: {{ .Values.config.ipfsGatewayUsername |quote }}
        password: {{ .Values.config.ipfsGatewayPassword | quote }}
      {{- end }}
{{- end }}
{{- if and .Values.config.dataexchangeOverride (not .Values.dataexchange.enabled) }}
dataexchange:
  {{- toYaml (tpl .Values.config.dataexchangeOverride .) | nindent 2 }}
{{- else }}
dataexchange:
  {{- if .Values.dataexchange.enabled }}
  https:
    url: http://{{ include "firefly.fullname" . }}-dx.{{ .Release.Namespace }}.svc:{{ .Values.dataexchange.service.apiPort }}
    {{- if .Values.dataexchange.apiKey }}
    headers:
      x-api-key: {{ .Values.dataexchange.apiKey | quote }}
    {{- end }}
  {{- else }}
  https:
    url: {{ tpl .Values.config.dataexchangeUrl . }}
    {{- if .Values.config.dataexchangeAPIKey }}
    headers:
      x-api-key: {{ .Values.config.dataexchangeAPIKey | quote }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}