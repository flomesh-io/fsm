package main

import (
	"context"
	"os"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakeKube "k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/constants"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	fakeConfig "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/fake"
)

var testNamespace = "test-namespace"

var testMeshConfig = &configv1alpha3.MeshConfig{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      meshConfigName,
	},
	Spec: configv1alpha3.MeshConfigSpec{},
}

var testMeshConfigWithLastAppliedAnnotation = &configv1alpha3.MeshConfig{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      meshConfigName,
		Annotations: map[string]string{
			"kubectl.kubernetes.io/last-applied-configuration": `{"metadata":{"name":"fsm-mesh-config","namespace":"test-namespace","creationTimestamp":null},"spec":{}}`,
		},
	},
	Spec: configv1alpha3.MeshConfigSpec{},
}

var testPresetMeshConfigMap = &corev1.ConfigMap{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      presetMeshConfigName,
		Namespace: testNamespace,
	},
	Data: map[string]string{
		presetMeshConfigJSONKey: `{
"sidecar": {
	"enablePrivilegedInitContainer": false,
	"logLevel": "error",
	"maxDataPlaneConnections": 0,
	"initContainerImage": "flomesh/init:latest-main",
	"configResyncInterval": "2s"
},
"traffic": {
	"enableEgress": true,
	"useHTTPSIngress": false,
	"enablePermissiveTrafficPolicyMode": true,
	"outboundPortExclusionList": [],
	"inboundPortExclusionList": [],
	"outboundIPRangeExclusionList": []
},
"observability": {
	"fsmLogLevel": "trace",
	"tracing": {
	 "enable": false
	}
},
"certificate": {
	"serviceCertValidityDuration": "23h"
},
"featureFlags": {
	"enableEgressPolicy": true,
	"enableAsyncProxyServiceMapping": false,
	"enableIngressBackendPolicy": true,
	"enableAccessControlPolicy": true,
	"enableAccessCertPolicy": true,
	"enableSidecarActiveHealthChecks": true,
	"enableSnapshotCacheMode": true,
	"enableRetryPolicy": false
	}
}`,
	},
}

var testMeshRootCertificate = &configv1alpha3.MeshRootCertificate{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      meshRootCertificateName,
	},
	Spec: configv1alpha3.MeshRootCertificateSpec{},
	Status: configv1alpha3.MeshRootCertificateStatus{
		State: constants.MRCStateActive,
	},
}

var testPresetMeshRootCertificate = &corev1.ConfigMap{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      presetMeshRootCertificateName,
		Namespace: testNamespace,
	},
	Data: map[string]string{
		presetMeshRootCertificateJSONKey: `{
"provider": {
	"tresor": {
	 "ca": {
	  "secretRef": {
		"name": "fsm-ca-bundle",
		"namespace": "test-namespace"
	  }
	 }
	}
	}
}`,
	},
}

