package cli

import (
	"time"

	"github.com/flomesh-io/fsm/pkg/announcements"
)

// BroadcastListener listens for broadcast messages from the message broker
func (c *client) BroadcastListener() {
	// Register for service config updates broadcast by the message broker
	connectorUpdatePubSub := c.msgBroker.GetConnectorUpdatePubSub()
	connectorUpdateChan := connectorUpdatePubSub.Sub(announcements.ConnectorUpdate.String())
	defer c.msgBroker.Unsub(connectorUpdatePubSub, connectorUpdateChan)

	// Wait for two informer synchronization periods
	slidingTimer := time.NewTimer(time.Second * 20)
	defer slidingTimer.Stop()

	reconfirm := true

	for {
		select {
		case <-connectorUpdateChan:
			// Wait for an informer synchronization period
			slidingTimer.Reset(time.Second * 5)
			// Avoid data omission
			reconfirm = true
		case <-slidingTimer.C:
			newJob := func() *connectorControllerJob {
				return &connectorControllerJob{
					done:                make(chan struct{}),
					connectorController: c,
				}
			}
			<-c.msgWorkQueues.AddJob(newJob())

			if reconfirm {
				reconfirm = false
				slidingTimer.Reset(time.Second * 10)
			}
		}
	}
}
