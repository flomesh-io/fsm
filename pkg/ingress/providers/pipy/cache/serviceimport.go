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
	"fmt"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	fsminformers "github.com/flomesh-io/fsm/pkg/k8s/informers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	utilcache "k8s.io/kubernetes/pkg/proxy/util"
	"reflect"
	"sync"
)

type BaseServiceImportInfo struct {
	address  string
	port     int
	portName string
	protocol corev1.Protocol
}

var _ ServicePort = &BaseServiceImportInfo{}

func (info *BaseServiceImportInfo) String() string {
	return fmt.Sprintf("%s:%d/%s", info.address, info.port, info.protocol)
}

func (info *BaseServiceImportInfo) Address() string {
	return info.address
}

func (info *BaseServiceImportInfo) Port() int {
	return info.port
}

func (info *BaseServiceImportInfo) Protocol() corev1.Protocol {
	return info.protocol
}

type enrichServiceImportInfoFunc func(port *mcsv1alpha1.ServicePort, svcImp *mcsv1alpha1.ServiceImport, info *BaseServiceInfo) ServicePort

type serviceImportChange struct {
	previous          ServiceImportMap
	current           ServiceImportMap
	previousEndpoints EndpointsMap
	currentEndpoints  EndpointsMap
}

type ServiceImportChangeTracker struct {
	lock                    sync.Mutex
	items                   map[types.NamespacedName]*serviceImportChange
	endpointItems           map[types.NamespacedName]*multiClusterEndpointsChange
	enrichServiceImportInfo enrichServiceImportInfoFunc
	enrichEndpointInfo      enrichMultiClusterEndpointFunc
	recorder                events.EventRecorder
	informers               *fsminformers.InformerCollection
}

type ServiceImportMap map[ServicePortName]ServicePort

type enrichMultiClusterEndpointFunc func(info *BaseEndpointInfo) Endpoint

type multiClusterEndpointsChange struct {
	previous MultiClusterEndpointsMap
	current  MultiClusterEndpointsMap
}

type MultiClusterEndpointsMap map[ServicePortName][]Endpoint

type serviceImportInfo struct {
	*BaseServiceInfo
	svcName types.NamespacedName
}

func (sct *ServiceImportChangeTracker) newBaseServiceInfo(port *mcsv1alpha1.ServicePort, svcImp *mcsv1alpha1.ServiceImport) *BaseServiceInfo {
	klog.V(5).Infof("ServiceImport %s/%s, Port %s", svcImp.Namespace, svcImp.Name, port.String())

	clusterIP := ""
	svc, exists := sct.serviceExists(svcImp)
	if exists {
		// ONLY supports IPv4 for now, uses Service ClusterIP, if a Service with same name exists
		clusterIP = utilcache.GetClusterIPByFamily(corev1.IPv4Protocol, svc)
	}

	info := &BaseServiceInfo{
		address:  clusterIP,
		port:     int(port.Port),
		portName: port.Name,
		protocol: port.Protocol,
	}

	return info
}

func NewServiceImportChangeTracker(enrichServiceImportInfo enrichServiceImportInfoFunc, enrichEndpointInfo enrichMultiClusterEndpointFunc, recorder events.EventRecorder, informers *fsminformers.InformerCollection) *ServiceImportChangeTracker {
	return &ServiceImportChangeTracker{
		items:                   make(map[types.NamespacedName]*serviceImportChange),
		endpointItems:           make(map[types.NamespacedName]*multiClusterEndpointsChange),
		enrichServiceImportInfo: enrichServiceImportInfo,
		enrichEndpointInfo:      enrichEndpointInfo,
		recorder:                recorder,
		informers:               informers,
	}
}

