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
	commons "github.com/flomesh-io/fsm-classic/apis"
	repocfg "github.com/flomesh-io/fsm-classic/pkg/route"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Route , Ingress Route interface
type Route interface {
	String() string
	Headers() map[string]string
	Host() string
	Path() string
	Backend() ServicePortName
	Rewrite() []string
	SessionSticky() bool
	LBType() commons.AlgoBalancer
	UpstreamSSLName() string
	UpstreamSSLCert() *repocfg.CertificateSpec
	UpstreamSSLVerify() bool
	Certificate() *repocfg.CertificateSpec
	IsTLS() bool
	IsWildcardHost() bool
	VerifyClient() bool
	VerifyDepth() int
	TrustedCA() *repocfg.CertificateSpec
	Protocol() string
}

type ServicePortName struct {
	types.NamespacedName
	Port     string
	Protocol v1.Protocol
}

func (spn ServicePortName) String() string {
	return fmt.Sprintf("%s%s", spn.NamespacedName.String(), fmtPortName(spn.Port))
}

func fmtPortName(in string) string {
	if in == "" {
		return ""
	}
	return fmt.Sprintf(":%s", in)
}

type ServicePort interface {
	String() string
	Address() string
	Port() int
	Protocol() v1.Protocol
}

type Endpoint interface {
	String() string
	IP() string
	Port() (int, error)
	NodeName() string
	HostName() string
	ClusterInfo() string
	Equal(Endpoint) bool
}

type ServiceEndpoint struct {
	Endpoint        string
	ServicePortName ServicePortName
}
