package utils

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"
)

// SecretNamespaceAndName returns the namespace and name of the secret.
func SecretNamespaceAndName(secretName string, ing *networkingv1.Ingress) (string, string, error) {
	secrNs, secrName, err := cache.SplitMetaNamespaceKey(secretName)
	if secrName == "" {
		return "", "", err
	}

	if secrNs == "" {
		return ing.Namespace, secrName, nil
	}

	return secrNs, secrName, nil
}
