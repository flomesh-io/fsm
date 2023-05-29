package certmanager

import (
	"crypto/rand"
	"crypto/x509"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	tassert "github.com/stretchr/testify/assert"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmfakeclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/tests"
)

var (
	mockCtrl         = gomock.NewController(ginkgo.GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
	crNotReady       = &cmapi.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fsm-123",
			Namespace: "fsm-system",
		},
	}
)

const (
	validity = 1 * time.Hour
	keySize  = 2048
)

func TestCertificateFromCertificateRequest(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := cmfakeclient.NewSimpleClientset()

	rootCertPEM, err := tests.GetPEMCert()
	assert.Nil(err)

	rootCert, err := certificate.DecodePEMCertificate(rootCertPEM)
	assert.Nil(err)

	rootKeyPEM, err := tests.GetPEMPrivateKey()
	assert.Nil(err)

	rootKey, err := certificate.DecodePEMPrivateKey(rootKeyPEM)
	assert.Nil(err)

	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()

	cm, err := New(
		fakeClient,
		"fsm-system",
		cmmeta.ObjectReference{Name: "fsm-ca"},
		keySize,
	)
	assert.Nil(err)

	signedCertDER, err := x509.CreateCertificate(rand.Reader, rootCert, rootCert, rootKey.Public(), rootKey)
	assert.Nil(err)

	signedCertPEM, err := certificate.EncodeCertDERtoPEM(signedCertDER)
	assert.Nil(err)

	crReady := crNotReady.DeepCopy()
	crReady.Status = cmapi.CertificateRequestStatus{
		Certificate: signedCertPEM,
		CA:          signedCertPEM,
		Conditions: []cmapi.CertificateRequestCondition{
			{
				Type:   cmapi.CertificateRequestConditionReady,
				Status: cmmeta.ConditionTrue,
			},
		},
	}
	emptyArr := []byte{}
	testCases := []struct {
		name              string
		cr                cmapi.CertificateRequest
		expectedCertIsNil bool
		expectedError     error
	}{
		{
			name:              "Could not decode PEM Cert",
			cr:                *crNotReady,
			expectedCertIsNil: true,
			expectedError:     certificate.ErrNoCertificateInPEM,
		},
		{
			name:              "default",
			cr:                *crReady,
			expectedCertIsNil: false,
			expectedError:     nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			cert, err := cm.certificateFromCertificateRequest(&tc.cr, emptyArr)

			assert.Equal(tc.expectedCertIsNil, cert == nil)
			assert.Equal(tc.expectedError, err)
		})
	}
	// Tests if cmapi.CertificateRequest is nil
	cert, err := cm.certificateFromCertificateRequest(nil, emptyArr)
	assert.Nil(cert)
	assert.Nil(err)
}

func TestNew(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := cmfakeclient.NewSimpleClientset()
	_, err := New(
		fakeClient,
		"fsm-system",
		cmmeta.ObjectReference{Name: "fsm-ca"},
		0,
	)

	assert.Error(err, "expected error from key size of zero")
	_, err = New(
		fakeClient,
		"fsm-system",
		cmmeta.ObjectReference{Name: "fsm-ca"},
		keySize,
	)
	assert.NoError(err, "expected no error from key size of zero, got: %s", err)
}
