package v2

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
)

// BroadcastListener listens for broadcast messages from the message broker
func (s *Server) BroadcastListener(stopCh <-chan struct{}) {
	xnetworkUpdatePubSub := s.msgBroker.GetXNetworkUpdatePubSub()
	xnetworkUpdateChan := xnetworkUpdatePubSub.Sub(announcements.XNetworkUpdate.String())

	// Wait for one informer synchronization periods
	meshSlidingTimer := time.NewTimer(time.Second * 10)

	if !s.enableMesh {
		s.msgBroker.Unsub(xnetworkUpdatePubSub, xnetworkUpdateChan)
		meshSlidingTimer.Stop()
	} else {
		defer s.msgBroker.Unsub(xnetworkUpdatePubSub, xnetworkUpdateChan)
		defer meshSlidingTimer.Stop()
	}

	e4lbScheduleTimer := time.NewTimer(time.Second * 5)
	eipScheduleTimer := time.NewTimer(time.Second * 2)
	if !s.enableE4lb {
		e4lbScheduleTimer.Stop()
		eipScheduleTimer.Stop()
	} else {
		defer e4lbScheduleTimer.Stop()
		defer eipScheduleTimer.Stop()
	}

	reconfirm := true

	for {
		select {
		case <-stopCh:
			return
		case <-xnetworkUpdateChan:
			if !reconfirm {
				// Wait for an informer synchronization period
				meshSlidingTimer.Reset(time.Second * 10)
				// Avoid data omission
				reconfirm = true
			}
		case <-meshSlidingTimer.C:
			if s.enableMesh {
				newJob := func() *xnetworkMeshJob {
					return &xnetworkMeshJob{
						done:   make(chan struct{}),
						server: s,
					}
				}
				<-s.workQueues.AddJob(newJob())
			}
			if reconfirm {
				reconfirm = false
				meshSlidingTimer.Reset(time.Second * 10)
			}
		case <-e4lbScheduleTimer.C:
			if s.enableE4lb {
				newJob := func() *xnetworkE4lbJob {
					return &xnetworkE4lbJob{
						done:   make(chan struct{}),
						server: s,
					}
				}
				<-s.workQueues.AddJob(newJob())
			}
			e4lbScheduleTimer.Reset(time.Second * 5)
		case <-eipScheduleTimer.C:
			if s.enableE4lb {
				newJob := func() *xnetworkEIPJob {
					return &xnetworkEIPJob{
						done:   make(chan struct{}),
						server: s,
					}
				}
				<-s.workQueues.AddJob(newJob())
			}
			eipScheduleTimer.Reset(time.Second * 2)
		}
	}
}