func (sct *ServiceImportChangeTracker) Update(previous, current *mcsv1alpha1.ServiceImport) bool {
	svcImp := current
	if svcImp == nil {
		svcImp = previous
	}

	if svcImp == nil {
		return false
	}

	if shouldSkipServiceImport(svcImp) {
		return false
	}

	namespacedName := types.NamespacedName{Namespace: svcImp.Namespace, Name: svcImp.Name}

	sct.lock.Lock()
	defer sct.lock.Unlock()

	// Service changes
	change, exists := sct.items[namespacedName]
	if !exists {
		change = &serviceImportChange{}
		change.previous = sct.serviceImportToServiceMap(previous)
		sct.items[namespacedName] = change
	}
	change.current = sct.serviceImportToServiceMap(current)
	if reflect.DeepEqual(change.previous, change.current) {
		delete(sct.items, namespacedName)
	} else {
		klog.V(2).Infof("Service %s updated: %d ports", namespacedName, len(change.current))
	}

	// Endpoints change
	epChange, exists := sct.endpointItems[namespacedName]
	if !exists {
		epChange = &multiClusterEndpointsChange{}
		epChange.previous = sct.endpointsToEndpointsMap(previous)
		sct.endpointItems[namespacedName] = epChange
	}
	epChange.current = sct.endpointsToEndpointsMap(current)
	if reflect.DeepEqual(epChange.previous, epChange.current) {
		delete(sct.endpointItems, namespacedName)
	} else {
		for spn, eps := range epChange.current {
			klog.V(2).Infof("Service port %s updated: %d endpoints", spn, len(eps))
		}
	}

	return len(sct.items) > 0 || len(sct.endpointItems) > 0
}

func (sm *ServiceImportMap) Update(changes *ServiceImportChangeTracker) {
	sm.apply(changes)
}

func (sm *ServiceImportMap) apply(changes *ServiceImportChangeTracker) {
	changes.lock.Lock()
	defer changes.lock.Unlock()
	for _, change := range changes.items {
		sm.merge(change.current)
		change.previous.filter(change.current)
		sm.unmerge(change.previous)
	}
	changes.items = make(map[types.NamespacedName]*serviceImportChange)
}

func (sm *ServiceImportMap) merge(other ServiceImportMap) sets.String {
	existingPorts := sets.NewString()
	for svcPortName, info := range other {
		existingPorts.Insert(svcPortName.String())
		_, exists := (*sm)[svcPortName]
		if !exists {
			klog.V(1).Infof("Adding new service port %q at %s", svcPortName, info.String())
		} else {
			klog.V(1).Infof("Updating existing service port %q at %s", svcPortName, info.String())
		}
		(*sm)[svcPortName] = info
	}
	return existingPorts
}

func (sm *ServiceImportMap) filter(other ServiceImportMap) {
	for svcPortName := range *sm {
		if _, ok := other[svcPortName]; ok {
			delete(*sm, svcPortName)
		}
	}
}

func (sm *ServiceImportMap) unmerge(other ServiceImportMap) {
	for svcPortName := range other {
		_, exists := (*sm)[svcPortName]
		if exists {
			klog.V(1).Infof("Removing service port %q", svcPortName)
			delete(*sm, svcPortName)
		} else {
			klog.Errorf("Service port %q doesn't exists", svcPortName)
		}
	}
}

func (sct *ServiceImportChangeTracker) serviceImportToServiceMap(svcImp *mcsv1alpha1.ServiceImport) ServiceImportMap {
	if svcImp == nil {
		return nil
	}

	serviceImportMap := make(ServiceImportMap)
	svcName := types.NamespacedName{Namespace: svcImp.Namespace, Name: svcImp.Name}
	for i := range svcImp.Spec.Ports {
		servicePort := &svcImp.Spec.Ports[i]
		svcPortName := ServicePortName{NamespacedName: svcName, Port: servicePort.Name, Protocol: servicePort.Protocol}
		baseSvcInfo := sct.newBaseServiceInfo(servicePort, svcImp)
		if baseSvcInfo == nil {
			// nil means we cannot handle such type of service
			continue
		}
		if sct.enrichServiceImportInfo != nil {
			serviceImportMap[svcPortName] = sct.enrichServiceImportInfo(servicePort, svcImp, baseSvcInfo)
		} else {
			serviceImportMap[svcPortName] = baseSvcInfo
		}
	}

	return serviceImportMap
}

