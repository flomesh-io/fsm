package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
)

// IsTLSListener returns true if the listener is a TLS listener.
func IsTLSListener(l gwv1.Listener) bool {
	switch l.Protocol {
	case gwv1.HTTPSProtocolType, gwv1.TLSProtocolType:
		if l.TLS == nil {
			return false
		}

		if l.TLS.Mode == nil {
			return true
		}

		if *l.TLS.Mode == gwv1.TLSModeTerminate {
			return true
		}
	}

	return false
}

type secretReferenceResolverFactory struct {
	gwtypes.SecretReferenceResolver
	client cache.Cache
}

func NewSecretReferenceResolverFactory(resolver gwtypes.SecretReferenceResolver, client cache.Cache) gwtypes.SecretReferenceResolverFactory {
	return &secretReferenceResolverFactory{SecretReferenceResolver: resolver, client: client}
}

func (f *secretReferenceResolverFactory) ResolveAllRefs(referer client.Object, refs []gwv1.SecretObjectReference) bool {
	resolved := true

	for _, ref := range refs {
		if _, err := f.SecretRefToSecret(referer, ref); err != nil {
			log.Error().Msgf("Error resolving secret reference: %v", err)
			resolved = false
			break
		}
	}

	if resolved {
		f.AddRefsResolvedCondition()
	}

	return resolved
}

func (f *secretReferenceResolverFactory) SecretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error) {
	if !IsValidRefToGroupKindOfSecret(ref) {
		f.AddInvalidCertificateRefCondition(ref)
		return nil, fmt.Errorf("unsupported group %s and kind %s for secret", *ref.Group, *ref.Kind)
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     corev1.GroupName,
			Kind:      constants.KubernetesSecretKind,
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
		GetSecretRefGrants(f.client),
	) {
		f.AddRefNotPermittedCondition(ref)

		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
	}

	getSecretFromCache := func(key types.NamespacedName) (*corev1.Secret, error) {
		obj := &corev1.Secret{}
		if err := f.client.Get(context.Background(), key, obj); err != nil {
			if errors.IsNotFound(err) {
				f.AddRefNotFoundCondition(key)
			} else {
				f.AddGetRefErrorCondition(key, err)
			}

			return nil, err
		}

		return obj, nil
	}

	return getSecretFromCache(types.NamespacedName{
		Namespace: NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
		Name:      string(ref.Name),
	})
}

type objectReferenceResolverFactory struct {
	gwtypes.ObjectReferenceResolver
	client cache.Cache
}

func NewObjectReferenceResolverFactory(resolver gwtypes.ObjectReferenceResolver, client cache.Cache) gwtypes.ObjectReferenceResolverFactory {
	return &objectReferenceResolverFactory{ObjectReferenceResolver: resolver, client: client}
}

func (f *objectReferenceResolverFactory) ResolveAllRefs(referer client.Object, refs []gwv1.ObjectReference) bool {
	resolved := true

	for _, ref := range refs {
		if ca := f.ObjectRefToCACertificate(referer, ref); len(ca) == 0 {
			resolved = false
			break
		}
	}

	if resolved {
		f.AddRefsResolvedCondition()
	}

	return resolved
}

// ObjectRefToCACertificate converts an ObjectReference to a CA Certificate.
// It supports Kubernetes Secret and ConfigMap as the referent.
func (f *objectReferenceResolverFactory) ObjectRefToCACertificate(referer client.Object, ref gwv1.ObjectReference) []byte {
	if !IsValidRefToGroupKindOfCA(ref) {
		f.AddInvalidRefCondition(ref)
		return nil
	}

	// If the secret is in a different namespace than the referer, check ReferenceGrants
	if ref.Namespace != nil && string(*ref.Namespace) != referer.GetNamespace() && !ValidCrossNamespaceRef(
		gwtypes.CrossNamespaceFrom{
			Group:     referer.GetObjectKind().GroupVersionKind().Group,
			Kind:      referer.GetObjectKind().GroupVersionKind().Kind,
			Namespace: referer.GetNamespace(),
		},
		gwtypes.CrossNamespaceTo{
			Group:     corev1.GroupName,
			Kind:      constants.KubernetesSecretKind,
			Namespace: string(*ref.Namespace),
			Name:      string(ref.Name),
		},
		GetCARefGrants(f.client),
	) {
		f.AddRefNotPermittedCondition(ref)
		return nil
	}

	ca := make([]byte, 0)
	switch ref.Kind {
	case constants.KubernetesSecretKind:
		getSecretFromCache := func(key types.NamespacedName) (*corev1.Secret, error) {
			obj := &corev1.Secret{}
			if err := f.client.Get(context.Background(), key, obj); err != nil {
				return nil, err
			}

			return obj, nil
		}

		secretKey := types.NamespacedName{
			Namespace: NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		}
		secret, err := getSecretFromCache(secretKey)
		if err != nil {
			if errors.IsNotFound(err) {
				f.AddRefNotFoundCondition(secretKey, string(ref.Kind))
			} else {
				f.AddGetRefErrorCondition(secretKey, string(ref.Kind), err)
			}

			return nil
		}

		caBytes, ok := secret.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, caBytes...)
		} else {
			f.AddNoRequiredCAFileCondition(secretKey, string(ref.Kind))

			return nil
		}
	case constants.KubernetesConfigMapKind:
		getConfigMapFromCache := func(key types.NamespacedName) (*corev1.ConfigMap, error) {
			obj := &corev1.ConfigMap{}
			if err := f.client.Get(context.Background(), key, obj); err != nil {
				return nil, err
			}

			return obj, nil
		}

		cmKey := types.NamespacedName{
			Namespace: NamespaceDerefOr(ref.Namespace, referer.GetNamespace()),
			Name:      string(ref.Name),
		}
		cm, err := getConfigMapFromCache(cmKey)
		if err != nil {
			if errors.IsNotFound(err) {
				f.AddRefNotFoundCondition(cmKey, string(ref.Kind))
			} else {
				f.AddGetRefErrorCondition(cmKey, string(ref.Kind), err)
			}

			return nil
		}

		caBytes, ok := cm.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, []byte(caBytes)...)
		} else {
			f.AddNoRequiredCAFileCondition(cmKey, string(ref.Kind))
			return nil
		}
	}

	if len(ca) == 0 {
		f.AddEmptyCACondition(ref, referer.GetNamespace())
		return nil
	}

	return ca
}