func TestBuildDefaultMeshConfig(t *testing.T) {
	assert := tassert.New(t)

	meshConfig, err := buildDefaultMeshConfig(testPresetMeshConfigMap)
	assert.NoError(err)
	assert.Contains(meshConfig.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	assert.Equal(meshConfig.Name, meshConfigName)
	assert.Equal(meshConfig.Spec.Sidecar.LogLevel, "error")
	assert.Equal(meshConfig.Spec.Sidecar.ConfigResyncInterval, "2s")
	assert.False(meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer)
	assert.True(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode)
	assert.True(meshConfig.Spec.Traffic.EnableEgress)
	assert.Equal(meshConfig.Spec.Certificate.ServiceCertValidityDuration, "23h")
	assert.True(meshConfig.Spec.FeatureFlags.EnableIngressBackendPolicy)
	assert.True(meshConfig.Spec.FeatureFlags.EnableAccessControlPolicy)
	assert.True(meshConfig.Spec.FeatureFlags.EnableAccessCertPolicy)
	assert.True(meshConfig.Spec.FeatureFlags.EnableSidecarActiveHealthChecks)
	assert.False(meshConfig.Spec.FeatureFlags.EnableRetryPolicy)
}

func TestBuildMeshRootCertificate(t *testing.T) {
	assert := tassert.New(t)

	meshRootCertificate, err := buildMeshRootCertificate(testPresetMeshRootCertificate)
	assert.Contains(meshRootCertificate.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	assert.NoError(err)
	assert.Equal(meshRootCertificate.Name, meshRootCertificateName)
	assert.Equal(meshRootCertificate.Spec.Provider.Tresor.CA.SecretRef.Name, "fsm-ca-bundle")
	assert.Equal(meshRootCertificate.Spec.Provider.Tresor.CA.SecretRef.Namespace, testNamespace)
	assert.Nil(meshRootCertificate.Spec.Provider.Vault)
	assert.Nil(meshRootCertificate.Spec.Provider.CertManager)
}

func TestValidateCLIParams(t *testing.T) {
	assert := tassert.New(t)

	// save original global values
	prevFsmNamespace := fsmNamespace

	tests := []struct {
		name   string
		setup  func()
		verify func(error)
	}{
		{
			name: "fsm-namespace is empty",
			setup: func() {
				fsmNamespace = ""
			},
			verify: func(err error) {
				assert.NotNil(err)
				assert.Contains(err.Error(), "--fsm-namespace")
			},
		},
		{
			name: "fsm-namespace is valid",
			setup: func() {
				fsmNamespace = "valid-ns"
			},
			verify: func(err error) {
				assert.Nil(err)
			},
		},
	}

	for _, tc := range tests {
		tc.setup()
		err := validateCLIParams()
		tc.verify(err)
	}

	// restore original global values
	fsmNamespace = prevFsmNamespace
}

func TestCreateDefaultMeshConfig(t *testing.T) {
	tests := []struct {
		name                    string
		namespace               string
		kubeClient              kubernetes.Interface
		configClient            configClientset.Interface
		expectDefaultMeshConfig bool
		expectErr               bool
	}{
		{
			name:                    "successfully create default meshconfig from preset configmap",
			namespace:               testNamespace,
			kubeClient:              fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshConfigMap}...),
			configClient:            fakeConfig.NewSimpleClientset(),
			expectDefaultMeshConfig: true,
			expectErr:               false,
		},
		{
			name:                    "preset configmap does not exist",
			namespace:               testNamespace,
			kubeClient:              fakeKube.NewSimpleClientset(),
			configClient:            fakeConfig.NewSimpleClientset(),
			expectDefaultMeshConfig: false,
			expectErr:               true,
		},
		{
			name:                    "default MeshConfig already exists",
			namespace:               testNamespace,
			kubeClient:              fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshConfigMap}...),
			configClient:            fakeConfig.NewSimpleClientset([]runtime.Object{testMeshConfig}...),
			expectDefaultMeshConfig: true,
			expectErr:               false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			b := bootstrap{
				kubeClient:   tc.kubeClient,
				configClient: tc.configClient,
				namespace:    tc.namespace,
			}

			err := b.createDefaultMeshConfig()
			assert.Equal(tc.expectErr, err != nil)

			_, err = b.configClient.ConfigV1alpha3().MeshConfigs(b.namespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})
			assert.Equal(tc.expectDefaultMeshConfig, err == nil)
		})
	}
}

func TestEnsureMeshConfig(t *testing.T) {
	tests := []struct {
		name         string
		namespace    string
		kubeClient   kubernetes.Interface
		configClient configClientset.Interface
		expectErr    bool
	}{
		{
			name:         "MeshConfig found with no last-applied annotation",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset(),
			configClient: fakeConfig.NewSimpleClientset([]runtime.Object{testMeshConfig}...),
			expectErr:    false,
		},
		{
			name:         "MeshConfig found with last-applied annotation",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset(),
			configClient: fakeConfig.NewSimpleClientset([]runtime.Object{testMeshConfigWithLastAppliedAnnotation}...),
			expectErr:    false,
		},
		{
			name:         "MeshConfig not found but successfully created",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshConfigMap}...),
			configClient: fakeConfig.NewSimpleClientset(),
			expectErr:    false,
		},
		{
			name:         "MeshConfig not found and error creating it",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset(),
			configClient: fakeConfig.NewSimpleClientset(),
			expectErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			b := bootstrap{
				kubeClient:   tc.kubeClient,
				configClient: tc.configClient,
				namespace:    tc.namespace,
			}

			err := b.ensureMeshConfig()
			assert.Equal(tc.expectErr, err != nil)
			if !tc.expectErr {
				config, err := b.configClient.ConfigV1alpha3().MeshConfigs(b.namespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})
				assert.Nil(err)
				assert.Contains(config.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
			}
		})
	}
}

