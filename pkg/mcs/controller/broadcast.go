package controller

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	"time"
)

func (s *Server) BroadcastListener() {
	// Register for proxy config updates broadcast by the message broker
	gatewayUpdatePubSub := s.msgBroker.GetGatewayUpdatePubSub()
	gatewayUpdateChan := gatewayUpdatePubSub.Sub(announcements.GatewayUpdate.String())
	defer s.msgBroker.Unsub(gatewayUpdatePubSub, gatewayUpdateChan)

	// Wait for two informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 20)
	defer slidingTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-gatewayUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 5)
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			//metricsstore.DefaultMetricsStore.ProxyConnectCount.Set(float64(len(proxies)))
			newJob := func() *GatewayConfGeneratorJob {
				return &GatewayConfGeneratorJob{
					cache: s.cache,
					done:  make(chan struct{}),
				}
			}
			<-s.workQueues.AddJob(newJob())

			if reconfirm {
				reconfirm = false
				slidingTimer.Reset(time.Second * 10)
			}
		}
	}
}
