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

package helper

import (
	"fmt"
	"github.com/flomesh-io/fsm/pkg/config"
)

func GetProxyProfileParentPath(mc *config.MeshConfig) string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/services

	//return util.EvaluateTemplate(commons.ProxyProfileParentPathTemplate, struct {
	//	Region  string
	//	Zone    string
	//	Group   string
	//	Cluster string
	//}{
	//	Region:  mc.Cluster.Region,
	//	Zone:    mc.Cluster.Zone,
	//	Group:   mc.Cluster.Group,
	//	Cluster: mc.Cluster.Name,
	//})

	return "/local/services"
}

func GetProxyProfilePath(proxyProfile string, mc *config.MeshConfig) string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/pf/{{ .ProxyProfile }}

	//return util.EvaluateTemplate(commons.ProxyProfilePathTemplate, struct {
	//	Region       string
	//	Zone         string
	//	Group        string
	//	Cluster      string
	//	ProxyProfile string
	//}{
	//	Region:       mc.Cluster.Region,
	//	Zone:         mc.Cluster.Zone,
	//	Group:        mc.Cluster.Group,
	//	Cluster:      mc.Cluster.Name,
	//	ProxyProfile: proxyProfile,
	//})

	return fmt.Sprintf("/local/pf/%s", proxyProfile)
}

func GetSidecarPath(proxyProfile string, sidecar string, mc *config.MeshConfig) string {
	// Format:
	//  /{{ .Region }}/{{ .Zone }}/{{ .Group }}/{{ .Cluster }}/sidecars/{{ .ProxyProfile }}/{{ .Sidecar }}

	//return util.EvaluateTemplate(commons.SidecarPathTemplate, struct {
	//	Region       string
	//	Zone         string
	//	Group        string
	//	Cluster      string
	//	ProxyProfile string
	//	Sidecar      string
	//}{
	//	Region:       mc.Cluster.Region,
	//	Zone:         mc.Cluster.Zone,
	//	Group:        mc.Cluster.Group,
	//	Cluster:      mc.Cluster.Name,
	//	ProxyProfile: proxyProfile,
	//	Sidecar:      sidecar,
	//})

	return fmt.Sprintf("/local/sidecars/%s/%s", proxyProfile, sidecar)
}
