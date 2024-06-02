{{/*
ServiceAccountName - GatewayAPI
*/}}
{{- define "fsm.gateway.serviceAccountName" -}}
{{ printf "fsm-gateway-%s" .Values.fsm.gateway.namespace }}
{{- end }}

{{/* fsm-gateway image */}}
{{- define "fsmGateway.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmGateway .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmGateway .Values.fsm.image.digest.fsmGateway -}}
{{- end -}}
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "fsm.labels" -}}
app.kubernetes.io/name: flomesh.io
app.kubernetes.io/instance: {{ .Values.fsm.meshName }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end -}}

{{/* fsm-curl image */}}
{{- define "fsmCurl.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmCurl .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmCurl .Values.fsm.image.digest.fsmCurl -}}
{{- end -}}
{{- end -}}
