package framework

import (
	"context"
	"fmt"

	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateIngress is a wrapper to create an ingress
func (td *FsmTestData) CreateIngress(ns string, ing networkingv1.Ingress) (*networkingv1.Ingress, error) {
	i, err := td.Client.NetworkingV1().Ingresses(ns).Create(context.Background(), &ing, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("could not create Ingress: %w", err)
		return nil, err
	}

	return i, nil
}

// CreateNamespacedIngress is a wrapper to create an ingress
func (td *FsmTestData) CreateNamespacedIngress(ns string, ing nsigv1alpha1.NamespacedIngress) (*nsigv1alpha1.NamespacedIngress, error) {
	i, err := td.NsigClient.FlomeshV1alpha1().NamespacedIngresses(ns).Create(context.Background(), &ing, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("could not create NamespacedIngress: %w", err)
		return nil, err
	}

	return i, nil
}
