package ctok

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	"github.com/flomesh-io/fsm/pkg/connector"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

type endpointsResource struct {
	sink *Sink
	// endpointsKeyToName maps from Kube controller keys to Kube endpoints names.
	// Controller keys are in the form <kube namespace>/<kube endpoints name>
	// e.g. default/foo, and are the keys Kube uses to inform that something
	// changed.
	endpointsKeyToName map[string]string
}

func (t *endpointsResource) Informer() cache.SharedIndexInformer {
	sink := t.sink
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return sink.KubeClient.CoreV1().Endpoints(sink.namespace()).List(sink.Ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return sink.KubeClient.CoreV1().Endpoints(sink.namespace()).Watch(sink.Ctx, options)
			},
		},
		&apiv1.Endpoints{},
		0,
		cache.Indexers{},
	)
}

func (t *endpointsResource) Upsert(key string, raw interface{}) error {
	// We expect a Service. If it isn't a service then just ignore it.
	endpoints, ok := raw.(*apiv1.Endpoints)
	if !ok {
		log.Warn().Msgf("UpsertEndpoints got invalid type, raw:%v", raw)
		return nil
	}

	sink := t.sink
	sink.lock.Lock()
	defer sink.lock.Unlock()

	// Store all the key to name mappings. We need this because the key
	// is opaque but we want to do all the lookups by endpoints name.
	t.endpointsKeyToName[key] = endpoints.Name

	servicePtr, exists := sink.serviceMapCache[endpoints.Name]
	if exists {
		service := *servicePtr
		if len(service.Annotations) > 0 {
			endpoints.Labels[CloudServiceLabel] = service.Name

			if len(endpoints.Annotations) == 0 {
				endpoints.Annotations = make(map[string]string)
			}
			if clusterId, clusterIDExists := service.Annotations[connector.AnnotationCloudServiceInheritedClusterID]; clusterIDExists {
				endpoints.Annotations[connector.AnnotationCloudServiceInheritedClusterID] = clusterId
			}
			if withGateway {
				if sink.DiscClient.IsInternalServices() {
					endpoints.Annotations[connector.AnnotationMeshServiceInternalSync] = True
				}
			}

			endpointSubset := apiv1.EndpointSubset{}
			ported := false
			for k := range service.Annotations {
				if !ported {
					if len(service.Spec.Ports) > 0 {
						for _, port := range service.Spec.Ports {
							endpointSubset.Ports = append(endpointSubset.Ports, apiv1.EndpointPort{
								Name:        port.Name,
								Port:        port.Port,
								Protocol:    port.Protocol,
								AppProtocol: port.AppProtocol,
							})
						}
					}
					ported = true
				}
				if strings.HasPrefix(k, connector.AnnotationMeshEndpointAddr) {
					ipIntStr := strings.TrimPrefix(k, fmt.Sprintf("%s-", connector.AnnotationMeshEndpointAddr))
					if ipInt, err := strconv.ParseUint(ipIntStr, 10, 32); err == nil {
						ip := utils.Int2IP4(uint32(ipInt))
						endpointSubset.Addresses = append(endpointSubset.Addresses, apiv1.EndpointAddress{IP: ip.To4().String()})
					}
				}
			}
			if len(endpointSubset.Addresses) > 0 {
				endpoints.Subsets = []apiv1.EndpointSubset{endpointSubset}
				eptClient := sink.KubeClient.CoreV1().Endpoints(sink.namespace())
				return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
					if updatedEpt, err := eptClient.Update(sink.Ctx, endpoints, metav1.UpdateOptions{}); err != nil {
						log.Warn().Err(err).Msgf("error update endpoints, name:%s", service.Name)
						return err
					} else {
						t.updateGatewayEndpointSlice(sink.Ctx, updatedEpt)
					}
					return nil
				})
			}
		}
	}
	return fmt.Errorf("error update endpoints, name:%s", endpoints.Name)
}

func (t *endpointsResource) Delete(key string, _ interface{}) error {
	sink := t.sink
	sink.lock.Lock()
	defer sink.lock.Unlock()

	name, ok := t.endpointsKeyToName[key]
	if !ok {
		return nil
	}

	delete(t.endpointsKeyToName, key)

	log.Info().Msgf("delete endpoints, key:%s name:%s", key, name)
	return nil
}

func (t *endpointsResource) updateGatewayEndpointSlice(ctx context.Context, endpoints *apiv1.Endpoints) {
	endpointsDup := endpoints.DeepCopy()
	sink := t.sink
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		eptSliceClient := sink.KubeClient.DiscoveryV1().EndpointSlices(endpointsDup.Namespace)
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

		_, err = eptSliceClient.Update(sink.Ctx, newEpSlice, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("error updating EndpointSlice, name:%s warn:%v", newEpSlice.Name, err)
		}
		return err
	})
}
