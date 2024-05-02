# Flomesh Service Mesh (FSM)

[![build](https://github.com/flomesh-io/fsm/workflows/Go/badge.svg)](https://github.com/flomesh-io/fsm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/flomesh-io/fsm)](https://goreportcard.com/report/github.com/flomesh-io/fsm)
[![codecov](https://codecov.io/gh/flomesh-io/fsm/branch/main/graph/badge.svg)](https://codecov.io/gh/flomesh-io/fsm)
![Github](https://img.shields.io/github/license/flomesh-io/fsm)
[![release](https://img.shields.io/github/release/flomesh-io/fsm/all.svg)](https://github.com/flomesh-io/fsm/releases)

The Flomesh Service Mesh (FSM) inherits a portion of the archived [OSM](https://github.com/openservicemesh/osm) code and introduces the following enhancements while maintaining compatibility with OSM:

* FSM utilizes [Flomesh Pipy](https://github.com/flomesh-io/pipy) proxy as a replacement for OSM's Envoy proxy. This enables FSM to achieve lightweight control and data planes, optimizing CPU and memory resources effectively.
* Implemented traffic interception using eBPF-based technology instead of iptables-based traffic interception.
* FSM offers comprehensive north-south traffic management capabilities, including Ingress and Gateway APIs.
* Additionally, it facilitates seamless interconnectivity among multiple clusters and incorporates service discovery functionality.

[Flomesh Pipy](https://flomesh.io/pipy) is a programmable network proxy that provides a high-performance, low-latency, and secure way to route traffic between services.

FSM is dedicated to providing a holistic, high-performance, and user-friendly suite of traffic management and service governance capabilities for microservices operating on the Kubernetes platform. By harnessing the combined strengths of FSM and Pipy, we present a dynamic and versatile service mesh solution that empowers Kubernetes-based environments.


## Table of Contents
- [Flomesh Service Mesh (FSM)](#flomesh-service-mesh-fsm)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
    - [Core Principles](#core-principles)
    - [Documentation](#documentation)
    - [Features](#features)
    - [Project status](#project-status)
    - [Support](#support)
    - [SMI Specification support](#smi-specification-support)
  - [FSM Design](#fsm-design)
  - [Install](#install)
    - [Prerequisites](#prerequisites)
    - [Get the FSM CLI](#get-the-fsm-cli)
    - [Install FSM](#install-fsm)
  - [Demonstration](#demonstration)
  - [Using FSM](#using-fsm)
    - [Quick Start](#quick-start)
    - [FSM Usage Patterns](#fsm-usage-patterns)
  - [Community](#community)
  - [Development Guide](#development-guide)
  - [Code of Conduct](#code-of-conduct)
  - [License](#license)


## Overview

FSM runs an Sidecar based control plane on Kubernetes, can be configured with SMI APIs, and works by injecting a [Pipy](https://flomesh.io) Sidecar proxy as a sidecar container next to each instance of your application. The proxy contains and executes rules around access control policies, implements routing configuration, and captures metrics. The control plane continually configures proxies to ensure policies and routing rules are up to date and ensures proxies are healthy.

### Core Principles
1. Simple to understand and contribute to
1. Effortless to install, maintain, and operate
1. Painless to troubleshoot
1. Easy to configure via [Service Mesh Interface (SMI)][2]

### Documentation
Documentation pertaining to the usage of Flomesh Service Mesh is made available at [fsm-docs.flomesh.io](https://fsm-docs.flomesh.io/).

Documentation pertaining to development, release workflows, and other repository specific documentation, can be found in the [docs folder](/docs).

### Features

1. Easily and transparently configure [traffic shifting][3] for deployments
1. Secure service to service communication by [enabling mTLS](https://fsm-docs.flomesh.io/docs/guides/certificates/)
1. Define and execute fine grained [access control][4] policies for services
1. [Observability](https://fsm-docs.flomesh.io/docs/troubleshooting/observability/) and insights into application metrics for debugging and monitoring services
1. Integrate with [external certificate management](https://fsm-docs.flomesh.io/docs/guides/certificates/) services/solutions with a pluggable interface
1. Onboard applications onto the mesh by enabling [automatic sidecar injection](https://fsm-docs.flomesh.io/docs/guides/app_onboarding/sidecar_injection/) of Sidecar proxy

### Project status

FSM is under active development and is ready for production workloads.

### Support

[Please search open issues on GitHub](https://github.com/flomesh-io/fsm/issues), and if your issue isn't already represented please [open a new one](https://github.com/flomesh-io/fsm/issues/new/choose). The FSM project maintainers will respond to the best of their abilities.

### SMI Specification support

|   Kind    | SMI Resource |         Supported Version          |          Comments          |
| :---------------------------- | - | :--------------------------------: |  :--------------------------------: |
| TrafficTarget  | traffictargets.access.smi-spec.io |  [v1alpha3](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md)  | |
| HTTPRouteGroup | httproutegroups.specs.smi-spec.io | [v1alpha4](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md#httproutegroup) | |
| TCPRoute | tcproutes.specs.smi-spec.io | [v1alpha4](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md#tcproute) | |
| UDPRoute | udproutes.specs.smi-spec.io | _not supported_ | |
| TrafficSplit | trafficsplits.split.smi-spec.io | [v1alpha4](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha4/traffic-split.md) | |
| TrafficMetrics  | \*.metrics.smi-spec.io | [v1alpha1](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-metrics/v1alpha1/traffic-metrics.md) | 🚧 **In Progress** 🚧 |

### GatewayAPI Specification Support
|   Kind    |        Supported Version          |          Comments          |
| :---------------------------- | :--------------------------------: |  :--------------------------------: |
| GatewayClass | v1 | |
| Gateway      | v1 | |
| HTTPRoute | v1 | |
| GRPCRoute | v1alpha2 | |
| TLSRoute | v1alpha2 | |
| TCPRoute | v1alpha2 | |
| UDPRoute | v1alpha2 | |
| ReferenceGrant | v1beta1 | |

## FSM Design

Read more about [FSM's high level goals, design, and architecture](DESIGN.md).

## Install

### Prerequisites
- Kubernetes cluster running Kubernetes v1.19.0 or greater
- kubectl current context is configured for the target cluster install
  - ```kubectl config current-context```

### Get the FSM CLI

The simplest way of installing Flomesh Service Mesh on a Kubernetes cluster is by using the `fsm` CLI.

Download the `fsm` binary from the [Releases page](https://github.com/flomesh-io/fsm/releases). Unpack the `fsm` binary and add it to `$PATH` to get started.
```shell
sudo mv ./fsm /usr/local/bin/fsm
```

### Install FSM
```shell
$ fsm install
```
![FSM Install Demo](docs/images/fsm-install-demo.gif "FSM Install Demo")

See the [installation guide](https://fsm-docs.flomesh.io/docs/guides/install/) for more detailed options.

## Demonstration

The FSM [Bookstore demo](https://fsm-docs.flomesh.io/docs/getting_started/) is a step-by-step walkthrough of how to install a bookbuyer and bookstore apps, and configure connectivity between these using SMI.

## Using FSM

After installing FSM, [onboard a microservice application](https://fsm-docs.flomesh.io/docs/guides/app_onboarding/) to the service mesh.

### Quick Start

Refer to [Quick Start](https://fsm-docs.flomesh.io/docs/quickstart/) guide for step-by-step guide on how to start quickly.

### FSM Usage Patterns

1. [Traffic Management](https://fsm-docs.flomesh.io/docs/guides/traffic_management/)
1. [Observability](https://fsm-docs.flomesh.io/docs/troubleshooting/observability/)
1. [Certificates](https://fsm-docs.flomesh.io/docs/guides/certificates/)
1. [Sidecar Injection](https://fsm-docs.flomesh.io/docs/guides/app_onboarding/sidecar_injection/)


## Community

Connect with the Flomesh Service Mesh community:

- GitHub [issues](https://github.com/flomesh-io/fsm/issues) and [pull requests](https://github.com/flomesh-io/fsm/pulls) in this repo
- FSM Slack: <a href="https://join.slack.com/t/flomesh-io/shared_invite/zt-16f4yv2hc-qvEgSrMATKn5LjmDAwzlbw">Join</a> the Flomesh-io Slack for related discussions

## Development Guide

If you would like to contribute to FSM, check out the [development guide](docs/development_guide/README.md).

## Code of Conduct

This project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for further details.

## License

This software is covered under the Apache 2.0 license. You can read the license [here](LICENSE).


[1]: https://en.wikipedia.org/wiki/Service_mesh
[2]: https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC_LATEST_STABLE.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha4/traffic-split.md
[4]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md
