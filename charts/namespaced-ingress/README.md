# Flomesh Service Mesh Helm Chart

![Version: 1.4.0-alpha.3](https://img.shields.io/badge/Version-1.4.0--alpha.3-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.4.0-alpha.3](https://img.shields.io/badge/AppVersion-1.4.0--alpha.3-informational?style=flat-square)

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
| fsm.fsmIngress.className | string | `"pipy"` |  |
| fsm.fsmIngress.env[0].name | string | `"GIN_MODE"` |  |
| fsm.fsmIngress.env[0].value | string | `"release"` |  |
| fsm.fsmIngress.http.containerPort | int | `8000` |  |
| fsm.fsmIngress.http.enabled | bool | `true` |  |
| fsm.fsmIngress.http.nodePort | int | `30508` |  |
| fsm.fsmIngress.http.port | int | `80` |  |
| fsm.fsmIngress.initResources | object | `{"limits":{"cpu":"500m","memory":"512M"},"requests":{"cpu":"200m","memory":"128M"}}` | initContainer resource parameters |
| fsm.fsmIngress.logLevel | string | `"info"` |  |
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
| fsm.fsmIngress.tls.containerPort | int | `8443` |  |
| fsm.fsmIngress.tls.enabled | bool | `false` |  |
| fsm.fsmIngress.tls.mTLS | bool | `false` |  |
| fsm.fsmIngress.tls.nodePort | int | `30607` |  |
| fsm.fsmIngress.tls.port | int | `443` |  |
| fsm.fsmIngress.tls.sslPassthrough.enabled | bool | `false` |  |
| fsm.fsmIngress.tls.sslPassthrough.upstreamPort | int | `443` |  |
| fsm.fsmIngress.tolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| fsm.fsmNamespace | string | `""` | Namespace to deploy FSM in. If not specified, the Helm release namespace is used. |
| fsm.image.digest | object | `{"fsmCurl":"","fsmIngress":""}` | Image digest (defaults to latest compatible tag) |
| fsm.image.digest.fsmCurl | string | `""` | fsm-curl's image digest |
| fsm.image.digest.fsmIngress | string | `""` | fsm-gateway's image digest |
| fsm.image.name | object | `{"fsmCurl":"fsm-curl","fsmIngress":"fsm-ingress"}` | Image name defaults |
| fsm.image.name.fsmCurl | string | `"fsm-curl"` | fsm-curl's image name |
| fsm.image.name.fsmIngress | string | `"fsm-ingress"` | fsm-ingress's image name |
| fsm.image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy for control plane containers |
| fsm.image.registry | string | `"flomesh"` | Container image registry for control plane images |
| fsm.image.tag | string | `"1.4.0-alpha.3"` | Container image tag for control plane images |
| fsm.imagePullSecrets | list | `[]` | `fsm-controller` image pull secret |
| fsm.meshName | string | `"fsm"` | Identifier for the instance of a service mesh within a cluster |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->