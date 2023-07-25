{{/* Determine fsm namespace */}}
{{- define "fsm.namespace" -}}
{{ default .Release.Namespace .Values.fsm.fsmNamespace}}
{{- end -}}

{{/* Default tracing address */}}
{{- define "fsm.tracingAddress" -}}
{{- $address := printf "jaeger.%s.svc.cluster.local" (include "fsm.namespace" .) -}}
{{ default $address .Values.fsm.tracing.address}}
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

{{/* Security context values for fluentbit */}}
{{- define "fluentbit.securityContext" -}}
securityContext:
    runAsUser: 0
    capabilities:
        drop:
            - ALL
{{- end -}}

{{/* Resource validator webhook name */}}
{{- define "fsm.validatorWebhookConfigName" -}}
{{- $validatorWebhookConfigName := printf "fsm-validator-mesh-%s" .Values.fsm.meshName -}}
{{ default $validatorWebhookConfigName .Values.fsm.validatorWebhook.webhookConfigurationName}}
{{- end -}}

{{/* fsm-controller image */}}
{{- define "fsmController.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmController .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmController .Values.fsm.image.digest.fsmController -}}
{{- end -}}
{{- end -}}

{{/* fsm-injector image */}}
{{- define "fsmInjector.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmInjector .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmInjector .Values.fsm.image.digest.fsmInjector -}}
{{- end -}}
{{- end -}}

{{/* fsm-consul-conector image */}}
{{- define "fsmConsulConnector.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmConsulConnector .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmConsulConnector .Values.fsm.image.digest.fsmInjector -}}
{{- end -}}
{{- end -}}

{{/* Sidecar init image */}}
{{- define "fsmSidecarInit.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmSidecarInit .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmSidecarInit .Values.fsm.image.digest.fsmSidecarInit -}}
{{- end -}}
{{- end -}}

{{/* fsm-bootstrap image */}}
{{- define "fsmBootstrap.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmBootstrap .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmBootstrap .Values.fsm.image.digest.fsmBootstrap -}}
{{- end -}}
{{- end -}}

{{/* fsm-crds image */}}
{{- define "fsmCRDs.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmCRDs .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmCRDs .Values.fsm.image.digest.fsmCRDs -}}
{{- end -}}
{{- end -}}

{{/* fsm-preinstall image */}}
{{- define "fsmPreinstall.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmPreinstall .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmPreinstall .Values.fsm.image.digest.fsmPreinstall -}}
{{- end -}}
{{- end -}}

{{/* fsm-healthcheck image */}}
{{- define "fsmHealthcheck.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmHealthcheck .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmHealthcheck .Values.fsm.image.digest.fsmHealthcheck -}}
{{- end -}}
{{- end -}}

{{/* fsm-interceptor image */}}
{{- define "fsmInterceptor.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmInterceptor .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmInterceptor .Values.fsm.image.digest.fsmController -}}
{{- end -}}
{{- end -}}

{{/* fsm-ingress image */}}
{{- define "fsmIngress.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmIngress .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmIngress .Values.fsm.image.digest.fsmIngress -}}
{{- end -}}
{{- end -}}

{{/* fsm-gateway image */}}
{{- define "fsmGateway.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmGateway .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmGateway .Values.fsm.image.digest.fsmGateway -}}
{{- end -}}
{{- end -}}

{{- define "fsmIngress.heath.port" -}}
{{- if .Values.fsm.fsmIngress.enabled }}
{{- if and .Values.fsm.fsmIngress.http.enabled (not (empty .Values.fsm.fsmIngress.http.containerPort)) }}
{{- .Values.fsm.fsmIngress.http.containerPort }}
{{- else if and .Values.fsm.fsmIngress.tls.enabled (not (empty .Values.fsm.fsmIngress.tls.containerPort)) }}
{{- .Values.fsm.fsmIngress.tls.containerPort }}
{{- else }}
8081
{{- end }}
{{- else }}
8081
{{- end }}
{{- end }}