func TestCreateMeshRootCertificate(t *testing.T) {
	tests := []struct {
		name                             string
		namespace                        string
		kubeClient                       kubernetes.Interface
		configClient                     configClientset.Interface
		expectDefaultMeshRootCertificate bool
		expectErr                        bool
	}{
		{
			name:                             "successfully create default MeshRootCertificate from preset configmap",
			namespace:                        testNamespace,
			kubeClient:                       fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshRootCertificate}...),
			configClient:                     fakeConfig.NewSimpleClientset(),
			expectDefaultMeshRootCertificate: true,
			expectErr:                        false,
		},
		{
			name:                             "preset configmap does not exist",
			namespace:                        testNamespace,
			kubeClient:                       fakeKube.NewSimpleClientset(),
			configClient:                     fakeConfig.NewSimpleClientset(),
			expectDefaultMeshRootCertificate: false,
			expectErr:                        true,
		},
		{
			name:                             "MeshRootCertificate already exists",
			namespace:                        testNamespace,
			kubeClient:                       fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshRootCertificate}...),
			configClient:                     fakeConfig.NewSimpleClientset([]runtime.Object{testMeshRootCertificate}...),
			expectDefaultMeshRootCertificate: true,
			expectErr:                        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			b := bootstrap{
				kubeClient:   tc.kubeClient,
				configClient: tc.configClient,
				namespace:    tc.namespace,
			}

			err := b.createMeshRootCertificate()
			if !tc.expectErr {
				assert.NoError(err)
			} else {
				assert.Error(err)
			}

			mrc, err := b.configClient.ConfigV1alpha3().MeshRootCertificates(b.namespace).Get(context.TODO(), meshRootCertificateName, metav1.GetOptions{})
			if tc.expectDefaultMeshRootCertificate {
				assert.NoError(err)
				assert.Equal(constants.MRCStateActive, mrc.Status.State)
			} else {
				assert.Error(err)
			}
		})
	}
}

func TestEnsureMeshRootCertificate(t *testing.T) {
	tests := []struct {
		name         string
		namespace    string
		kubeClient   kubernetes.Interface
		configClient configClientset.Interface
		expectErr    bool
	}{
		{
			name:         "MeshRootCertificate found",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset(),
			configClient: fakeConfig.NewSimpleClientset([]runtime.Object{testMeshRootCertificate}...),
			expectErr:    false,
		},
		{
			name:         "MeshRootCertificate not found but successfully created",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset([]runtime.Object{testPresetMeshRootCertificate}...),
			configClient: fakeConfig.NewSimpleClientset(),
			expectErr:    false,
		},
		{
			name:         "MeshRootCertificate not found and error creating it",
			namespace:    testNamespace,
			kubeClient:   fakeKube.NewSimpleClientset(),
			configClient: fakeConfig.NewSimpleClientset(),
			expectErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			b := bootstrap{
				kubeClient:   tc.kubeClient,
				configClient: tc.configClient,
				namespace:    tc.namespace,
			}

			err := b.ensureMeshRootCertificate()
			assert.Equal(tc.expectErr, err != nil)

			_, err = b.configClient.ConfigV1alpha3().MeshRootCertificates(b.namespace).Get(context.TODO(), meshRootCertificateName, metav1.GetOptions{})
			assert.Equal(tc.expectErr, err != nil)
		})
	}
}

func TestGetBootstrapPod(t *testing.T) {
	assert := tassert.New(t)
	testPodName := "test-pod-name"
	testNamespace := "test-namespace"
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
		},
	}

	tests := []struct {
		name                string
		namespace           string
		bootstrapPodNameEnv string
		kubeClient          kubernetes.Interface
		expectErr           bool
	}{
		{
			name:                "BOOTSTRAP_POD_NAME env var not set",
			namespace:           testNamespace,
			bootstrapPodNameEnv: "",
			kubeClient:          fakeKube.NewSimpleClientset(),
			expectErr:           true,
		},
		{
			name:                "BOOTSTRAP_POD_NAME env var set correctly and pod exists",
			namespace:           testNamespace,
			bootstrapPodNameEnv: testPodName,
			kubeClient:          fakeKube.NewSimpleClientset([]runtime.Object{testPod}...),
			expectErr:           false,
		},
		{
			name:                "BOOTSTRAP_POD_NAME env var set incorrectly",
			namespace:           testNamespace,
			bootstrapPodNameEnv: "something-random",
			kubeClient:          fakeKube.NewSimpleClientset([]runtime.Object{testPod}...),
			expectErr:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := bootstrap{
				namespace:  tc.namespace,
				kubeClient: tc.kubeClient,
			}
			defer func() {
				err := resetEnv("BOOTSTRAP_POD_NAME", os.Getenv("BOOTSTRAP_POD_NAME"))
				assert.Nil(err)
			}()

			err := os.Setenv("BOOTSTRAP_POD_NAME", tc.bootstrapPodNameEnv)
			assert.Nil(err)

			_, err = b.getBootstrapPod()
			assert.Equal(tc.expectErr, err != nil)
		})
	}
}

func resetEnv(key, val string) error {
	return os.Setenv(key, val)
}
