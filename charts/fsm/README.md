# Flomesh Service Mesh Helm Chart

![Version: 1.2.0-alpha.1](https://img.shields.io/badge/Version-1.2.0--alpha.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v1.2.0-alpha.1](https://img.shields.io/badge/AppVersion-v1.2.0--alpha.1-informational?style=flat-square)

A Helm chart to install the [fsm](https://github.com/flomesh-io/fsm) control plane on Kubernetes.

## Prerequisites

- Kubernetes >= 1.19.0-0

## Get Repo Info

```console
helm repo add fsm https://flomesh-io.github.io/fsm
helm repo update
```

## Install Chart

```console
helm install [RELEASE_NAME] fsm/fsm
```

The command deploys `fsm-controller` on the Kubernetes cluster in the default configuration.

_See [configuration](#configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Uninstall Chart

```console
helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
helm upgrade [RELEASE_NAME] [CHART] --install
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._

## Configuration

See [Customizing the Chart Before Installing](https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing). To see all configurable options with detailed comments, visit the chart's [values.yaml](./values.yaml), or run these configuration commands:

```console
helm show values fsm/fsm
```

The following table lists the configurable parameters of the fsm chart and their default values.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| fsm.caBundleSecretName | string | `"fsm-ca-bundle"` | The Kubernetes secret name to store CA bundle for the root CA used in FSM |
| fsm.certificateProvider.certKeyBitSize | int | `2048` | Certificate key bit size for data plane certificates issued to workloads to communicate over mTLS |
| fsm.certificateProvider.kind | string | `"tresor"` | The Certificate manager type: `tresor`, `vault` or `cert-manager` |
| fsm.certificateProvider.serviceCertValidityDuration | string | `"24h"` | Service certificate validity duration for certificate issued to workloads to communicate over mTLS |
| fsm.certmanager.issuerGroup | string | `"cert-manager.io"` | cert-manager issuer group |
| fsm.certmanager.issuerKind | string | `"Issuer"` | cert-manager issuer kind |
| fsm.certmanager.issuerName | string | `"fsm-ca"` | cert-manager issuer namecert-manager issuer name |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.cleanup.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.cleanup.nodeSelector | object | `{}` |  |
| fsm.cleanup.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.cloudConnector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.cloudConnector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.cloudConnector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.cloudConnector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-injector"` |  |
| fsm.cloudConnector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.cloudConnector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.cloudConnector.autoScale | object | `{"cpu":{"targetAverageUtilization":80},"enable":false,"maxReplicas":5,"memory":{"targetAverageUtilization":80},"minReplicas":1}` | Auto scale configuration |
| fsm.cloudConnector.autoScale.cpu.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| fsm.cloudConnector.autoScale.enable | bool | `false` | Enable Autoscale |
| fsm.cloudConnector.autoScale.maxReplicas | int | `5` | Maximum replicas for autoscale |
| fsm.cloudConnector.autoScale.memory.targetAverageUtilization | int | `80` | Average target memory utilization (%) |
| fsm.cloudConnector.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| fsm.cloudConnector.consul.deriveNamespace | string | `""` |  |
| fsm.cloudConnector.consul.filterTag | string | `""` |  |
| fsm.cloudConnector.consul.httpAddr | string | `"127.0.0.1:8500"` |  |
| fsm.cloudConnector.consul.passingOnly | bool | `true` |  |
| fsm.cloudConnector.consul.prefixTag | string | `""` |  |
| fsm.cloudConnector.consul.suffixTag | string | `""` |  |
| fsm.cloudConnector.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| fsm.cloudConnector.eureka.deriveNamespace | string | `""` |  |
| fsm.cloudConnector.eureka.filterTag | string | `""` |  |
| fsm.cloudConnector.eureka.httpAddr | string | `"127.0.0.1:8500"` |  |
| fsm.cloudConnector.eureka.passingOnly | bool | `true` |  |
| fsm.cloudConnector.eureka.prefixTag | string | `""` |  |
| fsm.cloudConnector.eureka.suffixTag | string | `""` |  |
| fsm.cloudConnector.nodeSelector | object | `{}` |  |
| fsm.cloudConnector.podLabels | object | `{}` | Sidecar injector's pod labels |
| fsm.cloudConnector.replicaCount | int | `1` | Sidecar injector's replica count (ignored when autoscale.enable is true) |
| fsm.cloudConnector.resource | object | `{"limits":{"cpu":"0.5","memory":"64M"},"requests":{"cpu":"0.3","memory":"64M"}}` | Sidecar injector's container resource parameters |
| fsm.cloudConnector.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.configResyncInterval | string | `"90s"` | Sets the resync interval for regular proxy broadcast updates, set to 0s to not enforce any resync |
| fsm.controlPlaneTolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.controllerLogLevel | string | `"info"` | Controller log verbosity |
| fsm.curlImage | string | `"curlimages/curl"` | Curl image for control plane init container |
| fsm.deployConsulConnector | bool | `false` | Deploy Consul Connector with FSM installation |
| fsm.deployEurekaConnector | bool | `false` | Deploy Eureka Connector with FSM installation |
| fsm.deployGrafana | bool | `false` | Deploy Grafana with FSM installation |
| fsm.deployJaeger | bool | `false` | Deploy Jaeger during FSM installation |
| fsm.deployPrometheus | bool | `false` | Deploy Prometheus with FSM installation |
| fsm.egressGateway.adminPort | int | `6060` |  |
| fsm.egressGateway.enabled | bool | `false` |  |
| fsm.egressGateway.logLevel | string | `"error"` |  |
| fsm.egressGateway.mode | string | `"http2tunnel"` |  |
| fsm.egressGateway.name | string | `"fsm-egress-gateway"` |  |
| fsm.egressGateway.podAnnotations | object | `{}` |  |
| fsm.egressGateway.podLabels | object | `{}` |  |
| fsm.egressGateway.port | int | `1080` |  |
| fsm.egressGateway.replicaCount | int | `1` | FSM Operator Manager's replica count (ignored when autoscale.enable is true) |
| fsm.egressGateway.resources | object | `{"limits":{"cpu":"500m","memory":"128M"},"requests":{"cpu":"100m","memory":"64M"}}` | FSM Operator Manager's container resource parameters. |
| fsm.enableDebugServer | bool | `false` | Enable the debug HTTP server on FSM controller |
| fsm.enableEgress | bool | `true` | Enable egress in the mesh |
| fsm.enableFluentbit | bool | `false` | Enable Fluent Bit sidecar deployment on FSM controller's pod |
| fsm.enablePermissiveTrafficPolicy | bool | `true` | Enable permissive traffic policy mode |
| fsm.enablePrivilegedInitContainer | bool | `false` | Run init container in privileged mode |
| fsm.enableReconciler | bool | `false` | Enable reconciler for FSM's CRDs and mutating webhook |
| fsm.enforceSingleMesh | bool | `true` | Enforce only deploying one mesh in the cluster |
| fsm.featureFlags.enableAccessCertPolicy | bool | `false` |  |
| fsm.featureFlags.enableAccessControlPolicy | bool | `true` | Enables FSM's AccessControl policy API. When enabled, FSM will use the AccessControl API allow access control traffic to mesh backends |
| fsm.featureFlags.enableAsyncProxyServiceMapping | bool | `false` | Enable async proxy-service mapping |
| fsm.featureFlags.enableAutoDefaultRoute | bool | `false` | Enable AutoDefaultRoute |
| fsm.featureFlags.enableEgressPolicy | bool | `true` | Enable FSM's Egress policy API. When enabled, fine grained control over Egress (external) traffic is enforced |
| fsm.featureFlags.enableIngressBackendPolicy | bool | `true` | Enables FSM's IngressBackend policy API. When enabled, FSM will use the IngressBackend API allow ingress traffic to mesh backends |
| fsm.featureFlags.enableMeshRootCertificate | bool | `false` | Enable the MeshRootCertificate to configure the FSM certificate provider |
| fsm.featureFlags.enablePluginPolicy | bool | `false` | Enable Plugin Policy for extend |
| fsm.featureFlags.enableRetryPolicy | bool | `false` | Enable Retry Policy for automatic request retries |
| fsm.featureFlags.enableSidecarActiveHealthChecks | bool | `false` | Enable Sidecar active health checks |
| fsm.featureFlags.enableSnapshotCacheMode | bool | `false` | Enables SnapshotCache feature for Sidecar xDS server. |
| fsm.featureFlags.enableValidateGRPCRouteHostnames | bool | `true` | Enable validate GRPC route hostnames, enforce the hostname is DNS name not IP address |
| fsm.featureFlags.enableValidateGatewayListenerHostname | bool | `true` | Enable validate Gateway listener hostname, enforce the hostname is DNS name not IP address |
| fsm.featureFlags.enableValidateHTTPRouteHostnames | bool | `true` | Enable validate HTTP route hostnames, enforce the hostname is DNS name not IP address |
| fsm.featureFlags.enableValidateTLSRouteHostnames | bool | `true` | Enable validate TLS route hostnames, enforce the hostname is DNS name not IP address |
| fsm.flb.baseUrl | string | `"http://localhost:1337"` |  |
| fsm.flb.defaultAddressPool | string | `"default"` |  |
| fsm.flb.defaultAlgo | string | `"rr"` |  |
| fsm.flb.enabled | bool | `false` |  |
| fsm.flb.k8sCluster | string | `"UNKNOWN"` |  |
| fsm.flb.password | string | `"admin"` |  |
| fsm.flb.secretName | string | `"fsm-flb-secret"` |  |
| fsm.flb.strictMode | bool | `false` |  |
| fsm.flb.username | string | `"admin"` |  |
| fsm.fluentBit.enableProxySupport | bool | `false` | Enable proxy support toggle for Fluent Bit |
| fsm.fluentBit.httpProxy | string | `""` | Optional HTTP proxy endpoint for Fluent Bit |
| fsm.fluentBit.httpsProxy | string | `""` | Optional HTTPS proxy endpoint for Fluent Bit |
| fsm.fluentBit.name | string | `"fluentbit-logger"` | Fluent Bit sidecar container name |
| fsm.fluentBit.outputPlugin | string | `"stdout"` | Fluent Bit output plugin |
| fsm.fluentBit.primaryKey | string | `""` | Primary Key for Fluent Bit output plugin to Log Analytics |
| fsm.fluentBit.pullPolicy | string | `"IfNotPresent"` | PullPolicy for Fluent Bit sidecar container |
| fsm.fluentBit.registry | string | `"fluent"` | Registry for Fluent Bit sidecar container |
| fsm.fluentBit.tag | string | `"1.6.4"` | Fluent Bit sidecar image tag |
| fsm.fluentBit.workspaceId | string | `""` | WorkspaceId for Fluent Bit output plugin to Log Analytics |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.fsmBootstrap.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.fsmBootstrap.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.fsmBootstrap.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmBootstrap.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-bootstrap"` |  |
| fsm.fsmBootstrap.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.fsmBootstrap.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.fsmBootstrap.nodeSelector | object | `{}` |  |
| fsm.fsmBootstrap.podLabels | object | `{}` | FSM bootstrap's pod labels |
| fsm.fsmBootstrap.replicaCount | int | `1` | FSM bootstrap's replica count |
| fsm.fsmBootstrap.resource | object | `{"limits":{"cpu":"0.5","memory":"128M"},"requests":{"cpu":"0.3","memory":"128M"}}` | FSM bootstrap's container resource parameters |
| fsm.fsmBootstrap.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.fsmController.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.fsmController.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.fsmController.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmController.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-controller"` |  |
| fsm.fsmController.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.fsmController.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.fsmController.autoScale | object | `{"cpu":{"targetAverageUtilization":80},"enable":false,"maxReplicas":5,"memory":{"targetAverageUtilization":80},"minReplicas":1}` | Auto scale configuration |
| fsm.fsmController.autoScale.cpu.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| fsm.fsmController.autoScale.enable | bool | `false` | Enable Autoscale |
| fsm.fsmController.autoScale.maxReplicas | int | `5` | Maximum replicas for autoscale |
| fsm.fsmController.autoScale.memory.targetAverageUtilization | int | `80` | Average target memory utilization (%) |
| fsm.fsmController.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| fsm.fsmController.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| fsm.fsmController.podLabels | object | `{}` | FSM controller's pod labels |
| fsm.fsmController.replicaCount | int | `1` | FSM controller's replica count (ignored when autoscale.enable is true) |
| fsm.fsmController.resource | object | `{"limits":{"cpu":"1.5","memory":"1G"},"requests":{"cpu":"0.5","memory":"128M"}}` | FSM controller's container resource parameters. See https://docs.flomesh.io/docs/guides/ha_scale/scale/ for more details. |
| fsm.fsmController.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.fsmGateway.enabled | bool | `false` |  |
| fsm.fsmGateway.logLevel | string | `"info"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.fsmIngress.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-ingress"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[1].key | string | `"ingress.flomesh.io/namespaced"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[1].operator | string | `"In"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[1].values[0] | string | `"false"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.fsmIngress.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.fsmIngress.className | string | `"pipy"` |  |
| fsm.fsmIngress.enabled | bool | `false` |  |
| fsm.fsmIngress.env[0].name | string | `"GIN_MODE"` |  |
| fsm.fsmIngress.env[0].value | string | `"release"` |  |
| fsm.fsmIngress.http.containerPort | int | `8000` |  |
| fsm.fsmIngress.http.enabled | bool | `true` |  |
| fsm.fsmIngress.http.nodePort | int | `30508` |  |
| fsm.fsmIngress.http.port | int | `80` |  |
| fsm.fsmIngress.logLevel | string | `"info"` |  |
| fsm.fsmIngress.namespaced | bool | `false` |  |
| fsm.fsmIngress.nodeSelector | object | `{}` | Node selector applied to control plane pods. |
| fsm.fsmIngress.podAnnotations | object | `{}` |  |
| fsm.fsmIngress.podLabels | object | `{}` | FSM Pipy Ingress Controller's pod labels |
| fsm.fsmIngress.podSecurityContext.runAsGroup | int | `65532` |  |
| fsm.fsmIngress.podSecurityContext.runAsNonRoot | bool | `true` |  |
| fsm.fsmIngress.podSecurityContext.runAsUser | int | `65532` |  |
| fsm.fsmIngress.podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| fsm.fsmIngress.replicaCount | int | `1` | FSM Pipy Ingress Controller's replica count (ignored when autoscale.enable is true) |
| fsm.fsmIngress.resources | object | `{"limits":{"cpu":"2","memory":"1G"},"requests":{"cpu":"0.5","memory":"128M"}}` | FSM Pipy Ingress Controller's container resource parameters. |
| fsm.fsmIngress.securityContext.allowPrivilegeEscalation | bool | `false` |  |
| fsm.fsmIngress.securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| fsm.fsmIngress.service.annotations | object | `{}` |  |
| fsm.fsmIngress.service.name | string | `"fsm-ingress"` |  |
| fsm.fsmIngress.service.type | string | `"LoadBalancer"` |  |
| fsm.fsmIngress.tls.containerPort | int | `8443` |  |
| fsm.fsmIngress.tls.enabled | bool | `false` |  |
| fsm.fsmIngress.tls.mTLS | bool | `false` |  |
| fsm.fsmIngress.tls.nodePort | int | `30607` |  |
| fsm.fsmIngress.tls.port | int | `443` |  |
| fsm.fsmIngress.tls.sslPassthrough.enabled | bool | `false` |  |
| fsm.fsmIngress.tls.sslPassthrough.upstreamPort | int | `443` |  |
| fsm.fsmIngress.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.fsmInterceptor.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.fsmInterceptor.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.fsmInterceptor.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmInterceptor.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-controller"` |  |
| fsm.fsmInterceptor.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.fsmInterceptor.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.fsmInterceptor.cniMode | bool | `true` |  |
| fsm.fsmInterceptor.kernelTracing | bool | `true` |  |
| fsm.fsmInterceptor.kindMode | bool | `false` |  |
| fsm.fsmInterceptor.resource.limits.cpu | string | `"1.5"` |  |
| fsm.fsmInterceptor.resource.limits.memory | string | `"1G"` |  |
| fsm.fsmInterceptor.resource.requests.cpu | string | `"0.5"` |  |
| fsm.fsmInterceptor.resource.requests.memory | string | `"256M"` |  |
| fsm.fsmInterceptor.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.fsmNamespace | string | `""` | Namespace to deploy FSM in. If not specified, the Helm release namespace is used. |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[2] | string | `"arm"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[3] | string | `"ppc64le"` |  |
| fsm.grafana.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[4] | string | `"s390x"` |  |
| fsm.grafana.enableRemoteRendering | bool | `false` | Enable Remote Rendering in Grafana |
| fsm.grafana.image | string | `"grafana/grafana:8.2.2"` | Image used for Grafana |
| fsm.grafana.nodeSelector | object | `{}` |  |
| fsm.grafana.port | int | `3000` | Grafana service's port |
| fsm.grafana.rendererImage | string | `"grafana/grafana-image-renderer:3.2.1"` | Image used for Grafana Renderer |
| fsm.grafana.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.http1PerRequestLoadBalancing | bool | `false` | Specifies a boolean indicating if load balancing based on request is enabled for http1. |
| fsm.http2PerRequestLoadBalancing | bool | `true` | Specifies a boolean indicating if load balancing based on request is enabled for http2. |
| fsm.image.digest | object | `{"fsmBootstrap":"","fsmCRDs":"","fsmConsulConnector":"","fsmController":"","fsmEurekaConnector":"","fsmGateway":"","fsmHealthcheck":"","fsmIngress":"","fsmInjector":"","fsmInterceptor":"","fsmPreinstall":"","fsmSidecarInit":""}` | Image digest (defaults to latest compatible tag) |
| fsm.image.digest.fsmBootstrap | string | `""` | fsm-boostrap's image digest |
| fsm.image.digest.fsmCRDs | string | `""` | fsm-crds' image digest |
| fsm.image.digest.fsmConsulConnector | string | `""` | fsm-consul-connector's image digest |
| fsm.image.digest.fsmController | string | `""` | fsm-controller's image digest |
| fsm.image.digest.fsmEurekaConnector | string | `""` | fsm-eureka-connector's image digest |
| fsm.image.digest.fsmGateway | string | `""` | fsm-gateway's image digest |
| fsm.image.digest.fsmHealthcheck | string | `""` | fsm-healthcheck's image digest |
| fsm.image.digest.fsmIngress | string | `""` | fsm-ingress's image digest |
| fsm.image.digest.fsmInjector | string | `""` | fsm-injector's image digest |
| fsm.image.digest.fsmInterceptor | string | `""` | fsm-interceptor's image digest |
| fsm.image.digest.fsmPreinstall | string | `""` | fsm-preinstall's image digest |
| fsm.image.digest.fsmSidecarInit | string | `""` | Sidecar init container's image digest |
| fsm.image.name | object | `{"fsmBootstrap":"fsm-bootstrap","fsmCRDs":"fsm-crds","fsmConsulConnector":"fsm-consul-connector","fsmController":"fsm-controller","fsmEurekaConnector":"fsm-eureka-connector","fsmGateway":"fsm-gateway","fsmHealthcheck":"fsm-healthcheck","fsmIngress":"fsm-ingress","fsmInjector":"fsm-injector","fsmInterceptor":"fsm-interceptor","fsmPreinstall":"fsm-preinstall","fsmSidecarInit":"fsm-sidecar-init"}` | Image name defaults |
| fsm.image.name.fsmBootstrap | string | `"fsm-bootstrap"` | fsm-boostrap's image name |
| fsm.image.name.fsmCRDs | string | `"fsm-crds"` | fsm-crds' image name |
| fsm.image.name.fsmConsulConnector | string | `"fsm-consul-connector"` | fsm-consul-connector's image name |
| fsm.image.name.fsmController | string | `"fsm-controller"` | fsm-controller's image name |
| fsm.image.name.fsmEurekaConnector | string | `"fsm-eureka-connector"` | fsm-eureka-connector's image name |
| fsm.image.name.fsmGateway | string | `"fsm-gateway"` | fsm-gateway's image name |
| fsm.image.name.fsmHealthcheck | string | `"fsm-healthcheck"` | fsm-healthcheck's image name |
| fsm.image.name.fsmIngress | string | `"fsm-ingress"` | fsm-ingress's image name |
| fsm.image.name.fsmInjector | string | `"fsm-injector"` | fsm-injector's image name |
| fsm.image.name.fsmInterceptor | string | `"fsm-interceptor"` | fsm-interceptor's image name |
| fsm.image.name.fsmPreinstall | string | `"fsm-preinstall"` | fsm-preinstall's image name |
| fsm.image.name.fsmSidecarInit | string | `"fsm-sidecar-init"` | Sidecar init container's image name |
| fsm.image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy for control plane containers |
| fsm.image.registry | string | `"flomesh"` | Container image registry for control plane images |
| fsm.image.tag | string | `"1.2.0-alpha.1"` | Container image tag for control plane images |
| fsm.imagePullSecrets | list | `[]` | `fsm-controller` image pull secret |
| fsm.inboundPortExclusionList | list | `[]` | Specifies a global list of ports to exclude from inbound traffic interception by the sidecar proxy. If specified, must be a list of positive integers. |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.injector.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.injector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.injector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.injector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-injector"` |  |
| fsm.injector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.injector.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.injector.autoScale | object | `{"cpu":{"targetAverageUtilization":80},"enable":false,"maxReplicas":5,"memory":{"targetAverageUtilization":80},"minReplicas":1}` | Auto scale configuration |
| fsm.injector.autoScale.cpu.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| fsm.injector.autoScale.enable | bool | `false` | Enable Autoscale |
| fsm.injector.autoScale.maxReplicas | int | `5` | Maximum replicas for autoscale |
| fsm.injector.autoScale.memory.targetAverageUtilization | int | `80` | Average target memory utilization (%) |
| fsm.injector.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| fsm.injector.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| fsm.injector.nodeSelector | object | `{}` |  |
| fsm.injector.podLabels | object | `{}` | Sidecar injector's pod labels |
| fsm.injector.replicaCount | int | `1` | Sidecar injector's replica count (ignored when autoscale.enable is true) |
| fsm.injector.resource | object | `{"limits":{"cpu":"0.5","memory":"64M"},"requests":{"cpu":"0.3","memory":"64M"}}` | Sidecar injector's container resource parameters |
| fsm.injector.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.injector.webhookTimeoutSeconds | int | `20` | Mutating webhook timeout |
| fsm.localDNSProxy | object | `{"enable":false}` | Local DNS Proxy improves the performance of your computer by caching the responses coming from your DNS servers |
| fsm.localProxyMode | string | `"Localhost"` | Proxy mode for the proxy sidecar. Acceptable values are ['Localhost', 'PodIP'] |
| fsm.maxDataPlaneConnections | int | `0` | Sets the max data plane connections allowed for an instance of fsm-controller, set to 0 to not enforce limits |
| fsm.meshName | string | `"fsm"` | Identifier for the instance of a service mesh within a cluster |
| fsm.networkInterfaceExclusionList | list | `[]` | Specifies a global list of network interface names to exclude for inbound and outbound traffic interception by the sidecar proxy. |
| fsm.outboundIPRangeExclusionList | list | `[]` | Specifies a global list of IP ranges to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of IP ranges of the form a.b.c.d/x. |
| fsm.outboundIPRangeInclusionList | list | `[]` | Specifies a global list of IP ranges to include for outbound traffic interception by the sidecar proxy. If specified, must be a list of IP ranges of the form a.b.c.d/x. |
| fsm.outboundPortExclusionList | list | `[]` | Specifies a global list of ports to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of positive integers. |
| fsm.pluginChains.inbound-http[0].plugin | string | `"modules/inbound-tls-termination"` |  |
| fsm.pluginChains.inbound-http[0].priority | int | `180` |  |
| fsm.pluginChains.inbound-http[1].plugin | string | `"modules/inbound-http-routing"` |  |
| fsm.pluginChains.inbound-http[1].priority | int | `170` |  |
| fsm.pluginChains.inbound-http[2].plugin | string | `"modules/inbound-metrics-http"` |  |
| fsm.pluginChains.inbound-http[2].priority | int | `160` |  |
| fsm.pluginChains.inbound-http[3].plugin | string | `"modules/inbound-tracing-http"` |  |
| fsm.pluginChains.inbound-http[3].priority | int | `150` |  |
| fsm.pluginChains.inbound-http[4].plugin | string | `"modules/inbound-logging-http"` |  |
| fsm.pluginChains.inbound-http[4].priority | int | `140` |  |
| fsm.pluginChains.inbound-http[5].plugin | string | `"modules/inbound-throttle-service"` |  |
| fsm.pluginChains.inbound-http[5].priority | int | `130` |  |
| fsm.pluginChains.inbound-http[6].plugin | string | `"modules/inbound-throttle-route"` |  |
| fsm.pluginChains.inbound-http[6].priority | int | `120` |  |
| fsm.pluginChains.inbound-http[7].plugin | string | `"modules/inbound-http-load-balancing"` |  |
| fsm.pluginChains.inbound-http[7].priority | int | `110` |  |
| fsm.pluginChains.inbound-http[8].plugin | string | `"modules/inbound-http-default"` |  |
| fsm.pluginChains.inbound-http[8].priority | int | `100` |  |
| fsm.pluginChains.inbound-tcp[0].disable | bool | `false` |  |
| fsm.pluginChains.inbound-tcp[0].plugin | string | `"modules/inbound-tls-termination"` |  |
| fsm.pluginChains.inbound-tcp[0].priority | int | `130` |  |
| fsm.pluginChains.inbound-tcp[1].disable | bool | `false` |  |
| fsm.pluginChains.inbound-tcp[1].plugin | string | `"modules/inbound-tcp-routing"` |  |
| fsm.pluginChains.inbound-tcp[1].priority | int | `120` |  |
| fsm.pluginChains.inbound-tcp[2].disable | bool | `false` |  |
| fsm.pluginChains.inbound-tcp[2].plugin | string | `"modules/inbound-tcp-load-balancing"` |  |
| fsm.pluginChains.inbound-tcp[2].priority | int | `110` |  |
| fsm.pluginChains.inbound-tcp[3].disable | bool | `false` |  |
| fsm.pluginChains.inbound-tcp[3].plugin | string | `"modules/inbound-tcp-default"` |  |
| fsm.pluginChains.inbound-tcp[3].priority | int | `100` |  |
| fsm.pluginChains.outbound-http[0].plugin | string | `"modules/outbound-http-routing"` |  |
| fsm.pluginChains.outbound-http[0].priority | int | `160` |  |
| fsm.pluginChains.outbound-http[1].plugin | string | `"modules/outbound-metrics-http"` |  |
| fsm.pluginChains.outbound-http[1].priority | int | `150` |  |
| fsm.pluginChains.outbound-http[2].plugin | string | `"modules/outbound-tracing-http"` |  |
| fsm.pluginChains.outbound-http[2].priority | int | `140` |  |
| fsm.pluginChains.outbound-http[3].plugin | string | `"modules/outbound-logging-http"` |  |
| fsm.pluginChains.outbound-http[3].priority | int | `130` |  |
| fsm.pluginChains.outbound-http[4].plugin | string | `"modules/outbound-circuit-breaker"` |  |
| fsm.pluginChains.outbound-http[4].priority | int | `120` |  |
| fsm.pluginChains.outbound-http[5].plugin | string | `"modules/outbound-http-load-balancing"` |  |
| fsm.pluginChains.outbound-http[5].priority | int | `110` |  |
| fsm.pluginChains.outbound-http[6].plugin | string | `"modules/outbound-http-default"` |  |
| fsm.pluginChains.outbound-http[6].priority | int | `100` |  |
| fsm.pluginChains.outbound-tcp[0].plugin | string | `"modules/outbound-tcp-routing"` |  |
| fsm.pluginChains.outbound-tcp[0].priority | int | `120` |  |
| fsm.pluginChains.outbound-tcp[1].plugin | string | `"modules/outbound-tcp-load-balancing"` |  |
| fsm.pluginChains.outbound-tcp[1].priority | int | `110` |  |
| fsm.pluginChains.outbound-tcp[2].plugin | string | `"modules/outbound-tcp-default"` |  |
| fsm.pluginChains.outbound-tcp[2].priority | int | `100` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.preinstall.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.preinstall.nodeSelector | object | `{}` |  |
| fsm.preinstall.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[2] | string | `"arm"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[3] | string | `"ppc64le"` |  |
| fsm.prometheus.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[4] | string | `"s390x"` |  |
| fsm.prometheus.image | string | `"prom/prometheus:v2.34.0"` | Image used for Prometheus |
| fsm.prometheus.nodeSelector | object | `{}` |  |
| fsm.prometheus.port | int | `7070` | Prometheus service's port |
| fsm.prometheus.resources | object | `{"limits":{"cpu":"1","memory":"2G"},"requests":{"cpu":"0.5","memory":"512M"}}` | Prometheus's container resource parameters |
| fsm.prometheus.retention | object | `{"time":"15d"}` | Prometheus data rentention configuration |
| fsm.prometheus.retention.time | string | `"15d"` | Prometheus data retention time |
| fsm.prometheus.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.remoteLogging.address | string | `""` | Address of the remote logging service (must contain the namespace). When left empty, this is computed in helper template to "remote-logging-service.<fsm-namespace>.svc.cluster.local". |
| fsm.remoteLogging.authorization | string | `""` | The authorization for remote logging service |
| fsm.remoteLogging.enable | bool | `false` | Toggles Sidecar's remote logging functionality on/off for all sidecar proxies in the mesh |
| fsm.remoteLogging.endpoint | string | `""` | Remote logging's API path where the spans will be sent to |
| fsm.remoteLogging.level | int | `2` | Level of the remote logging service |
| fsm.remoteLogging.port | int | `30514` | Port of the remote logging service |
| fsm.remoteLogging.sampledFraction | string | `"1.0"` | Sampled Fraction |
| fsm.remoteLogging.secretName | string | `"fsm-remote-logging-secret"` | Secret Name |
| fsm.repoServer | object | `{"codebase":"","image":"flomesh/pipy-repo:0.90.3-38","ipaddr":"127.0.0.1","standalone":false}` | Pipy RepoServer |
| fsm.repoServer.codebase | string | `""` | codebase is the folder used by fsmController. |
| fsm.repoServer.image | string | `"flomesh/pipy-repo:0.90.3-38"` | Image used for Pipy RepoServer |
| fsm.repoServer.ipaddr | string | `"127.0.0.1"` | ipaddr of host/service where Pipy RepoServer is installed |
| fsm.repoServer.standalone | bool | `false` | if false , Pipy RepoServer is installed within fsmController pod. |
| fsm.serviceAccessMode | string | `"mixed"` | Service access mode |
| fsm.serviceLB.enabled | bool | `false` |  |
| fsm.serviceLBImage | string | `"flomesh/mirrored-klipper-lb:v0.3.5"` | service-lb Image |
| fsm.sidecarClass | string | `"pipy"` | The class of the FSM Sidecar Driver |
| fsm.sidecarDrivers | list | `[{"proxyServerPort":6060,"sidecarImage":"flomesh/pipy:0.90.3-38","sidecarName":"pipy"}]` | Sidecar drivers supported by fsm |
| fsm.sidecarDrivers[0].proxyServerPort | int | `6060` | Remote destination port on which the Discovery Service listens for new connections from Sidecars. |
| fsm.sidecarDrivers[0].sidecarImage | string | `"flomesh/pipy:0.90.3-38"` | Sidecar image for Linux workloads |
| fsm.sidecarImage | string | `""` | Sidecar image for Linux workloads |
| fsm.sidecarLogLevel | string | `"error"` | Log level for the proxy sidecar. Non developers should generally never set this value. In production environments the LogLevel should be set to `error` |
| fsm.sidecarTimeout | int | `60` | Sets connect/idle/read/write timeout |
| fsm.tracing.address | string | `""` | Address of the tracing collector service (must contain the namespace). When left empty, this is computed in helper template to "jaeger.<fsm-namespace>.svc.cluster.local". Please override for BYO-tracing as documented in tracing.md |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[2] | string | `"ppc64le"` |  |
| fsm.tracing.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[3] | string | `"s390x"` |  |
| fsm.tracing.enable | bool | `false` | Toggles Sidecar's tracing functionality on/off for all sidecar proxies in the mesh |
| fsm.tracing.endpoint | string | `"/api/v2/spans"` | Tracing collector's API path where the spans will be sent to |
| fsm.tracing.image | string | `"jaegertracing/all-in-one"` | Image used for tracing |
| fsm.tracing.nodeSelector | object | `{}` |  |
| fsm.tracing.port | int | `9411` | Port of the tracing collector service |
| fsm.tracing.sampledFraction | string | `"1.0"` | Sampled Fraction |
| fsm.tracing.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.trafficInterceptionMode | string | `"iptables"` | Traffic interception mode in the mesh |
| fsm.trustDomain | string | `"cluster.local"` | The trust domain to use as part of the common name when requesting new certificates. |
| fsm.validatorWebhook.webhookConfigurationName | string | `""` | Name of the ValidatingWebhookConfiguration |
| fsm.vault.host | string | `""` | Hashicorp Vault host/service - where Vault is installed |
| fsm.vault.port | int | `8200` | port to use to connect to Vault |
| fsm.vault.protocol | string | `"http"` | protocol to use to connect to Vault |
| fsm.vault.role | string | `"flomesh"` | Vault role to be used by Mesh |
| fsm.vault.secret | object | `{"key":"","name":""}` | The Kubernetes secret storing the Vault token used in FSM. The secret must be located in the namespace of the FSM installation |
| fsm.vault.secret.key | string | `""` | The Kubernetes secret key with the value bring the Vault token |
| fsm.vault.secret.name | string | `""` | The Kubernetes secret name storing the Vault token used in FSM |
| fsm.vault.token | string | `""` | token that should be used to connect to Vault |
| fsm.webhookConfigNamePrefix | string | `"fsm-webhook"` | Prefix used in name of the webhook configuration resources |
| smi.validateTrafficTarget | bool | `true` | Enables validation of SMI Traffic Target |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->