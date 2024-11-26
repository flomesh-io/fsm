package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	fakeConfigClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/certificate/pem"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

func TestGetCertificateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()
	type testCase struct {
		name        string
		expectError bool

		// params
		kubeClient        kubernetes.Interface
		restConfig        *rest.Config
		cfg               configurator.Configurator
		providerNamespace string
		options           Options
		msgBroker         *messaging.Broker
	}
	testCases := []testCase{
		{
			name:              "tresor as the certificate manager",
			options:           TresorOptions{SecretName: "fsm-ca-bundle"},
			providerNamespace: "fsm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
		},
		{
			name:              "tresor with no secret",
			options:           TresorOptions{},
			providerNamespace: "fsm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			expectError:       true,
		},
		{
			name:              "certManager as the certificate manager",
			kubeClient:        fake.NewSimpleClientset(),
			restConfig:        &rest.Config{},
			cfg:               mockConfigurator,
			providerNamespace: "fsm-system",
			options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
		},
		{
			name:        "Fail to validate Config",
			options:     VaultOptions{},
			expectError: true,
		},
		{
			name: "Valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultToken:    "vault-token",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "http",
			},
			cfg: mockConfigurator,
		},
		{
			name: "Valid Vault protocol using vault secret",
			options: VaultOptions{
				VaultHost:                 "vault.default.svc.cluster.local",
				VaultRole:                 "role",
				VaultPort:                 8200,
				VaultProtocol:             "http",
				VaultTokenSecretName:      "secret",
				VaultTokenSecretKey:       "token",
				VaultTokenSecretNamespace: "fsm-system",
			},
			kubeClient: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "fsm-system",
				},
				Data: map[string][]byte{
					"token": []byte("secret"),
				},
			}),
			cfg: mockConfigurator,
		},
		{
			name: "Not a valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultToken:    "vault-token",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "hi",
			},
			expectError: true,
		},
		{
			name: "Invalid cert manager options",
			options: CertManagerOptions{
				IssuerKind:  "test-kind",
				IssuerGroup: "cert-manager.io",
			},
			cfg:         mockConfigurator,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			oldCA := getCA
			getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
				return pem.RootCertificate("id2"), nil
			}

			defer func() {
				getCA = oldCA
			}()

			manager, err := NewCertificateManager(context.Background(), tc.kubeClient, tc.restConfig, tc.cfg, tc.providerNamespace, tc.options, tc.msgBroker, 1*time.Hour, "cluster.local")
			if tc.expectError {
				assert.Empty(manager)
				assert.Error(err)
			} else {
				assert.NotEmpty(manager)
				assert.NoError(err)
			}

			if opt, ok := tc.options.(TresorOptions); ok && !tc.expectError {
				_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Get(context.TODO(), opt.SecretName, metav1.GetOptions{})
				assert.NoError(err)
			}
		})
	}
}

