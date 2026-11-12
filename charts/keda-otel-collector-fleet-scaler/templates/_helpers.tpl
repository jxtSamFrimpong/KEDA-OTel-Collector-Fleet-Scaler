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

{{/*
Fleet resources live in Release.Namespace by default. Set fleet.namespace only
when intentionally placing gateway/backends outside the release namespace.
*/}}
{{- define "keda-otel-collector-fleet-scaler.fleetNamespace" -}}
{{- if and .Values.fleet.namespace (ne .Values.fleet.namespace "") -}}
{{- .Values.fleet.namespace -}}
{{- else -}}
{{- .Release.Namespace -}}
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.gatewayName" -}}
{{- default "otel-gateway" .Values.fleet.gateway.name -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.backendName" -}}
{{- default "otel-backends" .Values.fleet.backend.name -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.gatewayServiceAccountName" -}}
{{- default (include "keda-otel-collector-fleet-scaler.gatewayName" .) .Values.fleet.gateway.serviceAccountName -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scalerAddress" -}}
{{- printf "%s.%s.svc:%v" (include "keda-otel-collector-fleet-scaler.fullname" .) .Release.Namespace .Values.service.grpcPort -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.backendResolverService" -}}
{{- printf "%s.%s" (include "keda-otel-collector-fleet-scaler.backendName" .) (include "keda-otel-collector-fleet-scaler.fleetNamespace" .) -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.labels" -}}
app.kubernetes.io/name: {{ include "keda-otel-collector-fleet-scaler.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.gatewayLabels" -}}
app.kubernetes.io/name: {{ include "keda-otel-collector-fleet-scaler.gatewayName" . }}
app.kubernetes.io/component: gateway
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ include "keda-otel-collector-fleet-scaler.name" . }}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.backendLabels" -}}
app.kubernetes.io/name: {{ include "keda-otel-collector-fleet-scaler.backendName" . }}
app.kubernetes.io/component: backend
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ include "keda-otel-collector-fleet-scaler.name" . }}
keda.otel.collector-fleet.scaler/fleet-profile: {{ include "keda-otel-collector-fleet-scaler.profile" . }}
{{- end -}}

{{/* ScaledObject HPA behavior defaults from profiles.yaml */}}
{{- define "keda-otel-collector-fleet-scaler.cooldownPeriod" -}}
{{- $v := .Values.fleet.scaledObject.cooldownPeriod | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
300
{{- else -}}
60
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scaleUpStabilization" -}}
{{- $v := .Values.fleet.scaledObject.scaleUpStabilizationSeconds | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
60
{{- else -}}
30
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scaleDownStabilization" -}}
{{- $v := .Values.fleet.scaledObject.scaleDownStabilizationSeconds | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
300
{{- else -}}
60
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scaleUpPods" -}}
{{- $v := .Values.fleet.scaledObject.scaleUpPodsPerPeriod | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
1
{{- else -}}
2
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scaleUpPeriod" -}}
{{- $v := .Values.fleet.scaledObject.scaleUpPeriodSeconds | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
120
{{- else -}}
60
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scaleDownPods" -}}
{{- $v := .Values.fleet.scaledObject.scaleDownPodsPerPeriod | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else -}}
1
{{- end -}}
{{- end -}}

{{- define "keda-otel-collector-fleet-scaler.scaleDownPeriod" -}}
{{- $v := .Values.fleet.scaledObject.scaleDownPeriodSeconds | int -}}
{{- if gt $v 0 -}}
{{- $v -}}
{{- else if eq (include "keda-otel-collector-fleet-scaler.profile" .) "stateful" -}}
180
{{- else -}}
60
{{- end -}}
{{- end -}}
