package ctok

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"

	"github.com/flomesh-io/fsm/pkg/constants"
)

func (s *Sink) updateGatewayEndpointSlice(ctx context.Context, endpoints *apiv1.Endpoints) {
	endpointsDup := endpoints.DeepCopy()
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		eptSliceClient := s.KubeClient.DiscoveryV1().EndpointSlices(endpointsDup.Namespace)
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
			constants.KubernetesEndpointSliceServiceNameLabel: endpointsDup.Name,
		}}
		listOptions := metav1.ListOptions{LabelSelector: labels.Set(labelSelector.MatchLabels).String()}
		eptSliceList, err := eptSliceClient.List(ctx, listOptions)
		if err != nil {
			return err
		}
		if eptSliceList == nil || len(eptSliceList.Items) == 0 {
			return fmt.Errorf("not exists EndpointSlice, name:%s", endpointsDup.Name)
		}
		newEpSlice := eptSliceList.Items[0].DeepCopy()
		if len(newEpSlice.Labels) > 0 {
			delete(newEpSlice.Labels, "endpointslice.kubernetes.io/managed-by")
		}
		var ports []discoveryv1.EndpointPort
		var epts []discoveryv1.Endpoint
		for _, subsets := range endpointsDup.Subsets {
			for _, port := range subsets.Ports {
				shadow := port
				ports = append(ports, discoveryv1.EndpointPort{
					Name:        &shadow.Name,
					Protocol:    &shadow.Protocol,
					Port:        &shadow.Port,
					AppProtocol: shadow.AppProtocol,
				})
			}
			if len(subsets.Addresses) > 0 {
				var ready = true
				var addrs []string
				ept := discoveryv1.Endpoint{
					Conditions: discoveryv1.EndpointConditions{
						Ready: &ready,
					},
				}
				for _, addr := range subsets.Addresses {
					addrs = append(addrs, addr.IP)
				}
				ept.Addresses = addrs
				epts = append(epts, ept)
			}
		}
		newEpSlice.Ports = ports
		newEpSlice.Endpoints = epts

		_, err = eptSliceClient.Update(s.Ctx, newEpSlice, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("error updating EndpointSlice, name:%s warn:%v", newEpSlice.Name, err)
		}
		return err
	})
}
