package repo

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/metricsstore"
	"github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy"
	"time"
)

func (c *client) broadcastListener() {
	// Register for proxy config updates broadcast by the message broker
	gatewayUpdatePubSub := c.msgBroker.GetGatewayUpdatePubSub()
	gatewayUpdateChan := gatewayUpdatePubSub.Sub(announcements.GatewayUpdate.String())
	defer c.msgBroker.Unsub(gatewayUpdatePubSub, gatewayUpdateChan)

	// Wait for two informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 20)
	defer slidingTimer.Stop()

	slidingTimerReset := func() {
		slidingTimer.Reset(time.Second * 5)
	}

	for {
		select {
		case <-gatewayUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 5)
			// Avoid data omission
			reconfirm = true

		case <-slidingTimer.C:
			connectedProxies := make(map[string]*pipy.Proxy)
			disconnectedProxies := make(map[string]*pipy.Proxy)
			proxies := s.fireExistProxies()
			metricsstore.DefaultMetricsStore.ProxyConnectCount.Set(float64(len(proxies)))
			for _, proxy := range proxies {
				if proxy.PodMetadata == nil {
					if err := s.recordPodMetadata(proxy); err != nil {
						slidingTimer.Reset(time.Second * 5)
						continue
					}
				}
				if proxy.PodMetadata == nil || proxy.Addr == nil || len(proxy.GetAddr()) == 0 {
					slidingTimer.Reset(time.Second * 5)
					continue
				}
				connectedProxies[proxy.UUID.String()] = proxy
			}

			s.proxyRegistry.RangeConnectedProxy(func(key, value interface{}) bool {
				proxyUUID := key.(string)
				if _, exists := connectedProxies[proxyUUID]; !exists {
					disconnectedProxies[proxyUUID] = value.(*pipy.Proxy)
				}
				return true
			})

			if len(connectedProxies) > 0 {
				for _, proxy := range connectedProxies {
					newJob := func() *PipyConfGeneratorJob {
						return &PipyConfGeneratorJob{
							proxy:      proxy,
							repoServer: s,
							done:       make(chan struct{}),
						}
					}
					<-s.workQueues.AddJob(newJob())
				}
			}

			if reconfirm {
				reconfirm = false
				slidingTimer.Reset(time.Second * 10)
			}

			go func() {
				if len(disconnectedProxies) > 0 {
					for _, proxy := range disconnectedProxies {
						s.proxyRegistry.UnregisterProxy(proxy)
						if _, err := s.repoClient.Delete(fmt.Sprintf("%s/%s", fsmSidecarCodebase, proxy.GetCNPrefix())); err != nil {
							log.Debug().Msgf("fail to delete %s/%s", fsmSidecarCodebase, proxy.GetCNPrefix())
						}
					}
				}
			}()
		}
	}
}
