{{/* Determine fsm namespace */}}
{{- define "fsm.namespace" -}}
{{ default .Release.Namespace .Values.fsm.fsmNamespace}}
{{- end -}}

{{/* Default tracing address */}}
{{- define "fsm.tracingAddress" -}}
{{- $address := printf "jaeger.%s" (include "fsm.namespace" .) -}}
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

{{/* fsm-conector image */}}
{{- define "fsmConnector.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmConnector .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmConnector .Values.fsm.image.digest.fsmInjector -}}
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

{{/* fsm-xmgt image */}}
{{- define "fsmXnetwork.xmgt.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmXnetmgmt .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmXnetmgmt .Values.fsm.image.digest.fsmController -}}
{{- end -}}
{{- end -}}

{{/* fsm-xnet image */}}
{{- define "fsmXnetwork.xnet.image" -}}
{{- if .Values.fsm.fsmXnetwork.xnet.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.fsmXnetwork.xnet.image.registry .Values.fsm.fsmXnetwork.xnet.image.name .Values.fsm.fsmXnetwork.xnet.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.fsmXnetwork.xnet.image.name .Values.fsm.fsmXnetwork.xnet.image.tag -}}
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

{{/* fsm-curl image */}}
{{- define "fsmCurl.image" -}}
{{- if .Values.fsm.image.tag -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmCurl .Values.fsm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.fsm.image.registry .Values.fsm.image.name.fsmCurl .Values.fsm.image.digest.fsmCurl -}}
{{- end -}}
{{- end -}}

{{/* pipy repo image */}}
{{- define "repoServer.image" -}}
{{- if .Values.fsm.repoServer.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.repoServer.image.registry .Values.fsm.repoServer.image.name .Values.fsm.repoServer.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.repoServer.image.name .Values.fsm.repoServer.image.tag -}}
{{- end -}}
{{- end -}}

{{/* pipy sidecar image */}}
{{- define "sidecar.image" -}}
{{- if .Values.fsm.sidecar.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.sidecar.image.registry .Values.fsm.sidecar.image.name .Values.fsm.sidecar.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.sidecar.image.name .Values.fsm.sidecar.image.tag -}}
{{- end -}}
{{- end -}}

{{/* serviceLB image */}}
{{- define "serviceLB.image" -}}
{{- if .Values.fsm.serviceLB.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.serviceLB.image.registry .Values.fsm.serviceLB.image.name .Values.fsm.serviceLB.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.serviceLB.image.name .Values.fsm.serviceLB.image.tag -}}
{{- end -}}
{{- end -}}

{{/* prometheus image */}}
{{- define "prometheus.image" -}}
{{- if .Values.fsm.prometheus.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.prometheus.image.registry .Values.fsm.prometheus.image.name .Values.fsm.prometheus.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.prometheus.image.name .Values.fsm.prometheus.image.tag -}}
{{- end -}}
{{- end -}}

{{/* grafana image */}}
{{- define "grafana.image" -}}
{{- if .Values.fsm.grafana.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.grafana.image.registry .Values.fsm.grafana.image.name .Values.fsm.grafana.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.grafana.image.name .Values.fsm.grafana.image.tag -}}
{{- end -}}
{{- end -}}

{{/* grafana renderer image */}}
{{- define "grafana.renderer.image" -}}
{{- if .Values.fsm.grafana.rendererImage.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.grafana.rendererImage.registry .Values.fsm.grafana.rendererImage.name .Values.fsm.grafana.rendererImage.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.grafana.rendererImage.name .Values.fsm.grafana.rendererImage.tag -}}
{{- end -}}
{{- end -}}

{{/* fluentBit image */}}
{{- define "fluentBit.image" -}}
{{- if .Values.fsm.fluentBit.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.fluentBit.image.registry .Values.fsm.fluentBit.image.name .Values.fsm.fluentBit.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.fluentBit.image.name .Values.fsm.fluentBit.image.tag -}}
{{- end -}}
{{- end -}}

{{/* tracing image */}}
{{- define "tracing.image" -}}
{{- if .Values.fsm.tracing.image.registry -}}
{{- printf "%s/%s:%s" .Values.fsm.tracing.image.registry .Values.fsm.tracing.image.name .Values.fsm.tracing.image.tag -}}
{{- else -}}
{{- printf "%s/%s:%s" .Values.fsm.image.registry .Values.fsm.tracing.image.name .Values.fsm.tracing.image.tag -}}
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

{{/* fsm-xnet node path of cni bin */}}
{{- define "fsmXnetwork.xnet.node.cniBin.path" -}}
{{- if .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.cniBin -}}
{{- else if .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.cniBin -}}
{{- else -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.cniBin -}}
{{- end -}}
{{- end -}}

{{/* fsm-xnet node path of cni netd */}}
{{- define "fsmXnetwork.xnet.node.cniNetd.path" -}}
{{- if .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.cniNetd -}}
{{- else if .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.cniNetd -}}
{{- else -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.cniNetd -}}
{{- end -}}
{{- end -}}

{{/* fsm-xnet node path of sys fs */}}
{{- define "fsmXnetwork.xnet.node.sysFs.path" -}}
{{- if .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.sysFs -}}
{{- else if .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.sysFs -}}
{{- else -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.sysFs -}}
{{- end -}}
{{- end -}}

{{/* fsm-xnet node path of sys run */}}
{{- define "fsmXnetwork.xnet.node.sysRun.path" -}}
{{- if .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k8s.sysRun -}}
{{- else if .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.enable -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.sysRun -}}
{{- else -}}
{{- printf "%s" .Values.fsm.fsmXnetwork.xnet.nodePaths.k3s.sysRun -}}
{{- end -}}
{{- end -}}