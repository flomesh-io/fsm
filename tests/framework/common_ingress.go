package framework

import (
	"context"
	"fmt"

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
