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

package config

import (
	"fmt"
	"net"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/utils"
)

// ConnectorConfig is the configuration for the connector
type ConnectorConfig struct {
	name            string
	region          string
	zone            string
	group           string
	key             string
	gatewayHost     string
	gatewayIP       net.IP
	gatewayPort     int32
	controlPlaneUID string
}

// NewConnectorConfig creates a new ConnectorConfig
func NewConnectorConfig(
	region, zone, group, name, gatewayHost string,
	gatewayPort int32,
	controlPlaneUID string,
) (*ConnectorConfig, error) {
	clusterKey := utils.EvaluateTemplate(constants.ClusterIDTemplate, struct {
		Region  string
		Zone    string
		Group   string
		Cluster string
	}{
		Region:  region,
		Zone:    zone,
		Group:   group,
		Cluster: name,
	})

	c := &ConnectorConfig{
		region:          region,
		zone:            zone,
		group:           group,
		name:            name,
		key:             clusterKey,
		controlPlaneUID: controlPlaneUID,
	}

	//if !inCluster {
	isDNSName := false
	if ipErrs := validation.IsValidIP(field.NewPath(""), gatewayHost); len(ipErrs) > 0 {
		// Not IPv4 address
		log.Warn().Msgf("%q is NOT a valid IP address: %v", gatewayHost, ipErrs)
		if dnsErrs := validation.IsDNS1123Subdomain(gatewayHost); len(dnsErrs) > 0 {
			// Not valid DNS domain name
			return nil, fmt.Errorf("invalid DNS name or IP %q: %v", gatewayHost, dnsErrs)
		}

		// is DNS name
		isDNSName = true
	}

	var gwIP net.IP
	if isDNSName {
		ipAddr, err := net.ResolveIPAddr("ip", gatewayHost)
		if err != nil {
			return nil, fmt.Errorf("%q cannot be resolved to IP, %s", gatewayHost, err)
		}
		log.Info().Msgf("%q is resolved to IP: %s", gatewayHost, ipAddr.IP)
		gwIP = ipAddr.IP
	} else {
		gwIP = net.ParseIP(gatewayHost)
	}

	if gwIP == nil {
		return nil, fmt.Errorf("%q cannot be resolved to an IP address", gatewayHost)
	}

	if gwIP.IsLoopback() || gwIP.IsUnspecified() {
		return nil, fmt.Errorf("gateway Host %s is resolved to Loopback IP or Unspecified", gatewayHost)
	}

	c.gatewayHost = gatewayHost
	c.gatewayPort = gatewayPort
	c.gatewayIP = gwIP
	//}

	return c, nil
}

// Name returns the name of the connected cluster
func (c *ConnectorConfig) Name() string {
	return c.name
}

// Region returns the region of the connected cluster
func (c *ConnectorConfig) Region() string {
	return c.region
}

// Zone returns the zone of the connected cluster
func (c *ConnectorConfig) Zone() string {
	return c.zone
}

// Group returns the group of the connected cluster
func (c *ConnectorConfig) Group() string {
	return c.group
}

//func (c *ConnectorConfig) IsInCluster() bool {
//	return c.inCluster
//}

// Key returns the key of the connected cluster
func (c *ConnectorConfig) Key() string {
	return c.key
}

// GatewayHost returns the gateway host of the connected cluster
func (c *ConnectorConfig) GatewayHost() string {
	//if c.inCluster {
	//	return ""
	//}
	return c.gatewayHost
}

// GatewayIP returns the gateway IP of the connected cluster
func (c *ConnectorConfig) GatewayIP() net.IP {
	//if c.inCluster {
	//	return net.IPv4zero
	//}
	return c.gatewayIP
}

// GatewayPort returns the gateway port of the connected cluster
func (c *ConnectorConfig) GatewayPort() int32 {
	//if c.inCluster {
	//	return 0
	//}
	return c.gatewayPort
}

// ControlPlaneUID returns the control plane UID of the connected cluster
func (c *ConnectorConfig) ControlPlaneUID() string {
	return c.controlPlaneUID
}
