{{/*
Expand the name of the chart.
*/}}
{{- define "go-cli-starter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "go-cli-starter.fullname" -}}
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
{{- define "go-cli-starter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "go-cli-starter.labels" -}}
helm.sh/chart: {{ include "go-cli-starter.chart" . }}
{{ include "go-cli-starter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: server
app.kubernetes.io/part-of: {{ include "go-cli-starter.name" . }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "go-cli-starter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "go-cli-starter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "go-cli-starter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "go-cli-starter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Resolve the container image reference. Falls back to .Chart.AppVersion when image.tag is empty.
*/}}
{{- define "go-cli-starter.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end }}

{{/*
Validate a seccompProfile block. Fails the install with a readable error when the profile is misconfigured.
Usage: {{- include "go-cli-starter.validateSeccompProfile" (dict "profile" .Values.podSecurityContext.seccompProfile "scope" "podSecurityContext") -}}
*/}}
{{- define "go-cli-starter.validateSeccompProfile" -}}
{{- $profile := .profile -}}
{{- $scope := .scope -}}
{{- if $profile -}}
{{- $type := $profile.type | default "" -}}
{{- if not (has $type (list "RuntimeDefault" "Localhost" "Unconfined")) -}}
{{- fail (printf "%s.seccompProfile.type must be one of RuntimeDefault, Localhost, Unconfined (got %q)" $scope $type) -}}
{{- end -}}
{{- if and (eq $type "Localhost") (not $profile.localhostProfile) -}}
{{- fail (printf "%s.seccompProfile.type=Localhost requires .localhostProfile to be set" $scope) -}}
{{- end -}}
{{- if and (ne $type "Localhost") $profile.localhostProfile -}}
{{- fail (printf "%s.seccompProfile.localhostProfile is only valid when type=Localhost (got type=%q)" $scope $type) -}}
{{- end -}}
{{- end -}}
{{- end }}

{{/*
Render the chart-managed Secret name (used for envFrom wiring when secret.create=true).
*/}}
{{- define "go-cli-starter.secretName" -}}
{{- printf "%s-secret" (include "go-cli-starter.fullname" .) -}}
{{- end }}

{{/*
Render the chart-managed ConfigMap name.
*/}}
{{- define "go-cli-starter.configmapName" -}}
{{- printf "%s-config" (include "go-cli-starter.fullname" .) -}}
{{- end }}
