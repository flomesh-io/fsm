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

package route

import commons "github.com/flomesh-io/fsm/pkg/apis"

type IngressData struct {
	//RouteBase `json:",inline"`
	// Hash
	Hash string `json:"hash" hash:"ignore"`
	// Routes
	Routes []IngressRouteSpec `json:"routes" hash:"set"`
}

type IngressRouteSpec struct {
	RouterSpec   `json:",inline"`
	BalancerSpec `json:",inline"`
	TLSSpec      `json:",inline"`
}

type RouterSpec struct {
	Host    string   `json:"-"`
	Path    string   `json:"-"`
	Service string   `json:"service,omitempty"`
	Rewrite []string `json:"rewrite,omitempty"`
}

type BalancerSpec struct {
	Sticky   bool                 `json:"sticky,omitempty"`
	Balancer commons.AlgoBalancer `json:"balancer,omitempty"`
	Upstream *UpstreamSpec        `json:"upstream,omitempty"`
}

type UpstreamSpec struct {
	Protocol  string             `json:"proto,omitempty"`
	SSLName   string             `json:"sslName,omitempty"`
	SSLCert   *CertificateSpec   `json:"sslCert,omitempty"`
	SSLVerify bool               `json:"sslVerify,omitempty"`
	Endpoints []UpstreamEndpoint `json:"endpoints,omitempty" hash:"set"`
}

type TLSSpec struct {
	IsTLS          bool             `json:"isTLS,omitempty"`
	IsWildcardHost bool             `json:"isWildcardHost,omitempty"`
	VerifyClient   bool             `json:"verifyClient,omitempty"`
	VerifyDepth    int              `json:"verifyDepth,omitempty"`
	Certificate    *CertificateSpec `json:"certificate,omitempty"`
	TrustedCA      *CertificateSpec `json:"trustedCA,omitempty"`
}

type CertificateSpec struct {
	Cert string `json:"cert,omitempty"`
	Key  string `json:"key,omitempty"`
	CA   string `json:"ca,omitempty"`
}

type UpstreamEndpoint struct {
	// IP is the entry's IP.  The IP address protocol corresponds to the HashFamily of IPSet.
	// All entries' IP addresses in the same ip set has same the protocol, IPv4 or IPv6.
	IP string `json:"ip,omitempty"`
	// Port is the entry's Port.
	Port int `json:"port,omitempty"`
	// Protocol is the entry's Protocol.  The protocols of entries in the same ip set are all
	// the same.  The accepted protocols are TCP, UDP and SCTP.
	Protocol string `json:"protocol,omitempty"`
}

type ServiceRoute struct {
	//RouteBase `json:",inline"`
	// Hash
	Hash   string              `json:"hash" hash:"ignore"`
	Routes []ServiceRouteEntry `json:"routes" hash:"set"`
}

type ServiceRouteEntry struct {
	// Name, the name of the service
	Name string `json:"name"`
	// Namespace, the namespace of the service, it has value no matter in cluster/out cluster, but will only be used for in-cluster
	Namespace string `json:"namespace"`
	// Targets
	Targets []Target `json:"targets" hash:"set"`
	// PortName
	PortName string `json:"portName,omitempty"`
}

type Target struct {
	// Address can be IP address if in the same cluster, or ingress address for out cluster route
	Address string `json:"address"`
	// Tag, reserved placeholder for futher features
	Tags map[string]string `json:"tags,omitempty" hash:"set"`
}

type IngressConfig struct {
	TrustedCAs     []string `json:"trustedCAs"`
	TLSConfig      `json:",inline"`
	RouterConfig   `json:",inline"`
	BalancerConfig `json:",inline"`
}

type TLSConfig struct {
	Certificates map[string]TLSSpec `json:"certificates"`
}

type RouterConfig struct {
	Routes map[string]RouterSpec `json:"routes"`
}

type BalancerConfig struct {
	Services map[string]BalancerSpec `json:"services"`
}

type ServiceRegistry struct {
	Services ServiceRegistryEntry `json:"services"`
}

type ServiceRegistryEntry map[string][]string
