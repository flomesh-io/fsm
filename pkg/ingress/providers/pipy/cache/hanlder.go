package cache

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// OnAdd is called whenever a new object is added to the store.
func (c *Cache) OnAdd(obj interface{}) bool {
	switch obj := obj.(type) {
	case *corev1.Endpoints:
		return c.endpointsChanges.Update(nil, obj)
	case *corev1.Service:
		return c.serviceChanges.Update(nil, obj)
	case *mcsv1alpha1.ServiceImport:
		return c.serviceImportChanges.Update(nil, obj)
	case *networkingv1.Ingress:
		return c.ingressChanges.Update(nil, obj)
	case *networkingv1.IngressClass:
		updateDefaultIngressClass(obj, obj.Name)
		return true
	}

	return false
}

// OnUpdate is called whenever an object is updated.
func (c *Cache) OnUpdate(oldObj, newObj interface{}) bool {
	switch objectForType(oldObj, newObj).(type) {
	case *corev1.Endpoints:
		oldObj := oldObj.(*corev1.Endpoints)
		newObj := newObj.(*corev1.Endpoints)
		return c.endpointsChanges.Update(oldObj, newObj)
	case *corev1.Service:
		oldObj := oldObj.(*corev1.Service)
		newObj := newObj.(*corev1.Service)
		return c.serviceChanges.Update(oldObj, newObj)
	case *mcsv1alpha1.ServiceImport:
		oldObj := oldObj.(*mcsv1alpha1.ServiceImport)
		newObj := newObj.(*mcsv1alpha1.ServiceImport)
		return c.serviceImportChanges.Update(oldObj, newObj)
	case *networkingv1.Ingress:
		oldObj := oldObj.(*networkingv1.Ingress)
		newObj := newObj.(*networkingv1.Ingress)
		return c.ingressChanges.Update(oldObj, newObj)
	case *networkingv1.IngressClass:
		oldObj := oldObj.(*networkingv1.IngressClass)
		newObj := newObj.(*networkingv1.IngressClass)

		if oldObj.ResourceVersion == newObj.ResourceVersion {
			return false
		}
		updateDefaultIngressClass(newObj, newObj.Name)
		return true
	}

	return false
}

func objectForType(oldObj, newObj interface{}) interface{} {
	if oldObj == nil {
		return newObj
	}

	if newObj == nil {
		return oldObj
	}

	return newObj
}

// OnDelete is called whenever an object is deleted from the store.
func (c *Cache) OnDelete(obj interface{}) bool {
	switch obj := obj.(type) {
	case *corev1.Endpoints:
		return c.endpointsChanges.Update(obj, nil)
	case *corev1.Service:
		return c.serviceChanges.Update(obj, nil)
	case *mcsv1alpha1.ServiceImport:
		return c.serviceImportChanges.Update(obj, nil)
	case *networkingv1.Ingress:
		return c.ingressChanges.Update(obj, nil)
	case *networkingv1.IngressClass:
		// if the default IngressClass is deleted, set the DefaultIngressClass variable to empty
		updateDefaultIngressClass(obj, constants.NoDefaultIngressClass)
		return true
	}

	return false
}

func updateDefaultIngressClass(class *networkingv1.IngressClass, className string) {
	isDefault, ok := class.GetAnnotations()[constants.IngressClassAnnotationKey]
	if ok && isDefault == "true" {
		constants.DefaultIngressClass = className
	}
}
