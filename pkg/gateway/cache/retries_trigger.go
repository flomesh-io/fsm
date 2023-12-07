package cache

import (
	"sync"

	gwpav1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha1"

	"github.com/flomesh-io/fsm/pkg/gateway/utils"
)

// RetryPoliciesTrigger is responsible for processing TLSRoute objects
type RetryPoliciesTrigger struct {
	mu sync.Mutex
}

// Insert adds a RetryPolicy to the cache and returns true if the target service is routable
func (p *RetryPoliciesTrigger) Insert(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.RetryPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cache.retries[utils.ObjectKey(policy)] = struct{}{}

	return cache.isRoutableTargetService(policy, policy.Spec.TargetRef)
}

// Delete removes a RetryPolicy from the cache and returns true if the policy was found
func (p *RetryPoliciesTrigger) Delete(obj interface{}, cache *GatewayCache) bool {
	policy, ok := obj.(*gwpav1alpha1.RetryPolicy)
	if !ok {
		log.Error().Msgf("unexpected object type %T", obj)
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := utils.ObjectKey(policy)
	_, found := cache.retries[key]
	delete(cache.retries, key)

	return found
}
