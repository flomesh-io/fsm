{{/*
ServiceAccountName - namespaced-ingress
*/}}
{{- define "fsm.namespaced-ingress.serviceAccountName" -}}
{{ default "fsm-namespaced-ingress" .Values.nsig.spec.serviceAccountName }}
{{- end }}


{{- define "fsm.namespaced-ingress.heath.port" -}}
{{- if .Values.nsig.spec.http.enabled }}
{{- default .Values.fsm.fsmIngress.http.containerPort .Values.nsig.spec.http.port.targetPort }}
{{- else if and .Values.nsig.spec.tls.enabled }}
{{- default .Values.fsm.fsmIngress.tls.containerPort .Values.nsig.spec.tls.port.targetPort }}
{{- else }}
9091
{{- end }}


{{/* fsm-ingress image */}}
{{- define "fsmIngress.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmIngress .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmIngress .Values.fsm.image.digest.fsmIngress -}}
{{- end -}}
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "fsm.labels" -}}
app.kubernetes.io/name: flomesh.io
app.kubernetes.io/instance: {{ .Values.fsm.meshName }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end -}}
