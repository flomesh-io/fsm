package cli

import (
	"sync"
	"time"

	"github.com/mitchellh/hashstructure/v2"

	"github.com/flomesh-io/fsm/pkg/connector"
)

const (
	retries = 1
)

type catalogTimeScale struct {
	lock      sync.Mutex
	refreshTs time.Time
	result    interface{}

	lastAccessTs time.Time
}

type registerTimeScale struct {
	registeredHash    uint64
	registeredTs      time.Time
	registeredRetries int

	deregisteredTs      time.Time
	deregisteredRetries int

	lastAccessTs time.Time
}

type cache struct {
	config

	cacheLock sync.Mutex

	catalogInstances    connector.ConcurrentMap[string, *catalogTimeScale]
	registeredInstances connector.ConcurrentMap[string, *registerTimeScale]
}

func (c *cache) getCatalogInstanceTimeScale(key string) *catalogTimeScale {
	ts, ok := c.catalogInstances.Get(key)
	if !ok {
		c.cacheLock.Lock()
		defer c.cacheLock.Unlock()
		if !c.catalogInstances.Has(key) {
			c.catalogInstances.Set(key, &catalogTimeScale{})
		}
		ts, _ = c.catalogInstances.Get(key)
	}
	ts.lastAccessTs = time.Now()
	return ts
}

func (c *cache) CacheCatalogInstances(key string, catalogFunc func() (interface{}, error)) (interface{}, error) {
	ts := c.getCatalogInstanceTimeScale(key)
	ts.lock.Lock()
	defer ts.lock.Unlock()
	if ts.result != nil && c.GetSyncPeriod() > time.Since(ts.refreshTs) {
		return ts.result, nil
	}
	result, err := catalogFunc()
	if err == nil {
		ts.result = result
		ts.refreshTs = time.Now()
	}
	return result, err
}

func (c *cache) getRegisteredInstanceTimeScale(key string) *registerTimeScale {
	ts, ok := c.registeredInstances.Get(key)
	if !ok {
		c.cacheLock.Lock()
		defer c.cacheLock.Unlock()
		if !c.registeredInstances.Has(key) {
			c.registeredInstances.Set(key, &registerTimeScale{})
		}
		ts, _ = c.registeredInstances.Get(key)
	}
	ts.lastAccessTs = time.Now()
	return ts
}

func (c *cache) CacheRegisterInstance(key string, instance interface{}, registerFunc func() error) error {
	ts := c.getRegisteredInstanceTimeScale(key)
	hash, err := hashstructure.Hash(instance, hashstructure.FormatV2,
		&hashstructure.HashOptions{
			ZeroNil:         true,
			IgnoreZeroValue: true,
			SlicesAsSets:    true,
		})
	if err == nil {
		if ts.registeredHash == hash {
			if ts.registeredTs.Before(ts.deregisteredTs) {
				ts.registeredRetries = 0
			}
			if ts.registeredRetries > retries {
				return nil
			}
		} else {
			ts.registeredRetries = 0
		}
	} else {
		ts.registeredRetries = 0
		log.Warn().Err(err)
	}
	err = registerFunc()
	if err == nil {
		ts.registeredRetries++
		ts.registeredHash = hash
		ts.registeredTs = time.Now()
	}
	return err
}

func (c *cache) CacheDeregisterInstance(key string, deregisterFunc func() error) error {
	ts := c.getRegisteredInstanceTimeScale(key)
	if ts.deregisteredTs.After(ts.registeredTs) {
		ts.deregisteredRetries = 0
	}
	if ts.deregisteredRetries > retries {
		return nil
	}
	err := deregisterFunc()
	if err == nil {
		ts.deregisteredRetries++
		ts.deregisteredTs = time.Now()
	}
	return err
}

func (c *cache) clean() {
	expirePeriod := 3 * c.GetSyncPeriod()
	for item := range c.registeredInstances.IterBuffered() {
		if time.Since(item.Val.lastAccessTs) > expirePeriod {
			c.registeredInstances.Remove(item.Key)
		}
	}
	for item := range c.catalogInstances.IterBuffered() {
		if time.Since(item.Val.lastAccessTs) > expirePeriod {
			c.catalogInstances.Remove(item.Key)
		}
	}
}

func (c *cache) CacheCleaner(stopCh <-chan struct{}) {
	cleanTimer := time.NewTimer(3 * c.GetSyncPeriod())
	defer cleanTimer.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-cleanTimer.C:
			c.clean()
			cleanTimer.Reset(3 * c.GetSyncPeriod())
		}
	}
}