func (sct *ServiceImportChangeTracker) serviceExists(svcImp *mcsv1alpha1.ServiceImport) (*corev1.Service, bool) {
	svc, err := sct.informers.GetListers().Service.Services(svcImp.Namespace).Get(svcImp.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false
		}
		return nil, false
	}

	return svc, true
}

func shouldSkipServiceImport(svcImp *mcsv1alpha1.ServiceImport) bool {
	if svcImp == nil {
		return true
	}

	return false
}

func (sct *ServiceImportChangeTracker) endpointsToEndpointsMap(svcImp *mcsv1alpha1.ServiceImport) MultiClusterEndpointsMap {
	if svcImp == nil {
		return nil
	}

	endpointsMap := make(MultiClusterEndpointsMap)
	for _, port := range svcImp.Spec.Ports {
		svcPortName := ServicePortName{
			NamespacedName: types.NamespacedName{Namespace: svcImp.Namespace, Name: svcImp.Name},
			Port:           port.Name,
			Protocol:       port.Protocol,
		}
		for _, ep := range port.Endpoints {
			baseEndpointInfo := newMultiClusterEndpointInfo(&ep, ep.Target)
			if sct.enrichEndpointInfo != nil {
				endpointsMap[svcPortName] = append(endpointsMap[svcPortName], sct.enrichEndpointInfo(baseEndpointInfo))
			} else {
				endpointsMap[svcPortName] = append(endpointsMap[svcPortName], baseEndpointInfo)
			}
		}
		klog.V(3).Infof("Setting endpoints for %q to %#v", svcPortName, formatEndpointsList(endpointsMap[svcPortName]))
	}

	return endpointsMap
}

func newMultiClusterEndpointInfo(ep *mcsv1alpha1.Endpoint, target mcsv1alpha1.Target) *BaseEndpointInfo {
	return &BaseEndpointInfo{
		Endpoint: fmt.Sprintf("%s:%d%s", target.Host, target.Port, target.Path),
		Cluster:  ep.ClusterKey,
	}
}

func (sct *ServiceImportChangeTracker) checkoutChanges() []*multiClusterEndpointsChange {
	sct.lock.Lock()
	defer sct.lock.Unlock()

	changes := make([]*multiClusterEndpointsChange, 0)
	for _, change := range sct.endpointItems {
		changes = append(changes, change)
	}
	sct.endpointItems = make(map[types.NamespacedName]*multiClusterEndpointsChange)
	return changes
}

func (em MultiClusterEndpointsMap) Update(changes *ServiceImportChangeTracker) {
	em.apply(changes)
}

func (em MultiClusterEndpointsMap) apply(sct *ServiceImportChangeTracker) {
	if sct == nil {
		return
	}

	changes := sct.checkoutChanges()
	for _, change := range changes {
		em.unmerge(change.previous)
		em.merge(change.current)
	}
}

func (em MultiClusterEndpointsMap) merge(other MultiClusterEndpointsMap) {
	for svcPortName := range other {
		em[svcPortName] = other[svcPortName]
	}
}

func (em MultiClusterEndpointsMap) unmerge(other MultiClusterEndpointsMap) {
	for svcPortName := range other {
		delete(em, svcPortName)
	}
}

func enrichServiceImportInfo(port *mcsv1alpha1.ServicePort, svcImp *mcsv1alpha1.ServiceImport, baseInfo *BaseServiceInfo) ServicePort {
	info := &serviceImportInfo{BaseServiceInfo: baseInfo}

	svcName := types.NamespacedName{Namespace: svcImp.Namespace, Name: svcImp.Name}
	info.svcName = svcName

	return info
}
