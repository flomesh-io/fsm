package multicluster

import (
	networkingv1 "github.com/flomesh-io/fsm/pkg/apis/networking/v1"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/service"
)

// GetIngressControllerServices returns ingress controller services.
func (c *Client) GetIngressControllerServices() []service.MeshService {
	ingressClassIfs := c.informers.List(informers.InformerKeyIngressClass)
	if len(ingressClassIfs) == 0 {
		return nil
	}

	var controllerServices []service.MeshService

	for _, ingressClassIf := range ingressClassIfs {
		ingressClass := ingressClassIf.(*networkingv1.IngressClass)
		if len(ingressClass.Annotations) > 0 {
			if ns, nsExist := ingressClass.Annotations[`meta.flomesh.io/fsm-namespace`]; nsExist && len(ns) > 0 {
				if svc, svcExist := ingressClass.Annotations[`meta.flomesh.io/ingress-pipy-svc`]; svcExist && len(svc) > 0 {
					controllerServices = append(controllerServices, service.MeshService{
						Namespace: ns,
						Name:      svc,
					})
				}
			} else if ns, nsExist = ingressClass.Annotations[`meta.flomesh.io/erie-canal-namespace`]; nsExist && len(ns) > 0 {
				if svc, svcExist := ingressClass.Annotations[`meta.flomesh.io/ingress-pipy-svc`]; svcExist && len(svc) > 0 {
					controllerServices = append(controllerServices, service.MeshService{
						Namespace: ns,
						Name:      svc,
					})
				}
			} else if ns, nsExist = ingressClass.Annotations[`meta.flomesh.io/namespace`]; nsExist && len(ns) > 0 {
				if svc, svcExist := ingressClass.Annotations[`meta.flomesh.io/ingress-pipy-svc`]; svcExist && len(svc) > 0 {
					controllerServices = append(controllerServices, service.MeshService{
						Namespace: ns,
						Name:      svc,
					})
				}
			}
		}
	}

	return controllerServices
}
