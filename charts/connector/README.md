# Flomesh Service Mesh Helm Chart

![Version: 1.4.0](https://img.shields.io/badge/Version-1.4.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.4.0](https://img.shields.io/badge/AppVersion-1.4.0-informational?style=flat-square)

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
| fsm.cloudConnector.connectorName | string | `""` |  |
| fsm.cloudConnector.connectorNamespace | string | `""` |  |
| fsm.cloudConnector.connectorProvider | string | `""` |  |
| fsm.cloudConnector.connectorUID | string | `""` |  |
| fsm.cloudConnector.enable | bool | `false` |  |
| fsm.cloudConnector.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| fsm.cloudConnector.leaderElection | bool | `false` |  |
| fsm.cloudConnector.nodeSelector | object | `{}` |  |
| fsm.cloudConnector.podLabels | object | `{}` | Sidecar injector's pod labels |
| fsm.cloudConnector.replicaCount | int | `1` | Sidecar injector's replica count (ignored when autoscale.enable is true) |
| fsm.cloudConnector.resource | object | `{"limits":{"cpu":"1","memory":"1G"},"requests":{"cpu":"0.5","memory":"128M"}}` | Sidecar injector's container resource parameters |
| fsm.cloudConnector.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.controllerLogLevel | string | `"info"` | Controller log verbosity |
| fsm.fsmNamespace | string | `""` | Namespace to deploy FSM in. If not specified, the Helm release namespace is used. |
| fsm.fsmServiceAccountName | string | `""` | ServiceAccountName to deploy FSM in. If not specified, the Helm release name is used. |
| fsm.image.digest | object | `{"fsmConnector":"","fsmCurl":""}` | Image digest (defaults to latest compatible tag) |
| fsm.image.digest.fsmConnector | string | `""` | fsm-connector's image digest |
| fsm.image.digest.fsmCurl | string | `""` | fsm-curl's image digest |
| fsm.image.name | object | `{"fsmConnector":"fsm-connector","fsmCurl":"fsm-curl"}` | Image name defaults |
| fsm.image.name.fsmConnector | string | `"fsm-connector"` | fsm-connector's image name |
| fsm.image.name.fsmCurl | string | `"fsm-curl"` | fsm-curl's image name |
| fsm.image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy for control plane containers |
| fsm.image.registry | string | `"flomesh"` | Container image registry for control plane images |
| fsm.image.tag | string | `"1.4.0"` | Container image tag for control plane images |
| fsm.imagePullSecrets | list | `[]` | `fsm-connector` image pull secret |
| fsm.meshName | string | `"fsm"` | Identifier for the instance of a service mesh within a cluster |
| fsm.trustDomain | string | `"cluster.local"` | The trust domain to use as part of the common name when requesting new certificates. |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->