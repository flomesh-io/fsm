package ctok

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/flomesh-io/fsm/pkg/connector"
)

type EndpointsSource struct {
	informer cache.SharedIndexInformer
}

func (t *EndpointsSource) Informer() cache.SharedIndexInformer {
	return t.informer
}

func (t *EndpointsSource) Upsert(key string, raw interface{}) error {
	return nil
}

func (t *EndpointsSource) Delete(key string, raw interface{}) error {
	return nil
}

func (s *CtoKSyncer) EndpointsSource() connector.Resource {
	if s.eptInformer == nil {
		s.eptInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					endpointsList, err := s.kubeClient.CoreV1().Endpoints(s.namespace()).List(s.ctx, options)
					if err != nil {
						log.Error().Msgf("cache.NewSharedIndexInformer Endpoints ListFunc:%v", err)
					}
					return endpointsList, err
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					endpoints, err := s.kubeClient.CoreV1().Endpoints(s.namespace()).Watch(s.ctx, options)
					if err != nil {
						log.Error().Msgf("cache.NewSharedIndexInformer Endpoints WatchFunc:%v", err)
					}
					return endpoints, err
				},
			},
			&corev1.Endpoints{},
			0,
			cache.Indexers{},
		)
	}
	return &EndpointsSource{
		informer: s.eptInformer,
	}
}
