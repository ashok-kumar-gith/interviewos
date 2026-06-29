{{/*
Expand the name of the chart.
*/}}
{{- define "interviewos.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified app name. Truncated at 63 chars for DNS label limits.
*/}}
{{- define "interviewos.fullname" -}}
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
Chart name and version label value.
*/}}
{{- define "interviewos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels.
*/}}
{{- define "interviewos.labels" -}}
helm.sh/chart: {{ include "interviewos.chart" . }}
{{ include "interviewos.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: interviewos
{{- end -}}

{{/*
Selector labels (stable; never change once a workload is deployed).
*/}}
{{- define "interviewos.selectorLabels" -}}
app.kubernetes.io/name: {{ include "interviewos.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Component-scoped names and selector labels. Usage:
  {{ include "interviewos.componentName" (dict "ctx" . "component" "backend") }}
*/}}
{{- define "interviewos.componentName" -}}
{{- printf "%s-%s" (include "interviewos.fullname" .ctx) .component | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "interviewos.componentSelectorLabels" -}}
{{ include "interviewos.selectorLabels" .ctx }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{/*
Names for shared config/secret objects.
*/}}
{{- define "interviewos.configName" -}}
{{- printf "%s-config" (include "interviewos.fullname" .) -}}
{{- end -}}

{{- define "interviewos.secretName" -}}
{{- printf "%s-secrets" (include "interviewos.fullname" .) -}}
{{- end -}}

{{- define "interviewos.postgresName" -}}
{{- printf "%s-postgres" (include "interviewos.fullname" .) -}}
{{- end -}}

{{- define "interviewos.redisName" -}}
{{- printf "%s-redis" (include "interviewos.fullname" .) -}}
{{- end -}}

{{/*
Resolved Redis DSN: explicit override wins, else the in-cluster service.
*/}}
{{- define "interviewos.redisUrl" -}}
{{- if .Values.config.redisUrl -}}
{{- .Values.config.redisUrl -}}
{{- else -}}
{{- printf "redis://%s:%v/0" (include "interviewos.redisName" .) .Values.redis.service.port -}}
{{- end -}}
{{- end -}}

{{/*
Resolved Postgres DSN: explicit override wins, else assembled from creds + the
in-cluster Postgres service.
*/}}
{{- define "interviewos.databaseUrl" -}}
{{- if .Values.secrets.databaseUrl -}}
{{- .Values.secrets.databaseUrl -}}
{{- else -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=disable" .Values.secrets.postgresUser .Values.secrets.postgresPassword (include "interviewos.postgresName" .) .Values.postgres.service.port .Values.secrets.postgresDb -}}
{{- end -}}
{{- end -}}
