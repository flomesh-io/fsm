# Flomesh Service Mesh Helm Chart

![Version: 1.3.0-alpha.6](https://img.shields.io/badge/Version-1.3.0--alpha.6-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.3.0-alpha.6](https://img.shields.io/badge/AppVersion-1.3.0--alpha.6-informational?style=flat-square)

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
| fsm.curlImage | string | `"curlimages/curl"` | Curl image for control plane init container |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key | string | `"kubernetes.io/os"` |  |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0] | string | `"linux"` |  |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].key | string | `"kubernetes.io/arch"` |  |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].operator | string | `"In"` |  |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[0] | string | `"amd64"` |  |
| fsm.fsmGateway.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[1].values[1] | string | `"arm64"` |  |
| fsm.fsmGateway.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].key | string | `"app"` |  |
| fsm.fsmGateway.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].operator | string | `"In"` |  |
| fsm.fsmGateway.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.labelSelector.matchExpressions[0].values[0] | string | `"fsm-gateway"` |  |
| fsm.fsmGateway.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].podAffinityTerm.topologyKey | string | `"kubernetes.io/hostname"` |  |
| fsm.fsmGateway.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[0].weight | int | `100` |  |
| fsm.fsmGateway.autoScale | object | `{"behavior":{"scaleDown":{"policies":[{"periodSeconds":60,"type":"Pods","value":1},{"periodSeconds":60,"type":"Percent","value":10}],"selectPolicy":"Min","stabilizationWindowSeconds":300},"scaleUp":{"policies":[{"periodSeconds":15,"type":"Percent","value":100},{"periodSeconds":15,"type":"Pods","value":2}],"selectPolicy":"Max","stabilizationWindowSeconds":0}},"cpu":{"targetAverageUtilization":80},"enable":false,"maxReplicas":10,"memory":{"targetAverageUtilization":80},"metrics":[{"resource":{"name":"cpu","target":{"averageUtilization":80,"type":"Utilization"}},"type":"Resource"},{"resource":{"name":"memory","target":{"averageUtilization":80,"type":"Utilization"}},"type":"Resource"}],"minReplicas":1}` | Auto scale configuration |
| fsm.fsmGateway.autoScale.behavior | object | `{"scaleDown":{"policies":[{"periodSeconds":60,"type":"Pods","value":1},{"periodSeconds":60,"type":"Percent","value":10}],"selectPolicy":"Min","stabilizationWindowSeconds":300},"scaleUp":{"policies":[{"periodSeconds":15,"type":"Percent","value":100},{"periodSeconds":15,"type":"Pods","value":2}],"selectPolicy":"Max","stabilizationWindowSeconds":0}}` | Auto scale behavior, for v2 API |
| fsm.fsmGateway.autoScale.cpu | object | `{"targetAverageUtilization":80}` | Auto scale cpu metrics, for v2beta2 API |
| fsm.fsmGateway.autoScale.cpu.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| fsm.fsmGateway.autoScale.enable | bool | `false` | Enable Autoscale |
| fsm.fsmGateway.autoScale.maxReplicas | int | `10` | Maximum replicas for autoscale |
| fsm.fsmGateway.autoScale.memory | object | `{"targetAverageUtilization":80}` | Auto scale memory metrics, for v2beta2 API |
| fsm.fsmGateway.autoScale.memory.targetAverageUtilization | int | `80` | Average target memory utilization (%) |
| fsm.fsmGateway.autoScale.metrics | list | `[{"resource":{"name":"cpu","target":{"averageUtilization":80,"type":"Utilization"}},"type":"Resource"},{"resource":{"name":"memory","target":{"averageUtilization":80,"type":"Utilization"}},"type":"Resource"}]` | Auto scale metrics, for v2 API |
| fsm.fsmGateway.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| fsm.fsmGateway.env[0].name | string | `"GIN_MODE"` |  |
| fsm.fsmGateway.env[0].value | string | `"release"` |  |
| fsm.fsmGateway.initResources | object | `{"limits":{"cpu":"500m","memory":"512M"},"requests":{"cpu":"200m","memory":"128M"}}` | initContainer resource configuration |
| fsm.fsmGateway.logLevel | string | `"info"` |  |
| fsm.fsmGateway.nodeSelector | object | `{}` | Node selector applied to control plane pods. |
| fsm.fsmGateway.podAnnotations | object | `{}` | FSM Gateway Controller's pod annotations |
| fsm.fsmGateway.podDisruptionBudget | object | `{"enabled":false,"minAvailable":1}` | Pod disruption budget configuration |
| fsm.fsmGateway.podDisruptionBudget.enabled | bool | `false` | Enable Pod Disruption Budget |
| fsm.fsmGateway.podDisruptionBudget.minAvailable | int | `1` | Minimum number of pods that must be available |
| fsm.fsmGateway.podLabels | object | `{}` | FSM Gateway Controller's pod labels |
| fsm.fsmGateway.podSecurityContext | object | `{"runAsGroup":65532,"runAsNonRoot":true,"runAsUser":65532,"seccompProfile":{"type":"RuntimeDefault"}}` | FSM Gateway Controller's pod security context |
| fsm.fsmGateway.replicas | int | `1` |  |
| fsm.fsmGateway.resources | object | `{"limits":{"cpu":"2","memory":"1G"},"requests":{"cpu":"0.5","memory":"128M"}}` | FSM Gateway's container resource parameters. |
| fsm.fsmGateway.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}` | FSM Gateway Controller's container security context |
| fsm.fsmGateway.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.fsmNamespace | string | `""` | Namespace to deploy FSM in. If not specified, the Helm release namespace is used. |
| fsm.image.digest | object | `{"fsmGateway":""}` | Image digest (defaults to latest compatible tag) |
| fsm.image.digest.fsmGateway | string | `""` | fsm-gateway's image digest |
| fsm.image.name | object | `{"fsmGateway":"fsm-gateway"}` | Image name defaults |
| fsm.image.name.fsmGateway | string | `"fsm-gateway"` | fsm-gateway's image name |
| fsm.image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy for control plane containers |
| fsm.image.registry | string | `"flomesh"` | Container image registry for control plane images |
| fsm.image.tag | string | `"1.3.0-alpha.6"` | Container image tag for control plane images |
| fsm.imagePullSecrets | list | `[]` | `fsm-gateway` image pull secret |
| fsm.meshName | string | `"fsm"` | Identifier for the instance of a service mesh within a cluster |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->