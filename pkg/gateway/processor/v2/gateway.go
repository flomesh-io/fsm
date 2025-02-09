package v2

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"

	fgwv2 "github.com/flomesh-io/fsm/pkg/gateway/fgw"

	corev1 "k8s.io/api/core/v1"

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
		if gwutils.IsTLSListener(l.Listener) && v2l.TLS != nil {
			c.processCertificates(l, v2l)
			c.processCACerts(l, v2l)
		}

		// process listener filters
		c.processListenerFilters(l, v2l)

		g2.Spec.Listeners = append(g2.Spec.Listeners, *v2l)
	}

	c.processGatewayBackendTLS(g2)

	return g2
}

func (c *ConfigGenerator) processCertificates(l gwtypes.Listener, v2l *fgwv2.Listener) {
	resolver := gwutils.NewSecretReferenceResolver(&DummySecretReferenceConditionProvider{}, c.client)

	for index, ref := range l.TLS.CertificateRefs {
		secret, err := resolver.SecretRefToSecret(c.gateway, ref)

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
		resolver := gwutils.NewObjectReferenceResolver(&DummyObjectReferenceConditionProvider{}, c.client)

		for index, ref := range l.TLS.FrontendValidation.CACertificateRefs {
			ca := resolver.ObjectRefToCACertificate(c.gateway, ref)

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

func (c *ConfigGenerator) processListenerFilters(l gwtypes.Listener, v2l *fgwv2.Listener) {
	list := &extv1alpha1.ListenerFilterList{}
	if err := c.client.List(context.Background(), list, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(constants.GatewayListenerFilterIndex, fmt.Sprintf("%s/%d", c.gateway.Name, l.Port)),
		Namespace:     c.gateway.Namespace,
	}); err != nil {
		return
	}

	if len(list.Items) == 0 {
		return
	}

	filters := make([]extv1alpha1.ListenerFilter, 0)
	routeFilters := make([]extv1alpha1.ListenerFilter, 0)

	for _, f := range list.Items {
		aspect := ptr.Deref(f.Spec.Aspect, extv1alpha1.FilterAspectListener)

		switch aspect {
		case extv1alpha1.FilterAspectListener:
			filters = append(filters, f)
		case extv1alpha1.FilterAspectRoute:
			routeFilters = append(routeFilters, f)
		default:
			continue
		}
	}

	v2l.Filters = c.resolveListenerFilters(filters)
	v2l.RouteFilters = c.resolveListenerFilters(routeFilters)
}

func (c *ConfigGenerator) resolveListenerFilters(filters []extv1alpha1.ListenerFilter) []fgwv2.ListenerFilter {
	sort.Slice(filters, func(i, j int) bool {
		if filters[i].Spec.Priority == nil || filters[j].Spec.Priority == nil {
			return filters[i].Spec.Type < filters[j].Spec.Type
		}

		if *filters[i].Spec.Priority == *filters[j].Spec.Priority {
			return filters[i].Spec.Type < filters[j].Spec.Type
		}

		return *filters[i].Spec.Priority < *filters[j].Spec.Priority
	})

	result := make([]fgwv2.ListenerFilter, 0)
	for _, f := range filters {
		filterType := f.Spec.Type

		result = append(result, fgwv2.ListenerFilter{
			Type:            filterType,
			ExtensionConfig: c.resolveFilterConfig(f.Namespace, f.Spec.ConfigRef),
			Key:             uuid.NewString(),
		})

		definition := c.resolveFilterDefinition(filterType, extv1alpha1.FilterScopeListener, f.Spec.DefinitionRef)
		if definition == nil {
			continue
		}

		filterProtocol := ptr.Deref(definition.Spec.Protocol, extv1alpha1.FilterProtocolHTTP)
		if c.filters[filterProtocol] == nil {
			c.filters[filterProtocol] = map[extv1alpha1.FilterType]string{}
		}
		if _, ok := c.filters[filterProtocol][filterType]; !ok {
			c.filters[filterProtocol][filterType] = definition.Spec.Script
		}
	}

	return result
}

func (c *ConfigGenerator) processGatewayBackendTLS(g2 *fgwv2.Gateway) {
	if c.gateway.Spec.BackendTLS != nil && c.gateway.Spec.BackendTLS.ClientCertificateRef != nil {
		ref := c.gateway.Spec.BackendTLS.ClientCertificateRef

		resolver := gwutils.NewSecretReferenceResolver(&DummySecretReferenceConditionProvider{}, c.client)
		secret, err := resolver.SecretRefToSecret(c.gateway, *ref)

		if err != nil {
			log.Error().Msgf("Failed to resolve Secret: %s", err)
			return
		}

		if secret.Type != corev1.SecretTypeTLS {
			log.Warn().Msgf("BackendTLS Secret %s/%s is not of type %s, will be ignored for Gateway %s/%s",
				secret.Namespace, secret.Name, corev1.SecretTypeTLS,
				c.gateway.Namespace, c.gateway.Name)
			return
		}

		certName := fmt.Sprintf("gw-bk-tls-%s-%s.crt", c.gateway.Namespace, c.gateway.Name)
		keyName := fmt.Sprintf("gw-bk-tls-%s-%s.key", c.gateway.Namespace, c.gateway.Name)

		g2.Spec.BackendTLS = &fgwv2.GatewayBackendTLS{
			ClientCertificate: map[string]string{
				corev1.TLSCertKey:       certName,
				corev1.TLSPrivateKeyKey: keyName,
			},
		}

		c.secretFiles[certName] = string(secret.Data[corev1.TLSCertKey])
		c.secretFiles[keyName] = string(secret.Data[corev1.TLSPrivateKeyKey])
	}
}
