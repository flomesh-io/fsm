{{/* Determine fsm namespace */}}
{{- define "fsm.namespace" -}}
{{ default .Release.Namespace .Values.fsm.fsmNamespace}}
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "fsm.labels" -}}
app.kubernetes.io/name: flomesh.io
app.kubernetes.io/instance: {{ .Values.fsm.meshName }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end -}}

{{/* Security context values that ensure restricted access to host resources */}}
{{- define "restricted.securityContext" -}}
securityContext:
    runAsUser: 1000
    runAsGroup: 3000
    fsGroup: 2000
    supplementalGroups: [5555]
{{- end -}}

{{/* fsm-conector image */}}
{{- define "fsmConnector.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmConnector .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmConnector .Values.fsm.image.digest.fsmInjector -}}
{{- end -}}
{{- end -}}

{{/* fsm connector's provider */}}
{{- define "fsmConnector.provider" -}}
{{- printf .Values.fsm.cloudConnector.connectorProvider -}}
{{- end -}}

{{/* fsm connector's name */}}
{{- define "fsmConnector.name" -}}
{{- printf "fsm-connector-%s-%s" .Values.fsm.cloudConnector.connectorProvider .Values.fsm.cloudConnector.connectorName -}}
{{- end -}}
