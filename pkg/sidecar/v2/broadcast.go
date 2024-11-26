package v2

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
)

// BroadcastListener listens for broadcast messages from the message broker
func (s *Server) BroadcastListener(stopCh <-chan struct{}) {
	xnetworkUpdatePubSub := s.msgBroker.GetXNetworkUpdatePubSub()
	xnetworkUpdateChan := xnetworkUpdatePubSub.Sub(announcements.XNetworkUpdate.String())
	defer s.msgBroker.Unsub(xnetworkUpdatePubSub, xnetworkUpdateChan)

	// Wait for one informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 10)
	defer slidingTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-stopCh:
			return
		case <-xnetworkUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 10)
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			newJob := func() *xnetworkConfigJob {
				return &xnetworkConfigJob{
					done:   make(chan struct{}),
					server: s,
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
