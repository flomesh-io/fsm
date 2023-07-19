package repo

import (
	"github.com/flomesh-io/fsm/pkg/announcements"
	"time"
)

func (s *Server) BroadcastListener() {
	// Register for proxy config updates broadcast by the message broker
	ingressUpdatePubSub := s.msgBroker.GetIngressUpdatePubSub()
	ingressUpdateChan := ingressUpdatePubSub.Sub(announcements.IngressUpdate.String())
	defer s.msgBroker.Unsub(ingressUpdatePubSub, ingressUpdateChan)

	// Wait for two informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 20)
	defer slidingTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-ingressUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 5)
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			//metricsstore.DefaultMetricsStore.ProxyConnectCount.Set(float64(len(proxies)))
			newJob := func() *IngressConfGeneratorJob {
				return &IngressConfGeneratorJob{
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
