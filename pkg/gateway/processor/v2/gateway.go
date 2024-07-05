package v2

import (
	"fmt"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	corev1 "k8s.io/api/core/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	gwtypes "github.com/flomesh-io/fsm/pkg/gateway/types"
	gwutils "github.com/flomesh-io/fsm/pkg/gateway/utils"
)

func (c *ConfigGenerator) processGateway() *fgwv2.Gateway {
	g2 := &fgwv2.Gateway{}

	if err := gwutils.DeepCopy(g2, c.gateway); err != nil {
		log.Error().Msgf("Failed to copy gateway: %v", err)
		return nil
	}

	// replace listeners with valid listeners
	g2.Spec.Listeners = make([]fgwv2.Listener, 0)
	for _, l := range gwutils.GetValidListenersForGateway(c.gateway) {
		v2l := &fgwv2.Listener{
			Name:     l.Name,
			Hostname: l.Hostname,
			Port:     l.Port,
			Protocol: l.Protocol,
		}

		if l.TLS != nil {
			v2l.TLS = &fgwv2.GatewayTLSConfig{
				Mode:         l.TLS.Mode,
				Certificates: []map[string]string{},
				Options:      l.TLS.Options,
			}

			if l.TLS.FrontendValidation != nil {
				v2l.TLS.FrontendValidation = &fgwv2.FrontendTLSValidation{CACertificates: []map[string]string{}}
			}
		}

		// get certificates and CA certificates
		if c.tls(l) && v2l.TLS != nil {
			c.processCertificates(l, v2l)
			c.processCACerts(l, v2l)
		}

		g2.Spec.Listeners = append(g2.Spec.Listeners, *v2l)
	}

	return g2
}

func (c *ConfigGenerator) tls(l gwtypes.Listener) bool {
	switch l.Protocol {
	case gwv1.HTTPSProtocolType:
		// Terminate
		if l.TLS != nil {
			if l.TLS.Mode == nil || *l.TLS.Mode == gwv1.TLSModeTerminate {
				return true
			}
		}
	case gwv1.TLSProtocolType:
		// Terminate & Passthrough
		if l.TLS != nil {
			if l.TLS.Mode == nil {
				return true
			}

			switch *l.TLS.Mode {
			case gwv1.TLSModeTerminate:
				return true
			}
		}
	}

	return false
}

func (c *ConfigGenerator) processCertificates(l gwtypes.Listener, v2l *fgwv2.Listener) {
	for index, ref := range l.TLS.CertificateRefs {
		secret, err := c.secretRefToSecret(c.gateway, ref)

		if err != nil {
			log.Error().Msgf("Failed to resolve Secret: %s", err)
			continue
		}

		if secret.Type != corev1.SecretTypeTLS {
			log.Warn().Msgf("Secret %s/%s is not of type %s, will be ignored for port %d of Gateway %s/%s",
				secret.Namespace, secret.Name, corev1.SecretTypeTLS,
				l.Port, c.gateway.Namespace, c.gateway.Name)
			continue
		}

		certName := fmt.Sprintf("tls-%d-%d.crt", l.Port, index)
		keyName := fmt.Sprintf("tls-%d-%d.key", l.Port, index)

		v2l.TLS.Certificates = append(v2l.TLS.Certificates, map[string]string{
			corev1.TLSCertKey:       certName,
			corev1.TLSPrivateKeyKey: keyName,
		})

		c.secretFiles[certName] = string(secret.Data[corev1.TLSCertKey])
		c.secretFiles[keyName] = string(secret.Data[corev1.TLSPrivateKeyKey])
	}
}

func (c *ConfigGenerator) processCACerts(l gwtypes.Listener, v2l *fgwv2.Listener) {
	if l.TLS.FrontendValidation != nil && len(l.TLS.FrontendValidation.CACertificateRefs) > 0 {
		for index, ref := range l.TLS.FrontendValidation.CACertificateRefs {
			ca := c.objectRefToCACertificate(c.gateway, ref)

			if len(ca) == 0 {
				continue
			}

			caName := fmt.Sprintf("ca-%d-%d.crt", l.Port, index)

			v2l.TLS.FrontendValidation.CACertificates = append(v2l.TLS.FrontendValidation.CACertificates, map[string]string{
				corev1.ServiceAccountRootCAKey: caName,
			})

			c.secretFiles[caName] = string(ca)
		}
	}
}
