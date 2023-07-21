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

import "github.com/flomesh-io/fsm/pkg/gateway/route"

type ProcessorType string

const (
	ServicesProcessorType       ProcessorType = "services"
	EndpointSlicesProcessorType ProcessorType = "endpointslices"
	EndpointsProcessorType      ProcessorType = "endpoints"
	ServiceImportsProcessorType ProcessorType = "serviceimports"
	SecretsProcessorType        ProcessorType = "secrets"
	GatewayClassesProcessorType ProcessorType = "gatewayclasses"
	GatewaysProcessorType       ProcessorType = "gateways"
	HTTPRoutesProcessorType     ProcessorType = "httproutes"
	GRPCRoutesProcessorType     ProcessorType = "grpcroutes"
	TCPRoutesProcessorType      ProcessorType = "tcproutes"
	TLSRoutesProcessorType      ProcessorType = "tlsroutes"
)

type Processor interface {
	Insert(obj interface{}, cache *GatewayCache) bool
	Delete(obj interface{}, cache *GatewayCache) bool
}

type Cache interface {
	Insert(obj interface{}) bool
	Delete(obj interface{}) bool
	BuildConfigs()
}

type serviceInfo struct {
	svcPortName route.ServicePortName
	filters     []route.Filter
}

type endpointInfo struct {
	address string
	port    int32
}
