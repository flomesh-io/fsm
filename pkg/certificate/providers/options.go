package providers

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
)

// Validate validates the options for Tresor certificate provider
func (options TresorOptions) Validate() error {
	if options.SecretName == "" {
		return errors.New("tresor CA bundle secret name must be set")
	}
	return nil
}

// AsProviderSpec returns the provider spec generated from the tresor options
func (options TresorOptions) AsProviderSpec() v1alpha3.ProviderSpec {
	return v1alpha3.ProviderSpec{
		Tresor: &v1alpha3.TresorProviderSpec{
			CA: v1alpha3.TresorCASpec{
				SecretRef: corev1.SecretReference{
					Name: options.SecretName,
				},
			},
		},
	}
}

// Validate validates the options for Hashi Vault certificate provider
func (options VaultOptions) Validate() error {
	if options.VaultHost == "" {
		return errors.New("VaultHost not specified in Hashi Vault options")
	}

	if options.VaultToken == "" && (options.VaultTokenSecretKey == "" || options.VaultTokenSecretName == "") {
		return errors.New("VaultTokenSecretKey and VaultTokenSecretName must both specified if VaultToken is not specified in Hashi Vault options")
	}

	if options.VaultRole == "" {
		return errors.New("VaultRole not specified in Hashi Vault options")
	}

	if _, ok := map[string]interface{}{"http": nil, "https": nil}[options.VaultProtocol]; !ok {
		return fmt.Errorf("VaultProtocol in Hashi Vault options must be one of [http, https], got %s", options.VaultProtocol)
	}

	return nil
}

// AsProviderSpec returns the provider spec generated from the vault options
func (options VaultOptions) AsProviderSpec() v1alpha3.ProviderSpec {
	return v1alpha3.ProviderSpec{
		Vault: &v1alpha3.VaultProviderSpec{
			Protocol: options.VaultProtocol,
			Host:     options.VaultHost,
			Token: v1alpha3.VaultTokenSpec{
				SecretKeyRef: v1alpha3.SecretKeyReferenceSpec{
					Name:      options.VaultTokenSecretName,
					Namespace: options.VaultTokenSecretNamespace,
					Key:       options.VaultTokenSecretKey,
				},
			},
			Role: options.VaultRole,
			Port: options.VaultPort,
		},
	}
}

// Validate validates the options for cert-manager.io certificate provider
func (options CertManagerOptions) Validate() error {
	if options.IssuerName == "" {
		return errors.New("IssuerName not specified in cert-manager.io options")
	}

	if options.IssuerKind == "" {
		return errors.New("IssuerKind not specified in cert-manager.io options")
	}

	if options.IssuerGroup == "" {
		return errors.New("IssuerGroup not specified in cert-manager.io options")
	}

	return nil
}

// AsProviderSpec returns the provider spec generated from the CertManager options
func (options CertManagerOptions) AsProviderSpec() v1alpha3.ProviderSpec {
	return v1alpha3.ProviderSpec{
		CertManager: &v1alpha3.CertManagerProviderSpec{
			IssuerName:  options.IssuerName,
			IssuerKind:  options.IssuerKind,
			IssuerGroup: options.IssuerGroup,
		},
	}
}
