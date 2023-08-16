package configurator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fakeConfig "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/metricsstore"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
)

const (
	fsmNamespace      = "-test-fsm-namespace-"
	fsmMeshConfigName = "-test-fsm-mesh-config-"
)

func TestGetMeshConfig(t *testing.T) {
	a := assert.New(t)

	meshConfigClient := fakeConfig.NewSimpleClientset()
	stop := make(chan struct{})

	ic, err := informers.NewInformerCollection("fsm", stop, informers.WithConfigClient(meshConfigClient, fsmMeshConfigName, fsmNamespace))
	a.Nil(err)

	c := NewConfigurator(ic, fsmNamespace, fsmMeshConfigName, nil)

	// Returns empty MeshConfig if informer cache is empty
	a.Equal(configv1alpha3.MeshConfig{}, c.getMeshConfig())

	newObj := &configv1alpha3.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.flomesh.io",
			Kind:       "MeshConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: fsmNamespace,
			Name:      fsmMeshConfigName,
		},
	}
	err = c.informers.Add(informers.InformerKeyMeshConfig, newObj, t)
	a.Nil(err)
	a.Equal(*newObj, c.getMeshConfig())
}

func TestMetricsHandler(t *testing.T) {
	a := assert.New(t)

	c := &Client{
		meshConfigName: fsmMeshConfigName,
		informers:      &informers.InformerCollection{},
	}
	handlers := c.metricsHandler()
	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.FeatureFlagEnabled)

	// Adding the MeshConfig
	handlers.OnAdd(&configv1alpha3.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: fsmMeshConfigName,
		},
		Spec: configv1alpha3.MeshConfigSpec{
			FeatureFlags: configv1alpha3.FeatureFlags{
				EnableRetryPolicy: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 1` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n"))

	// Updating the MeshConfig
	handlers.OnUpdate(nil, &configv1alpha3.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: fsmMeshConfigName,
		},
		Spec: configv1alpha3.MeshConfigSpec{
			FeatureFlags: configv1alpha3.FeatureFlags{
				EnableSnapshotCacheMode: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 1` + "\n"))

	// Deleting the MeshConfig
	handlers.OnDelete(&configv1alpha3.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: fsmMeshConfigName,
		},
		Spec: configv1alpha3.MeshConfigSpec{
			FeatureFlags: configv1alpha3.FeatureFlags{
				EnableSnapshotCacheMode: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`fsm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n"))
}
