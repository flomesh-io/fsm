# FSM

Flomesh Service Mesh (FSM) is a simple, lightweight, [SMI](https://github.com/servicemeshinterface/smi-spec)-compliant, complete, and standalone service mesh solution for Kubernetes East-West and North-South traffic management. FSM's control plane is based on [Open Service Mesh's](https://github.com/openservicemesh/osm) code base, but uses [Flomesh Pipy](https://github.com/flomesh-io/pipy), a programmable network proxy, as its data plane.

Here are some of the key features of FSM:

* **Simple**: FSM is easy to install and configure.
* **Lightweight**: FSM has a small footprint, which makes it ideal for use in resource-constrained environments.
* **SMI-compliant**: FSM is compliant with the [Service Mesh Interface (SMI)](https://github.com/servicemeshinterface/smi-spec), which makes it easy to integrate with other SMI-compliant tools.
* **Complete**: FSM provides a comprehensive set of features for service mesh management, including service discovery, load balancing, and fault tolerance.
* **Standalone**: FSM can be deployed and managed independently of other Kubernetes components.

[Flomesh Pipy](https://flomesh.io/pipy) is a programmable network proxy that provides a high-performance, low-latency, and secure way to route traffic between services. Pipy is built on top of the C++ programming language and is designed to be easy to use and extend by using planet most used programming language JavaScript.

Together, FSM and Pipy provide a powerful and flexible service mesh solution for Kubernetes. FSM is a good choice for organizations that are looking for a simple, lightweight, and SMI-compliant service mesh solution.