{{- define "keda-otel-collector-fleet-scaler.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.fullname" -}}
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

{{- define "keda-otel-collector-fleet-scaler.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "keda-otel-collector-fleet-scaler.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.profile" -}}
{{- $p := .Values.fleet.profile | default "forward" -}}
{{- if eq $p "stateful" -}}stateful{{- else -}}forward{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.backendKind" -}}
{{- $kind := .Values.fleet.backend.kind -}}
{{- if $kind -}}
{{- $kind -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
StatefulSet
{{- else -}}
Deployment
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.preStopSeconds" -}}
{{- $v := .Values.fleet.backend.preStopSeconds | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
35
{{- else -}}
15
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.terminationGrace" -}}
{{- $v := .Values.fleet.backend.terminationGracePeriodSeconds | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
60
{{- else -}}
30
{{- end -}}
{{- end -}}
