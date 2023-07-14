// Package fake moves fakes to their own sub-package
package fake

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/certificate/pem"
	"github.com/flomesh-io/fsm/pkg/certificate/providers/tresor"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

const (
	rootCertOrganization = "Flomesh Service Mesh Tresor"
)

type fakeMRCClient struct{}

func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha3.MeshRootCertificate) (certificate.Issuer, pem.RootCertificate, error) {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	ca, err := tresor.NewCA("Fake Tresor CN", 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		return nil, nil, err
	}
	issuer, err := tresor.New(ca, rootCertOrganization, 2048)
	return issuer, pem.RootCertificate("rootCA"), err
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) List() ([]*v1alpha3.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha3.MeshRootCertificate{{Spec: v1alpha3.MeshRootCertificateSpec{TrustDomain: "fake.example.com"}}}, nil
}

func (c *fakeMRCClient) Watch(ctx context.Context) (<-chan certificate.MRCEvent, error) {
	ch := make(chan certificate.MRCEvent)
	go func() {
		ch <- certificate.MRCEvent{
			Type: certificate.MRCEventAdded,
			MRC: &v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						Tresor: &v1alpha3.TresorProviderSpec{
							CA: v1alpha3.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "fsm-ca-bundle",
									Namespace: "fsm-system",
								},
							},
						},
					},
					TrustDomain: "cluster.local",
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			},
		}
		close(ch)
	}()

	return ch, nil
}

// NewFake constructs a fake certificate client using a certificate
func NewFake(msgBroker *messaging.Broker, checkInterval time.Duration) *certificate.Manager {
	getValidityDuration := func() time.Duration { return 1 * time.Hour }
	return NewFakeWithValidityDuration(getValidityDuration, msgBroker, checkInterval)
}

// NewFakeWithValidityDuration constructs a fake certificate manager with specified cert validity duration
func NewFakeWithValidityDuration(getCertValidityDuration func() time.Duration, msgBroker *messaging.Broker, checkInterval time.Duration) *certificate.Manager {
	tresorCertManager, err := certificate.NewManager(context.Background(), &fakeMRCClient{}, getCertValidityDuration, getCertValidityDuration, msgBroker, checkInterval)
	if err != nil {
		log.Error().Err(err).Msg("error encountered creating fake cert manager")
		return nil
	}
	return tresorCertManager
}

// NewFakeCertificate is a helper creating Certificates for unit tests.
func NewFakeCertificate() *certificate.Certificate {
	return &certificate.Certificate{
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		IssuingCA:    pem.RootCertificate("xx"),
		TrustedCAs:   pem.RootCertificate("xx"),
		Expiration:   time.Now(),
		CommonName:   "foo.bar.co.uk",
		SerialNumber: "-the-certificate-serial-number-",
	}
}
