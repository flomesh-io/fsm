package cli

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
)

// BroadcastListener listens for broadcast messages from the message broker
func (c *client) BroadcastListener(stopCh <-chan struct{}) {
	// Register for service config updates broadcast by the message broker
	connectorUpdatePubSub := c.msgBroker.GetConnectorUpdatePubSub()
	connectorUpdateChan := connectorUpdatePubSub.Sub(announcements.ConnectorUpdate.String())
	defer c.msgBroker.Unsub(connectorUpdatePubSub, connectorUpdateChan)

	// Wait for one informer synchronization periods
	slidingTimer := time.NewTimer(c.GetSyncPeriod())
	defer slidingTimer.Stop()

	statusTimer := time.NewTimer(c.GetSyncPeriod())
	defer statusTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-stopCh:
			return
		case <-connectorUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(c.GetSyncPeriod())
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			newJob := func() *connectControllerJob {
				return &connectControllerJob{
					done:              make(chan struct{}),
					connectController: c,
				}
			}
			<-c.msgWorkQueues.AddJob(newJob())

			if reconfirm {
				reconfirm = false
				slidingTimer.Reset(c.GetSyncPeriod())
			}
		case <-statusTimer.C:
			c.updateConnectorStatus()
			statusTimer.Reset(c.GetSyncPeriod())
		}
	}
}
