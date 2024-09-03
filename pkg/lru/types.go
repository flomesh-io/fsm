package lru

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
)

var (
	lock     sync.Mutex
	lruCache *expirable.LRU[string, interface{}]
)

func Get(key string) (interface{}, bool) {
	if lruCache == nil {
		lock.Lock()
		if lruCache == nil {
			lruCache = expirable.NewLRU[string, interface{}](1024*256, nil, time.Second*600)
		}
		lock.Unlock()
	}
	return lruCache.Get(key)
}

func Add(key string, value interface{}) bool {
	if lruCache == nil {
		lock.Lock()
		if lruCache == nil {
			lruCache = expirable.NewLRU[string, interface{}](1024*256, nil, time.Second*600)
		}
		lock.Unlock()
	}
	return lruCache.Add(key, value)
}

func MicroSvcMetaExists(svc *corev1.Service) bool {
	hash := svc.Annotations[constants.AnnotationMeshEndpointHash]
	key := fmt.Sprintf("%s.%s.%s", svc.Namespace, svc.Name, hash)
	_, exists := Get(key)
	return exists
}
