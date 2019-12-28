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
Render chart metadata labels: "chart", "heritage" unless "openmatch.noChartMeta" is set.
*/}}
{{- define "openmatch.chartmeta" -}}
{{- if not .Values.noChartMeta -}}
chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
heritage: {{ .Release.Service }}
{{- end }}
{{- end -}}

{{- define "prometheus.annotations" -}}
{{- if and (.prometheus.serviceDiscovery) (.prometheus.enabled) -}}
prometheus.io/scrape: "true"
prometheus.io/port: {{ .port | quote }}
prometheus.io/path: {{ .prometheus.endpoint }}
{{- end -}}
{{- end -}}

{{- define "openmatch.container.common" -}}
imagePullPolicy: {{ .Values.global.image.pullPolicy }}
resources:
{{- toYaml .Values.global.kubernetes.resources | nindent 2 }}
{{- end -}}

{{- define "openmatch.volumemounts.configs" -}}
{{- range $configIndex, $configValues := .configs }}
- name: {{ $configValues.volumeName }}
  mountPath: {{ $configValues.mountPath }}
{{- end }}
{{- end -}}

{{- define "openmatch.volumes.configs" -}}
{{- range $configIndex, $configValues := .configs }}
- name: {{ $configValues.volumeName }}
  configMap:
    name: {{ $configValues.configName }}
{{- end }}
{{- end -}}

{{- define "openmatch.volumemounts.tls" -}}
{{- if .Values.global.tls.enabled }}
- name: tls-server-volume
  mountPath: {{ .Values.global.tls.server.mountPath }}
- name: root-ca-volume
  mountPath: {{ .Values.global.tls.rootca.mountPath }}
{{- end -}}
{{- end -}}

{{- define "openmatch.volumes.tls" -}}
{{- if .Values.global.tls.enabled }}
- name: tls-server-volume
  secret:
    secretName: om-tls-server
- name: root-ca-volume
  secret:
    secretName: om-tls-rootca
{{- end -}}
{{- end -}}

{{- define "openmatch.volumemounts.withredis" -}}
{{- if .Values.redis.usePassword }}
- name: redis-password
  mountPath: {{ .Values.redis.secretMountPath }}
{{- end -}}
{{- end -}}

{{- define "openmatch.volumes.withredis" -}}
{{- if .Values.redis.usePassword }}
- name: redis-password
  secret:
    secretName: {{ .Values.redis.fullnameOverride }}
{{- end -}}
{{- end -}}

{{- define "openmatch.labels.nodegrouping" -}}
{{- if .Values.global.kubernetes.affinity }}
affinity:
{{ toYaml .Values.global.kubernetes.affinity | nindent 2 }}
{{- end }}
{{- if .Values.global.kubernetes.nodeSelector }}
nodeSelector:
{{ toYaml .Values.global.kubernetes.nodeSelector | nindent 2 }}
{{- end }}
{{- if .Values.global.kubernetes.tolerations }}
tolerations:
{{ toYaml .Values.global.kubernetes.tolerations | nindent 2 }}
{{- end }}
{{- end -}}

{{- define "kubernetes.probe" -}}
livenessProbe:
  httpGet:
    scheme: {{ if (.isHTTPS) }}HTTPS{{ else }}HTTP{{ end }}
    path: /healthz
    port: {{ .port }}
  initialDelaySeconds: 10
  periodSeconds: 10
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

{{- define "openmatch.HorizontalPodAutoscaler.spec.common" -}}
minReplicas: {{ .Values.global.kubernetes.horizontalPodAutoScaler.minReplicas }}
maxReplicas: {{ .Values.global.kubernetes.horizontalPodAutoScaler.maxReplicas }}
targetCPUUtilizationPercentage: {{ .Values.global.kubernetes.horizontalPodAutoScaler.targetCPUUtilizationPercentage }}
{{- end -}}
