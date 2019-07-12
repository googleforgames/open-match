{*
 Copyright 2019 Google LLC

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*}

{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "openmatch.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "openmatch.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "openmatch.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Render chart metadata labels unless "openmatch.noChartMeta" is set.
Expects to find in a scope a field "indent" with an integer value to pass to function "nindent".
*/}}
{{- define "openmatch.chartmeta" -}}
{{- if not .Values.openmatch.noChartMeta -}}
{{- include "openmatch.chartmetalabels" . | nindent .indent }}
{{- end }}
{{- end -}}

{{/*
Print chart metadata labels: "chart", "release", "heritage".
*/}}
{{- define "openmatch.chartmetalabels" -}}
chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
release: {{ .Release.Name }}
heritage: {{ .Release.Service }}
{{- end -}}


{{- define "prometheus.annotations" -}}
{{- if and (.prometheus.serviceDiscovery) (.prometheus.enabled) -}}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .port | quote }}
prometheus.io/path: {{ .prometheus.endpoint }}
{{- end -}}
{{- end -}}

{{- define "openmatch.container.common" -}}
imagePullPolicy: {{ .Values.openmatch.image.pullPolicy }}
resources:
  requests:
    memory: 100Mi
    cpu: 100m
volumeMounts:
- name: om-config-volume
  mountPath: {{ .Values.openmatch.config.mountPath }}
{{- if .Values.openmatch.tls.enabled }}
- name: root-ca-volume
  mountPath: {{ .Values.openmatch.tls.root.mountPath }}
- name: tls-server-volume
  mountPath: {{ .Values.openmatch.tls.server.mountPath }}
{{- end }}
{{- end -}}

{{- define "openmatch.spec.common" -}}
serviceAccountName: {{ .Values.openmatch.kubernetes.serviceAccount }}
volumes:
- name: om-config-volume
  configMap:
    name: om-configmap
{{- if .Values.openmatch.tls.enabled }}
- name: root-ca-volume
  secret:
    secretName: om-tls-rootca
- name: tls-server-volume
  secret:
    secretName: om-tls-server
{{- end }}
{{- end -}}

{{- define "openmatch.container.withredis" -}}
env:
- name: REDIS_SERVICE_HOST
  value: "$(OM_REDIS_MASTER_SERVICE_HOST)"
- name: REDIS_SERVICE_PORT
  value: "$(OM_REDIS_MASTER_SERVICE_PORT)"
{{- if .Values.redis.usePassword }}
- name: REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.redis.fullnameOverride }}
      key: redis-password
{{- end}}
{{- end -}}

{{- define "kubernetes.probe" -}}
livenessProbe:
  httpGet:
    scheme: {{ if (.isHTTPS) }}HTTPS{{ else }}HTTP{{ end }}
    path: /healthz
    port: {{ .port }}
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
readinessProbe:
  httpGet:
    scheme: {{ if (.isHTTPS) }}HTTPS{{ else }}HTTP{{ end }}
    path: /healthz?readiness=true
    port: {{ .port }}
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 2
{{- end -}}
