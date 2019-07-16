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
imagePullPolicy: {{ .Values.image.pullPolicy }}
resources:
  requests:
    memory: 200Mi
    cpu: 200m
volumeMounts:
- name: {{ .Values.config.volumeName }}
  mountPath: {{ .Values.config.mountPath }}
{{- if .Values.global.tls.enabled }}
- name: root-ca-volume
  mountPath: {{ .Values.global.tls.root.mountPath }}
- name: tls-server-volume
  mountPath: {{ .Values.global.tls.server.mountPath }}
{{- end }}
{{- end -}}

{{- define "openmatch.spec.common" -}}
volumes:
- name: {{ .Values.config.volumeName }}
  configMap:
    name: {{ .Values.config.mapName }}
{{- if .Values.global.tls.enabled }}
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
