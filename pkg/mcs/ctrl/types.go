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

// Package ctrl is the control plane server
package ctrl

import (
	"sync"

	"github.com/flomesh-io/fsm/pkg/configurator"
	conn "github.com/flomesh-io/fsm/pkg/mcs/remote"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/workerpool"
)

const (
	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0
)

// ControlPlaneServer is the control plane server
type ControlPlaneServer struct {
	cfg         configurator.Configurator
	msgBroker   *messaging.Broker
	workQueues  *workerpool.WorkerPool
	mu          sync.Mutex
	backgrounds map[string]*conn.Background
}

// NewControlPlaneServer creates a new ControlPlaneServer
func NewControlPlaneServer(cfg configurator.Configurator, msgBroker *messaging.Broker) *ControlPlaneServer {
	return &ControlPlaneServer{
		cfg:         cfg,
		msgBroker:   msgBroker,
		workQueues:  workerpool.NewWorkerPool(workerPoolSize),
		backgrounds: make(map[string]*conn.Background),
	}
}
