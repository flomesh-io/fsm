package cache

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/flomesh-io/fsm/pkg/k8s"

	"github.com/flomesh-io/fsm/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *GatewayCache) getActiveGateways() []*gwv1.Gateway {
	list := &gwv1.GatewayList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.ClassGatewayIndex, constants.FSMGatewayClassName),
	}); err != nil {
		log.Error().Msgf("Failed to list Gateways: %v", err)
		return nil
	}

	return gwutils.GetActiveGateways(gwutils.ToSlicePtr(list.Items))
}

func (c *GatewayCache) getSecretFromCache(key client.ObjectKey) (*corev1.Secret, error) {
	obj := &corev1.Secret{}
	err := c.client.Get(context.TODO(), key, obj)
	//obj, err := c.informers.GetListers().Secret.Secrets(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	//obj.GetObjectKind().SetGroupVersionKind(constants.SecretGVK)

	return obj, nil
}

func (c *GatewayCache) getConfigMapFromCache(key client.ObjectKey) (*corev1.ConfigMap, error) {
	obj := &corev1.ConfigMap{}
	err := c.client.Get(context.TODO(), key, obj)
	//obj, err := c.informers.GetListers().ConfigMap.ConfigMaps(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	//obj.GetObjectKind().SetGroupVersionKind(constants.ConfigMapGVK)

	return obj, nil
}

func (c *GatewayCache) getServiceFromCache(key client.ObjectKey) (*corev1.Service, error) {
	obj := &corev1.Service{}
	err := c.client.Get(context.TODO(), key, obj)
	//obj, err := c.informers.GetListers().Service.Services(key.Namespace).Get(key.Name)
	if err != nil {
		return nil, err
	}

	//obj.GetObjectKind().SetGroupVersionKind(constants.ServiceGVK)

	return obj, nil
}

func (c *GatewayCache) getReferenceGrantsFromCache() []*gwv1beta1.ReferenceGrant {
	list := &gwv1beta1.ReferenceGrantList{}
	err := c.client.List(context.TODO(), list)
	if err != nil {
		log.Error().Msgf("Failed to list ReferenceGrants: %v", err)
		return nil
	}

	return gwutils.ToSlicePtr(list.Items)
}

func (c *GatewayCache) isHeadlessService(key client.ObjectKey) bool {
	service, err := c.getServiceFromCache(key)
	if err != nil {
		log.Error().Msgf("failed to get service from cache: %v", err)
		return false
	}

	return k8s.IsHeadlessService(*service)
}

//func (c *GatewayCache) isRefToService(referer client.Object, ref gwv1.BackendObjectReference, service client.ObjectKey) bool {
//	if !isValidBackendRefToGroupKindOfService(ref) {
//		log.Debug().Msgf("Unsupported backend group %s and kind %s for service", *ref.Group, *ref.Kind)
//		return false
//	}
//
//	// fast-fail, not refer to the service with the same name
//	if string(ref.Name) != service.Name {
//		log.Debug().Msgf("Not refer to the service with the same name, ref.Name: %s, service.Name: %s", ref.Name, service.Name)
//		return false
//	}
//
//	if ns := gwutils.Namespace(ref.Namespace, referer.GetNamespace()); ns != service.Namespace {
//		log.Debug().Msgf("Not refer to the service with the same namespace, resolved namespace: %s, service.Namespace: %s", ns, service.Namespace)
//		return false
//	}
//
//	if ref.Namespace != nil && string(*ref.Namespace) == service.Namespace && string(*ref.Namespace) != referer.GetNamespace() {
//		gvk := referer.GetObjectKind().GroupVersionKind()
//		return gwutils.ValidCrossNamespaceRef(
//			c.getReferenceGrantsFromCache(),
//			gwtypes.CrossNamespaceFrom{
//				Group:     gvk.Group,
//				Kind:      gvk.Kind,
//				Namespace: referer.GetNamespace(),
//			},
//			gwtypes.CrossNamespaceTo{
//				Group:     string(*ref.Group),
//				Kind:      string(*ref.Kind),
//				Namespace: service.Namespace,
//				Name:      service.Name,
//			},
//		)
//	}
//
//	log.Debug().Msgf("Found a match, ref: %s/%s, service: %s/%s", gwutils.Namespace(ref.Namespace, referer.GetNamespace()), ref.Name, service.Namespace, service.Name)
//	return true
//}
//
//func (c *GatewayCache) isRefToSecret(referer client.Object, ref gwv1.SecretObjectReference, secret client.ObjectKey) bool {
//	if !isValidRefToGroupKindOfSecret(ref) {
//		return false
//	}
//
//	// fast-fail, not refer to the secret with the same name
//	if string(ref.Name) != secret.Name {
//		log.Debug().Msgf("Not refer to the secret with the same name, ref.Name: %s, secret.Name: %s", ref.Name, secret.Name)
//		return false
//	}
//
//	if ns := gwutils.Namespace(ref.Namespace, referer.GetNamespace()); ns != secret.Namespace {
//		log.Debug().Msgf("Not refer to the secret with the same namespace, resolved namespace: %s, secret.Namespace: %s", ns, secret.Namespace)
//		return false
//	}
//
//	if ref.Namespace != nil && string(*ref.Namespace) == secret.Namespace && string(*ref.Namespace) != referer.GetNamespace() {
//		return gwutils.ValidCrossNamespaceRef(
//			c.getReferenceGrantsFromCache(),
//			gwtypes.CrossNamespaceFrom{
//				Group:     referer.GetObjectKind().GroupVersionKind().Group,
//				Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
//				Namespace: referer.GetNamespace(),
//			},
//			gwtypes.CrossNamespaceTo{
//				Group:     corev1.GroupName,
//				Kind:      constants.KubernetesSecretKind,
//				Namespace: secret.Namespace,
//				Name:      secret.Name,
//			},
//		)
//	}
//
//	return true
//}
//
//func (c *GatewayCache) isRefToConfigMap(referer client.Object, ref gwv1.ObjectReference, cm client.ObjectKey) bool {
//	if !isValidRefToGroupKindOfConfigMap(ref) {
//		return false
//	}
//
//	// fast-fail, not refer to the cm with the same name
//	if string(ref.Name) != cm.Name {
//		log.Debug().Msgf("Not refer to the cm with the same name, ref.Name: %s, cm.Name: %s", ref.Name, cm.Name)
//		return false
//	}
//
//	if ns := gwutils.Namespace(ref.Namespace, referer.GetNamespace()); ns != cm.Namespace {
//		log.Debug().Msgf("Not refer to the cm with the same namespace, resolved namespace: %s, cm.Namespace: %s", ns, cm.Namespace)
//		return false
//	}
//
//	if ref.Namespace != nil && string(*ref.Namespace) == cm.Namespace && string(*ref.Namespace) != referer.GetNamespace() {
//		return gwutils.ValidCrossNamespaceRef(
//			c.getReferenceGrantsFromCache(),
//			gwtypes.CrossNamespaceFrom{
//				Group:     referer.GetObjectKind().GroupVersionKind().Group,
//				Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
//				Namespace: referer.GetNamespace(),
//			},
//			gwtypes.CrossNamespaceTo{
//				Group:     corev1.GroupName,
//				Kind:      constants.KubernetesConfigMapKind,
//				Namespace: cm.Namespace,
//				Name:      cm.Name,
//			},
//		)
//	}
//
//	return true
//}
