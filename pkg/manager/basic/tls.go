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

package basic

import (
	"github.com/flomesh-io/fsm/pkg/constants"
	fctx "github.com/flomesh-io/fsm/pkg/context"
	"github.com/flomesh-io/fsm/pkg/manager/utils"
)

func SetupTLS(ctx *fctx.ControllerContext) error {
	log.Info().Msgf("[MGR] Setting up TLS ...")

	mc := ctx.Config
	//log.Info().Msgf("mc.Ingress.TLS=%v", mc.Ingress.TLS)

	if mc.IsIngressTLSEnabled() {
		if mc.IsIngressSSLPassthroughEnabled() {
			// SSL Passthrough
			if err := utils.UpdateSSLPassthrough(
				constants.DefaultIngressBasePath,
				ctx.RepoClient,
				mc.IsIngressSSLPassthroughEnabled(),
				mc.GetIngressSSLPassthroughUpstreamPort(),
			); err != nil {
				return err
			}
		} else {
			// TLS Offload
			if err := utils.IssueCertForIngress(constants.DefaultIngressBasePath, ctx.RepoClient, ctx.CertificateManager, mc); err != nil {
				return err
			}
		}
	}

	return nil
}
