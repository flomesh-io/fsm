/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package event

import (
	"github.com/cskr/pubsub"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

type Broker struct {
	queue      workqueue.RateLimitingInterface
	messageBus *pubsub.PubSub
}

func NewBroker(stopCh <-chan struct{}) *Broker {
	b := &Broker{
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		messageBus: pubsub.New(0),
	}

	go b.runWorkqueueProcessor(stopCh)

	return b
}

func (b *Broker) GetMessageBus() *pubsub.PubSub {
	return b.messageBus
}

//func (b *Broker) GetQueue() workqueue.RateLimitingInterface {
//	return b.queue
//}

func (b *Broker) Enqueue(msg Message) {
	//msg, ok := obj.(Message)
	//if !ok {
	//	klog.Errorf("Received unexpected message %T, expected event.Message", obj)
	//	return
	//}

	b.queue.AddRateLimited(msg)
}

func (b *Broker) Unsub(pubSub *pubsub.PubSub, ch chan interface{}) {
	go pubSub.Unsub(ch)
	for range ch {
		// Drain channel until 'Unsub' results in a close on the subscribed channel
	}
}

func (b *Broker) runWorkqueueProcessor(stopCh <-chan struct{}) {
	go wait.Until(
		func() {
			for b.processNextItem() {
			}
		},
		time.Second,
		stopCh,
	)
}

func (b *Broker) processNextItem() bool {
	item, shutdown := b.queue.Get()
	if shutdown {
		return false
	}

	defer b.queue.Done(item)

	msg, ok := item.(Message)
	if !ok {
		b.queue.Forget(item)
		// Process next item in the queue
		return true
	}

	b.processEvent(msg)
	b.queue.Forget(item)

	return true
}

func (b *Broker) processEvent(msg Message) {
	klog.V(5).Infof("Processing event %v", msg)
	b.messageBus.Pub(msg, string(msg.Kind))
}
