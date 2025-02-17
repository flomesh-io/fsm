package repo

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
)

// BroadcastListener listens for broadcast messages from the message broker
func (s *Server) BroadcastListener() {
	// Register for proxy config updates broadcast by the message broker
	gatewayUpdatePubSub := s.msgBroker.GetGatewayUpdatePubSub()
	gatewayUpdateChan := gatewayUpdatePubSub.Sub(announcements.GatewayUpdate.String())
	defer s.msgBroker.Unsub(gatewayUpdatePubSub, gatewayUpdateChan)

	// Wait for one informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 10)
	defer slidingTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-gatewayUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 10)
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			newJob := func() *GatewayConfGeneratorJob {
				return &GatewayConfGeneratorJob{
					processor: s.processor,
					done:      make(chan struct{}),
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
