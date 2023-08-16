package certificate

import (
	"context"
	"fmt"
	"time"

	"github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/certificate/pem"
	"github.com/flomesh-io/fsm/pkg/constants"
)

var (
	validity = time.Hour
)

type fakeMRCClient struct{}

func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha3.MeshRootCertificate) (Issuer, pem.RootCertificate, error) {
	return &fakeIssuer{}, pem.RootCertificate("rootCA"), nil
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) List() ([]*v1alpha3.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha3.MeshRootCertificate{{
		Spec: v1alpha3.MeshRootCertificateSpec{
			TrustDomain: "fake.domain.com",
		},
	}}, nil
}

func (c *fakeMRCClient) Watch(ctx context.Context) (<-chan MRCEvent, error) {
	ch := make(chan MRCEvent)
	go func() {
		ch <- MRCEvent{
			Type: MRCEventAdded,
			MRC: &v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					TrustDomain: "fake.domain.com",
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

type fakeIssuer struct {
	err bool
	id  string
}

// IssueCertificate is a testing helper to satisfy the certificate client interface
func (i *fakeIssuer) IssueCertificate(cn CommonName, saNames []string, validityPeriod time.Duration) (*Certificate, error) {
	if i.err {
		return nil, fmt.Errorf("%s failed", i.id)
	}
	return &Certificate{
		CommonName: cn,
		SANames:    saNames,
		Expiration: time.Now().Add(validityPeriod),
		// simply used to distinguish the private/public key from other issuers
		IssuingCA:  pem.RootCertificate(i.id),
		TrustedCAs: pem.RootCertificate(i.id),
		PrivateKey: pem.PrivateKey(i.id),
	}, nil
}

// FakeCertManager is a testing helper that returns a *certificate.Manager
func FakeCertManager() (*Manager, error) {
	getCertValidityDuration := func() time.Duration { return validity }
	cm, err := NewManager(
		context.Background(),
		&fakeMRCClient{},
		getCertValidityDuration,
		getCertValidityDuration,
		nil,
		1*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}
	return cm, nil
}
