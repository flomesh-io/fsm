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
	"github.com/flomesh-io/fsm-classic/pkg/mcs/config"
	mcsv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/multicluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type EventType string

const (
	ServiceExportCreated  EventType = "service.export.created"
	ServiceExportDeleted  EventType = "service.export.deleted"
	ServiceExportAccepted EventType = "service.export.accepted"
	ServiceExportRejected EventType = "service.export.rejected"
)

type Message struct {
	Kind   EventType
	OldObj interface{}
	NewObj interface{}
}

//type GeoInfo struct {
//	Region  string
//	Zone    string
//	Group   string
//	Cluster string
//}

type ServiceExportEvent struct {
	Geo           *config.ConnectorConfig
	ServiceExport *mcsv1alpha1.ServiceExport
	Service       *corev1.Service
	Error         string
	//Data          map[string]interface{}
}

func (e *ServiceExportEvent) ClusterKey() string {
	return e.Geo.Key()
}

//func NewServiceExportMessage(eventType EventType, geo *config.ConnectorConfig, serviceExport *mcsv1alpha1.ServiceExport, svc *corev1.Service, data map[string]interface{}) *Message {
//	obj := ServiceExportEvent{Geo: geo, ServiceExport: serviceExport, Service: svc, Data: data}
//
//	switch eventType {
//	case ServiceExportAccepted, ServiceExportCreated, ServiceExportRejected:
//		return &Message{
//			Kind:   eventType,
//			OldObj: nil,
//			NewObj: obj,
//		}
//	case ServiceExportDeleted:
//		return &Message{
//			Kind:   eventType,
//			OldObj: obj,
//			NewObj: nil,
//		}
//	}
//
//	return nil
//}
