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
	"github.com/flomesh-io/fsm/pkg/utils"
)

type endpointsResource struct {
	controller connector.ConnectController
	syncer     *CtoKSyncer
}

func newEndpointsResource(controller connector.ConnectController, syncer *CtoKSyncer) *endpointsResource {
	resource := endpointsResource{
		controller: controller,
		syncer:     syncer,
	}
	return &resource
}

func (t *endpointsResource) Informer() cache.SharedIndexInformer {
	syncer := t.syncer
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return syncer.kubeClient.CoreV1().Endpoints(syncer.namespace()).List(syncer.ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return syncer.kubeClient.CoreV1().Endpoints(syncer.namespace()).Watch(syncer.ctx, options)
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

	syncer := t.syncer
	syncer.lock.Lock()
	defer syncer.lock.Unlock()

	// Store all the key to name mappings. We need this because the key
	// is opaque but we want to do all the lookups by endpoints name.
	t.controller.GetC2KContext().EndpointsKeyToName[key] = endpoints.Name

	servicePtr, exists := syncer.controller.GetC2KContext().ServiceMapCache[endpoints.Name]
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
			if clusterSet, clusterSetyExists := service.Annotations[connector.AnnotationCloudServiceClusterSet]; clusterSetyExists {
				endpoints.Annotations[connector.AnnotationCloudServiceClusterSet] = clusterSet
			}
			if viaGateway, viaGatewayExists := service.Annotations[connector.AnnotationCloudServiceViaGateway]; viaGatewayExists {
				endpoints.Annotations[connector.AnnotationCloudServiceViaGateway] = viaGateway
			}
			if t.controller.GetC2KWithGateway() {
				endpoints.Annotations[connector.AnnotationCloudServiceWithGateway] = "true"
				endpoints.Annotations[connector.AnnotationCloudServiceWithMultiGateways] = fmt.Sprintf("%t", t.controller.GetC2KMultiGateways())
				if syncer.discClient.IsInternalServices() {
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
				eptClient := syncer.kubeClient.CoreV1().Endpoints(syncer.namespace())
				return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
					if updatedEpt, err := eptClient.Update(syncer.ctx, endpoints, metav1.UpdateOptions{}); err != nil {
						log.Warn().Err(err).Msgf("error update endpoints, name:%s", service.Name)
						return err
					} else {
						t.updateGatewayEndpointSlice(syncer.ctx, updatedEpt)
					}
					return nil
				})
			}
		}
	}
	return fmt.Errorf("error update endpoints, name:%s", endpoints.Name)
}

func (t *endpointsResource) Delete(key string, _ interface{}) error {
	syncer := t.syncer
	syncer.lock.Lock()
	defer syncer.lock.Unlock()

	name, ok := t.controller.GetC2KContext().EndpointsKeyToName[key]
	if !ok {
		return nil
	}

	delete(t.controller.GetC2KContext().EndpointsKeyToName, key)

	log.Info().Msgf("delete endpoints, key:%s name:%s", key, name)
	return nil
}

func (t *endpointsResource) updateGatewayEndpointSlice(ctx context.Context, endpoints *apiv1.Endpoints) {
	endpointsDup := endpoints.DeepCopy()
	syncer := t.syncer
	_ = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		eptSliceClient := syncer.kubeClient.DiscoveryV1().EndpointSlices(endpointsDup.Namespace)
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
			discoveryv1.LabelServiceName: endpointsDup.Name,
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
		var viaAddr string
		var viaPort int32
		if viaGateway, viaGatewayExists := endpointsDup.Annotations[connector.AnnotationCloudServiceViaGateway]; viaGatewayExists {
			if segs := strings.Split(viaGateway, ":"); len(segs) == 2 {
				if port, convErr := strconv.Atoi(segs[1]); convErr == nil {
					viaPort = int32(port & 0xFFFF)
					viaAddr = segs[0]
				}
			}
		}
		for _, subsets := range endpointsDup.Subsets {
			for _, port := range subsets.Ports {
				shadow := port
				if viaPort > 0 {
					ports = append(ports, discoveryv1.EndpointPort{
						Name:        &shadow.Name,
						Protocol:    &shadow.Protocol,
						Port:        &viaPort,
						AppProtocol: shadow.AppProtocol,
					})
				} else {
					ports = append(ports, discoveryv1.EndpointPort{
						Name:        &shadow.Name,
						Protocol:    &shadow.Protocol,
						Port:        &shadow.Port,
						AppProtocol: shadow.AppProtocol,
					})
				}
			}
			if len(subsets.Addresses) > 0 {
				var ready = true
				var addrs []string
				ept := discoveryv1.Endpoint{
					Conditions: discoveryv1.EndpointConditions{
						Ready: &ready,
					},
				}
				if len(viaAddr) > 0 {
					addrs = append(addrs, viaAddr)
				} else {
					for _, addr := range subsets.Addresses {
						addrs = append(addrs, addr.IP)
					}
				}

				ept.Addresses = addrs
				epts = append(epts, ept)
			}
		}
		newEpSlice.Ports = ports
		newEpSlice.Endpoints = epts

		_, err = eptSliceClient.Update(syncer.ctx, newEpSlice, metav1.UpdateOptions{})
		if err != nil {
			log.Error().Msgf("error updating EndpointSlice, name:%s warn:%v", newEpSlice.Name, err)
		}
		return err
	})
}
