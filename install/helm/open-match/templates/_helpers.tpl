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
Instead of .Chart.Name, we hard-code "open-match" as we need to call this from subcharts, but get the
same result as if called from this chart.
*/}}
{{- define "openmatch.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default "open-match" .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

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
    name: {{ tpl $configValues.configName $ }}
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
    secretName: {{ include "openmatch.fullname" . }}-tls-server
- name: root-ca-volume
  secret:
    secretName: {{ include "openmatch.fullname" . }}-tls-rootca
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
    secretName: {{ include "call-nested" (list . "redis" "redis.fullname") }}
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

{{- define "openmatch.serviceAccount.name" -}}
{{- .Values.global.kubernetes.serviceAccount | default (printf "%s-unprivileged-service" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.swaggerui.hostName" -}}
{{- .Values.swaggerui.hostName | default (printf "%s-swaggerui" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.query.hostName" -}}
{{- .Values.query.hostName | default (printf "%s-query" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.frontend.hostName" -}}
{{- .Values.frontend.hostName | default (printf "%s-frontend" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.backend.hostName" -}}
{{- .Values.backend.hostName | default (printf "%s-backend" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.synchronizer.hostName" -}}
{{- .Values.synchronizer.hostName | default (printf "%s-synchronizer" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.evaluator.hostName" -}}
{{- .Values.evaluator.hostName | default (printf "%s-evaluator" (include "openmatch.fullname" . ) ) -}}
{{- end -}}

{{- define "openmatch.configmap.default" -}}
{{- printf "%s-configmap-default" (include "openmatch.fullname" . ) -}}
{{- end -}}

{{- define "openmatch.configmap.override" -}}
{{- printf "%s-configmap-override" (include "openmatch.fullname" . ) -}}
{{- end -}}

{{- define "openmatch.jaeger.agent" -}}
{{- if index .Values "open-match-telemetry" "enabled" -}}
{{- if index .Values "open-match-telemetry" "jaeger" "enabled" -}}
{{ include "call-nested" (list . "open-match-telemetry.jaeger" "jaeger.agent.name") }}:6831
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "openmatch.jaeger.collector" -}}
{{- if index .Values "open-match-telemetry" "enabled" -}}
{{- if index .Values "open-match-telemetry" "jaeger" "enabled" -}}
http://{{ include "call-nested" (list . "open-match-telemetry.jaeger" "jaeger.collector.name") }}:14268/api/traces
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Call templates from sub-charts in a synthesized context, workaround for https://github.com/helm/helm/issues/3920
Mainly useful for things like `{{ include "call-nested" (list . "redis" "redis.fullname") }}`
https://github.com/helm/helm/issues/4535#issuecomment-416022809
https://github.com/helm/helm/issues/4535#issuecomment-477778391
*/}}
{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 | splitList "." }}
{{- $template := index . 2 }}
{{- $values := $dot.Values }}
{{- range $subchart }}
{{- $values = index $values . }}
{{- end }}
{{- include $template (dict "Chart" (dict "Name" (last $subchart)) "Values" $values "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end }}