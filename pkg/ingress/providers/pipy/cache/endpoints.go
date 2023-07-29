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

package cache

import (
	"net"
	"reflect"
	"strconv"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	utilcache "k8s.io/kubernetes/pkg/proxy/util"
	utilnet "k8s.io/utils/net"
)

type baseEndpointInfo struct {
	Endpoint string
	Nodename string
	Hostname string
	Cluster  string
}

var _ Endpoint = &baseEndpointInfo{}

func (info *baseEndpointInfo) String() string {
	return info.Endpoint
}

func (info *baseEndpointInfo) IP() string {
	return utilcache.IPPart(info.Endpoint)
}

func (info *baseEndpointInfo) Port() (int, error) {
	return utilcache.PortPart(info.Endpoint)
}

func (info *baseEndpointInfo) NodeName() string {
	return info.Nodename
}

func (info *baseEndpointInfo) HostName() string {
	return info.Hostname
}

func (info *baseEndpointInfo) ClusterInfo() string {
	return info.Cluster
}

func (info *baseEndpointInfo) Equal(other Endpoint) bool {
	return info.String() == other.String()
}

func newBaseEndpointInfo(IP string, port int, nodename string, hostname string) *baseEndpointInfo {
	return &baseEndpointInfo{
		Endpoint: net.JoinHostPort(IP, strconv.Itoa(port)),
		Nodename: nodename,
		Hostname: hostname,
	}
}

type enrichEndpointFunc func(info *baseEndpointInfo) Endpoint

// EndpointChangeTracker tracks changes in endpoints
type EndpointChangeTracker struct {
	lock               sync.Mutex
	items              map[types.NamespacedName]*endpointsChange
	enrichEndpointInfo enrichEndpointFunc
	recorder           events.EventRecorder
}

// NewEndpointChangeTracker creates a new EndpointChangeTracker
func NewEndpointChangeTracker(enrichEndpointInfo enrichEndpointFunc, recorder events.EventRecorder) *EndpointChangeTracker {
	return &EndpointChangeTracker{
		items:              make(map[types.NamespacedName]*endpointsChange),
		enrichEndpointInfo: enrichEndpointInfo,
		recorder:           recorder,
	}
}

// Update updates the tracker with the current endpoints
func (t *EndpointChangeTracker) Update(previous, current *corev1.Endpoints) bool {
	endpoints := current
	if endpoints == nil {
		endpoints = previous
	}
	if endpoints == nil {
		return false
	}

	namespacedName := types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name}

	t.lock.Lock()
	defer t.lock.Unlock()

	change, exists := t.items[namespacedName]
	if !exists {
		change = &endpointsChange{}
		change.previous = t.endpointsToEndpointsMap(previous)
		t.items[namespacedName] = change
	}

	change.current = t.endpointsToEndpointsMap(current)

	if reflect.DeepEqual(change.previous, change.current) {
		delete(t.items, namespacedName)
	} else {
		for spn, eps := range change.current {
			log.Info().Msgf("Service port %s updated: %d endpoints", spn, len(eps))
		}
	}

	return len(t.items) > 0
}

func (t *EndpointChangeTracker) checkoutChanges() []*endpointsChange {
	t.lock.Lock()
	defer t.lock.Unlock()

	changes := []*endpointsChange{}
	for _, change := range t.items {
		changes = append(changes, change)
	}
	t.items = make(map[types.NamespacedName]*endpointsChange)
	return changes
}

type endpointsChange struct {
	previous EndpointsMap
	current  EndpointsMap
}

// Update updates the endpoints map with the changes
func (em EndpointsMap) Update(changes *EndpointChangeTracker) {
	em.apply(changes)
}

// EndpointsMap is a map of service port name to endpoints
type EndpointsMap map[ServicePortName][]Endpoint

func (t *EndpointChangeTracker) endpointsToEndpointsMap(endpoints *corev1.Endpoints) EndpointsMap {
	if endpoints == nil {
		return nil
	}

	endpointsMap := make(EndpointsMap)
	for i := range endpoints.Subsets {
		ss := &endpoints.Subsets[i]
		for i := range ss.Ports {
			port := &ss.Ports[i]
			if port.Port == 0 {
				log.Warn().Msgf("ignoring invalid endpoint port %s", port.Name)
				continue
			}
			svcPortName := ServicePortName{
				NamespacedName: types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name},
				Port:           port.Name,
				Protocol:       port.Protocol,
			}
			for i := range ss.Addresses {
				addr := &ss.Addresses[i]
				if addr.IP == "" {
					log.Warn().Msgf("ignoring invalid endpoint port %s with empty host", port.Name)
					continue
				}

				// ONLY supports IPv4
				if !utilnet.IsIPv4String(addr.IP) {
					continue
				}

				log.Info().Msgf("Address = %v", addr)

				baseEndpointInfo := newBaseEndpointInfo(addr.IP, int(port.Port), nodename(addr), addr.Hostname)
				if t.enrichEndpointInfo != nil {
					endpointsMap[svcPortName] = append(endpointsMap[svcPortName], t.enrichEndpointInfo(baseEndpointInfo))
				} else {
					endpointsMap[svcPortName] = append(endpointsMap[svcPortName], baseEndpointInfo)
				}
			}

			log.Info().Msgf("Setting endpoints for %q to %v", svcPortName, formatEndpointsList(endpointsMap[svcPortName]))
		}
	}

	return endpointsMap
}

func nodename(addr *corev1.EndpointAddress) string {
	if addr == nil {
		return ""
	}

	nodeName := ""
	if addr.NodeName != nil {
		nodeName = *addr.NodeName
	}

	return nodeName
}

func formatEndpointsList(endpoints []Endpoint) []string {
	var formattedList []string
	for _, ep := range endpoints {
		formattedList = append(formattedList, ep.String())
	}
	return formattedList
}

func (em EndpointsMap) apply(t *EndpointChangeTracker) {
	if t == nil {
		return
	}

	changes := t.checkoutChanges()
	for _, change := range changes {
		em.unmerge(change.previous)
		em.merge(change.current)
	}
}

func (em EndpointsMap) merge(other EndpointsMap) {
	for svcPortName := range other {
		em[svcPortName] = other[svcPortName]
	}
}

func (em EndpointsMap) unmerge(other EndpointsMap) {
	for svcPortName := range other {
		delete(em, svcPortName)
	}
}
