package cache

import (
	"sync"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// AccessControlPoliciesTrigger is responsible for processing AccessControlPolicy objects
type AccessControlPoliciesTrigger struct {
	mu sync.Mutex
}

// Insert adds a AccessControlPolicy to the cache and returns true if the target service is routable
func (p *AccessControlPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.AccessControlPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cache.accesscontrols[utils.ObjectKey(policy)] = struct{}{}

	return cache.isEffectiveTargetRef(policy.Spec.TargetRef)
}

// Delete removes a AccessControlPolicy from the cache and returns true if the policy was found
func (p *AccessControlPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.AccessControlPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(policy)
	_, found := cache.accesscontrols[key]
	delete(cache.accesscontrols, key)

	return found
}
