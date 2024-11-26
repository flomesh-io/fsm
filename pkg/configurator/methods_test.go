package configurator

import (
	"os"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	testclient "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"

	"github.com/flomesh-io/fsm/pkg/constants"
)

func TestGetMeshConfigCacheKey(t *testing.T) {
	c := Client{
		meshConfigName: "configName",
		fsmNamespace:   "namespaceName",
	}
	expected := "namespaceName/configName"
	actual := c.getMeshConfigCacheKey()
	tassert.Equal(t, expected, actual)
}

func TestCreateUpdateConfig(t *testing.T) {
	t.Run("MeshConfig doesn't exist", func(t *testing.T) {
		meshConfigClientSet := testclient.NewSimpleClientset()
		stop := make(chan struct{})

		ic, err := informers.NewInformerCollection("fsm", stop, informers.WithConfigClient(meshConfigClientSet, fsmMeshConfigName, fsmNamespace))
		tassert.Nil(t, err)

		cfg := NewConfigurator(ic, fsmNamespace, fsmMeshConfigName, nil)
		tassert.Equal(t, configv1alpha3.MeshConfig{}, cfg.getMeshConfig())
	})

	tests := []struct {
		name                  string
		initialMeshConfigData *configv1alpha3.MeshConfigSpec
		checkCreate           func(*tassert.Assertions, Configurator)
		updatedMeshConfigData *configv1alpha3.MeshConfigSpec
		checkUpdate           func(*tassert.Assertions, Configurator)
	}{
		{
			name: "default",

			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					EnablePrivilegedInitContainer: true,
					LogLevel:                      "error",
					MaxDataPlaneConnections:       0,
					ConfigResyncInterval:          "2m",
					SidecarImage:                  "flomesh/pipy:latest",
				},
				Traffic: configv1alpha3.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: false,
					EnableEgress:                      true,
				},
				Observability: configv1alpha3.ObservabilitySpec{
					FSMLogLevel: constants.DefaultFSMLogLevel,
					Tracing: configv1alpha3.TracingSpec{
						Enable: true,
					},
				},
				Certificate: configv1alpha3.CertificateSpec{
					ServiceCertValidityDuration: "24h",
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				expectedConfig := &configv1alpha3.MeshConfigSpec{
					Sidecar: configv1alpha3.SidecarSpec{
						EnablePrivilegedInitContainer: true,
						LogLevel:                      "error",
						MaxDataPlaneConnections:       0,
						ConfigResyncInterval:          "2m",
						SidecarImage:                  "flomesh/pipy:latest",
					},
					Traffic: configv1alpha3.TrafficSpec{
						EnablePermissiveTrafficPolicyMode: false,
						EnableEgress:                      true,
					},
					Observability: configv1alpha3.ObservabilitySpec{
						FSMLogLevel: constants.DefaultFSMLogLevel,
						Tracing: configv1alpha3.TracingSpec{
							Enable: true,
						},
					},
					Certificate: configv1alpha3.CertificateSpec{
						ServiceCertValidityDuration: "24h",
					},
				}
				expectedConfigJSON, err := marshalConfigToJSON(*expectedConfig)
				assert.Nil(err)

				configJSON, err := cfg.GetMeshConfigJSON()
				assert.Nil(err)
				assert.Equal(expectedConfigJSON, configJSON)
			},
		},
		{
			name: "IsPermissiveTrafficPolicyMode",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Traffic: configv1alpha3.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPermissiveTrafficPolicyMode())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Traffic: configv1alpha3.TrafficSpec{
					EnablePermissiveTrafficPolicyMode: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPermissiveTrafficPolicyMode())
			},
		},
		{
			name: "IsEgressEnabled",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Traffic: configv1alpha3.TrafficSpec{
					EnableEgress: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsEgressEnabled())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Traffic: configv1alpha3.TrafficSpec{
					EnableEgress: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsEgressEnabled())
			},
		},
		{
			name: "IsTracingEnabled",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Observability: configv1alpha3.ObservabilitySpec{
					Tracing: configv1alpha3.TracingSpec{
						Enable:   true,
						Address:  "myjaeger",
						Port:     12121,
						Endpoint: "/my/endpoint",
					},
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsTracingEnabled())
				assert.Equal("myjaeger", cfg.GetTracingHost())
				assert.Equal(uint32(12121), cfg.GetTracingPort())
				assert.Equal("/my/endpoint", cfg.GetTracingEndpoint())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Observability: configv1alpha3.ObservabilitySpec{
					Tracing: configv1alpha3.TracingSpec{
						Enable:   false,
						Address:  "myjaeger",
						Port:     12121,
						Endpoint: "/my/endpoint",
					},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsTracingEnabled())
			},
		},
		{
			name:                  "GetSidecarLogLevel",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("error", cfg.GetSidecarLogLevel())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					LogLevel: "info",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("info", cfg.GetSidecarLogLevel())
			},
		},
		{
			name:                  "GetSidecarImage",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("", cfg.GetSidecarImage())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					SidecarImage: "flomesh/pipy:latest",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("flomesh/pipy:latest", cfg.GetSidecarImage())
			},
		},
		{
			name:                  "GetInitContainerImage",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("", cfg.GetInitContainerImage())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					SidecarImage: "flomesh/pipy:latest",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				os.Setenv("FSM_DEFAULT_INIT_CONTAINER_IMAGE", "flomesh/init:v0.8.2")
				assert.Equal("flomesh/init:v0.8.2", cfg.GetInitContainerImage())
			},
		},
		{
			name: "GetServiceCertValidityDuration",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Certificate: configv1alpha3.CertificateSpec{
					ServiceCertValidityDuration: "24h",
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(24*time.Hour, cfg.GetServiceCertValidityPeriod())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Certificate: configv1alpha3.CertificateSpec{
					ServiceCertValidityDuration: "1h",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(1*time.Hour, cfg.GetServiceCertValidityPeriod())
			},
		},
		{
			name: "GetCertKeyBitSize",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Certificate: configv1alpha3.CertificateSpec{
					CertKeyBitSize: 4096,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(4096, cfg.GetCertKeyBitSize())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Certificate: configv1alpha3.CertificateSpec{
					CertKeyBitSize: -10,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(defaultCertKeyBitSize, cfg.GetCertKeyBitSize())
			},
		},
		{
			name: "IsPrivilegedInitContainer",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					EnablePrivilegedInitContainer: true,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.True(cfg.IsPrivilegedInitContainer())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					EnablePrivilegedInitContainer: false,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.False(cfg.IsPrivilegedInitContainer())
			},
		},
		{
			name:                  "GetResyncInterval",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					ConfigResyncInterval: "2m",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(time.Duration(2*time.Minute), interval)
			},
		},
		{
			name:                  "NegativeGetResyncInterval",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					ConfigResyncInterval: "Non-duration string",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				interval := cfg.GetConfigResyncInterval()
				assert.Equal(interval, time.Duration(0))
			},
		},
		{
			name:                  "GetMaxDataplaneConnections",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(0, cfg.GetMaxDataPlaneConnections())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					MaxDataPlaneConnections: 1000,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(1000, cfg.GetMaxDataPlaneConnections())
			},
		},
		{
			name:                  "GetProxyResources",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				res := cfg.GetProxyResources()
				assert.Equal(0, len(res.Limits))
				assert.Equal(0, len(res.Requests))
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Sidecar: configv1alpha3.SidecarSpec{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1"),
							v1.ResourceMemory: resource.MustParse("256M"),
						},
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("2"),
							v1.ResourceMemory: resource.MustParse("512M"),
						},
					},
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				res := cfg.GetProxyResources()
				assert.Equal(resource.MustParse("1"), res.Requests[v1.ResourceCPU])
				assert.Equal(resource.MustParse("256M"), res.Requests[v1.ResourceMemory])
				assert.Equal(resource.MustParse("2"), res.Limits[v1.ResourceCPU])
				assert.Equal(resource.MustParse("512M"), res.Limits[v1.ResourceMemory])
			},
		},
		{
			name:                  "IsEgressPolicyEnabled",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(false, cfg.GetFeatureFlags().EnableEgressPolicy)
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				FeatureFlags: configv1alpha3.FeatureFlags{
					EnableEgressPolicy: true,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(true, cfg.GetFeatureFlags().EnableEgressPolicy)
			},
		},
		{
			name:                  "IsAsyncProxyServiceMappingEnabled",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(false, cfg.GetFeatureFlags().EnableAsyncProxyServiceMapping)
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				FeatureFlags: configv1alpha3.FeatureFlags{
					EnableAsyncProxyServiceMapping: true,
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(true, cfg.GetFeatureFlags().EnableAsyncProxyServiceMapping)
			},
		},
		{
			name: "FSMLogLevel",
			initialMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Observability: configv1alpha3.ObservabilitySpec{
					FSMLogLevel: constants.DefaultFSMLogLevel,
				},
			},
			checkCreate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal(constants.DefaultFSMLogLevel, cfg.GetFSMLogLevel())
			},
			updatedMeshConfigData: &configv1alpha3.MeshConfigSpec{
				Observability: configv1alpha3.ObservabilitySpec{
					FSMLogLevel: "warn",
				},
			},
			checkUpdate: func(assert *tassert.Assertions, cfg Configurator) {
				assert.Equal("warn", cfg.GetFSMLogLevel())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			meshConfigClientSet := testclient.NewSimpleClientset()

			// Create configurator
			stop := make(chan struct{})
			defer close(stop)

			ic, err := informers.NewInformerCollection("fsm", stop, informers.WithConfigClient(meshConfigClientSet, fsmMeshConfigName, fsmNamespace))
			assert.Nil(err)

			cfg := NewConfigurator(ic, fsmNamespace, fsmMeshConfigName, nil)

			meshConfig := configv1alpha3.MeshConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: fsmNamespace,
					Name:      fsmMeshConfigName,
				},
				Spec: *test.initialMeshConfigData,
			}

			err = cfg.informers.Add(informers.InformerKeyMeshConfig, &meshConfig, t)
			assert.Nil(err)

			test.checkCreate(assert, cfg)

			if test.checkUpdate == nil {
				return
			}

			meshConfig.Spec = *test.updatedMeshConfigData
			err = cfg.informers.Update(informers.InformerKeyMeshConfig, &meshConfig, t)

			assert.Nil(err)

			test.checkUpdate(assert, cfg)
		})
	}
}
