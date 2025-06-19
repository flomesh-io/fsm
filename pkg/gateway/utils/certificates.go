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

type secretReferenceResolver struct {
	gwtypes.SecretReferenceConditionProvider
	client cache.Cache
}

func NewSecretReferenceResolver(conditionProvider gwtypes.SecretReferenceConditionProvider, client cache.Cache) gwtypes.SecretReferenceResolver {
	return &secretReferenceResolver{SecretReferenceConditionProvider: conditionProvider, client: client}
}

func (f *secretReferenceResolver) ResolveAllRefs(referer client.Object, refs []gwv1.SecretObjectReference) bool {
	resolved := true

	for _, ref := range refs {
		if _, err := f.SecretRefToSecret(referer, ref); err != nil {
			log.Error().Msgf("[GW] Error resolving secret reference: %v", err)
			resolved = false
			break
		}
	}

	if resolved {
		f.AddRefsResolvedCondition(referer)
	}

	return resolved
}

func (f *secretReferenceResolver) SecretRefToSecret(referer client.Object, ref gwv1.SecretObjectReference) (*corev1.Secret, error) {
	if !IsValidRefToGroupKindOfSecret(ref) {
		f.AddInvalidCertificateRefCondition(referer, ref)
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
		f.AddRefNotPermittedCondition(referer, ref)

		return nil, fmt.Errorf("cross-namespace secert reference from %s.%s %s/%s to %s.%s %s/%s is not allowed",
			referer.GetObjectKind().GroupVersionKind().Kind, referer.GetObjectKind().GroupVersionKind().Group, referer.GetNamespace(), referer.GetName(),
			string(*ref.Kind), string(*ref.Group), string(*ref.Namespace), ref.Name)
	}

	getSecretFromCache := func(key types.NamespacedName) (*corev1.Secret, error) {
		obj := &corev1.Secret{}
		if err := f.client.Get(context.Background(), key, obj); err != nil {
			if errors.IsNotFound(err) {
				f.AddRefNotFoundCondition(referer, key)
			} else {
				f.AddGetRefErrorCondition(referer, key, err)
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

type objectReferenceResolver struct {
	gwtypes.ObjectReferenceConditionProvider
	client cache.Cache
}

func NewObjectReferenceResolver(conditionProvider gwtypes.ObjectReferenceConditionProvider, client cache.Cache) gwtypes.ObjectReferenceResolver {
	return &objectReferenceResolver{ObjectReferenceConditionProvider: conditionProvider, client: client}
}

func (f *objectReferenceResolver) ResolveAllRefs(referer client.Object, refs []gwv1.ObjectReference) bool {
	resolved := true

	for _, ref := range refs {
		if ca := f.ObjectRefToCACertificate(referer, ref); len(ca) == 0 {
			resolved = false
			break
		}
	}

	if resolved {
		f.AddRefsResolvedCondition(referer)
	}

	return resolved
}

// ObjectRefToCACertificate converts an ObjectReference to a CA Certificate.
// It supports Kubernetes Secret and ConfigMap as the referent.
func (f *objectReferenceResolver) ObjectRefToCACertificate(referer client.Object, ref gwv1.ObjectReference) []byte {
	if !IsValidRefToGroupKindOfCA(ref) {
		f.AddInvalidRefCondition(referer, ref)
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
		f.AddRefNotPermittedCondition(referer, ref)
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
				f.AddRefNotFoundCondition(referer, secretKey, string(ref.Kind))
			} else {
				f.AddGetRefErrorCondition(referer, secretKey, string(ref.Kind), err)
			}

			return nil
		}

		caBytes, ok := secret.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, caBytes...)
		} else {
			f.AddNoRequiredCAFileCondition(referer, secretKey, string(ref.Kind))

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
				f.AddRefNotFoundCondition(referer, cmKey, string(ref.Kind))
			} else {
				f.AddGetRefErrorCondition(referer, cmKey, string(ref.Kind), err)
			}

			return nil
		}

		caBytes, ok := cm.Data[corev1.ServiceAccountRootCAKey]
		if ok {
			ca = append(ca, []byte(caBytes)...)
		} else {
			f.AddNoRequiredCAFileCondition(referer, cmKey, string(ref.Kind))
			return nil
		}
	}

	if len(ca) == 0 {
		f.AddEmptyCACondition(referer, ref)
		return nil
	}

	return ca
}