func TestGetCertificateManagerFromMRC(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()

	type testCase struct {
		name        string
		expectError bool

		// params
		kubeClient        kubernetes.Interface
		configClient      configClientset.Interface
		restConfig        *rest.Config
		cfg               configurator.Configurator
		providerNamespace string
		options           Options
		msgBroker         *messaging.Broker
	}
	testCases := []testCase{
		{
			name:              "tresor as the certificate manager",
			options:           TresorOptions{SecretName: "fsm-ca-bundle"},
			providerNamespace: "fsm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
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
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
		},
		{
			name:              "tresor with no secret",
			options:           TresorOptions{},
			providerNamespace: "fsm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			expectError:       true,
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						Tresor: &v1alpha3.TresorProviderSpec{
							CA: v1alpha3.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "",
									Namespace: "",
								},
							},
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
		},
		{
			name:              "certManager as the certificate manager",
			kubeClient:        fake.NewSimpleClientset(),
			restConfig:        &rest.Config{},
			cfg:               mockConfigurator,
			providerNamespace: "fsm-system",
			options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						CertManager: &v1alpha3.CertManagerProviderSpec{
							IssuerName:  "test-name",
							IssuerKind:  "ClusterIssuer",
							IssuerGroup: "cert-manager.io",
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
		},
		{
			name:        "Fail to validate Config",
			options:     VaultOptions{},
			kubeClient:  fake.NewSimpleClientset(),
			expectError: true,
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						Vault: &v1alpha3.VaultProviderSpec{
							Host:     "",
							Port:     0,
							Role:     "",
							Protocol: "",
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
		},
		{
			name: "Valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "http",
				VaultToken:    "vault-token",
			},
			cfg:        mockConfigurator,
			kubeClient: fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						Vault: &v1alpha3.VaultProviderSpec{
							Host:     "vault.default.svs.cluster.local",
							Port:     8200,
							Role:     "role",
							Protocol: "http",
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
		},
		{
			name: "Valid Vault protocol using vault secret",
			options: VaultOptions{
				VaultHost:                 "vault.default.svc.cluster.local",
				VaultRole:                 "role",
				VaultPort:                 8200,
				VaultProtocol:             "http",
				VaultTokenSecretName:      "secret",
				VaultTokenSecretKey:       "token",
				VaultTokenSecretNamespace: "fsm-system",
			},
			kubeClient: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "fsm-system",
				},
				Data: map[string][]byte{
					"token": []byte("secret"),
				},
			}),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						Vault: &v1alpha3.VaultProviderSpec{
							Host:     "vault.default.svc.cluster.local",
							Role:     "role",
							Port:     8200,
							Protocol: "http",
							Token: v1alpha3.VaultTokenSpec{
								SecretKeyRef: v1alpha3.SecretKeyReferenceSpec{
									Name:      "secret",
									Namespace: "fsm-system",
									Key:       "token",
								},
							},
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			cfg: mockConfigurator,
		},
		{
			name: "Not a valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultToken:    "vault-token",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "hi",
			},
			expectError: true,
			cfg:         mockConfigurator,
			kubeClient:  fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						Vault: &v1alpha3.VaultProviderSpec{
							Host:     "vault.default.svs.cluster.local",
							Port:     8200,
							Role:     "role",
							Protocol: "hi",
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
		},
		{
			name: "Invalid cert manager options",
			options: CertManagerOptions{
				IssuerKind:  "test-kind",
				IssuerGroup: "cert-manager.io",
			},
			cfg:        mockConfigurator,
			kubeClient: fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha3.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fsm-mesh-root-certificate",
					Namespace: "fsm-system",
				},
				Spec: v1alpha3.MeshRootCertificateSpec{
					Provider: v1alpha3.ProviderSpec{
						CertManager: &v1alpha3.CertManagerProviderSpec{
							IssuerName:  "",
							IssuerKind:  "test-kind",
							IssuerGroup: "cert-manager.io",
						},
					},
				},
				Status: v1alpha3.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			oldCA := getCA
			getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
				return pem.RootCertificate("id2"), nil
			}

			defer func() {
				getCA = oldCA
			}()

			tc.kubeClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
				GitVersion: "v1.21.0",
			}
			ic, err := informers.NewInformerCollection("fsm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "fsm-system"))
			assert.NoError(err)
			assert.NotNil(ic)

			manager, err := NewCertificateManagerFromMRC(context.Background(), tc.kubeClient, tc.restConfig, tc.cfg, tc.providerNamespace, tc.options, tc.msgBroker, ic, 1*time.Hour)
			if tc.expectError {
				assert.Empty(manager)
				assert.Error(err)
			} else {
				assert.NotEmpty(manager)
				assert.NoError(err)
			}

			if opt, ok := tc.options.(TresorOptions); ok && !tc.expectError {
				_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Get(context.TODO(), opt.SecretName, metav1.GetOptions{})
				assert.NoError(err)
			}
		})
	}
}

func TestGetHashiVaultFSMToken(t *testing.T) {
	validVaultTokenSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fsm-system",
			Name:      "fsm-vault-token",
		},
		Data: map[string][]byte{
			"token": []byte("token"),
		},
	}

	invalidVaultTokenSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "fsm-system",
			Name:      "fsm-vault-token",
		},
		Data: map[string][]byte{
			"noop": []byte("noop"),
		},
	}

	testCases := []struct {
		name         string
		secretKeyRef *v1alpha3.SecretKeyReferenceSpec
		kubeClient   kubernetes.Interface
		expectError  bool
	}{
		{
			name: "No Vault token secret",
			secretKeyRef: &v1alpha3.SecretKeyReferenceSpec{
				Name:      "fsm-vault-token",
				Namespace: "fsm-system",
				Key:       "token",
			},
			kubeClient:  fake.NewSimpleClientset(),
			expectError: true,
		},
		{
			name: "Invalid Vault token secret",
			secretKeyRef: &v1alpha3.SecretKeyReferenceSpec{
				Name:      "fsm-vault-token",
				Namespace: "fsm-system",
				Key:       "token",
			},
			kubeClient:  fake.NewSimpleClientset([]runtime.Object{invalidVaultTokenSecret}...),
			expectError: true,
		},
		{
			name: "Valid Vault token secret",
			secretKeyRef: &v1alpha3.SecretKeyReferenceSpec{
				Name:      "fsm-vault-token",
				Namespace: "fsm-system",
				Key:       "token",
			},
			kubeClient:  fake.NewSimpleClientset([]runtime.Object{validVaultTokenSecret}...),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			token, err := getHashiVaultFSMToken(tc.secretKeyRef, tc.kubeClient)
			if tc.expectError {
				assert.Empty(token)
				assert.Error(err)
			} else {
				assert.NotEmpty(token)
				assert.NoError(err)
			}
		})
	}
}
