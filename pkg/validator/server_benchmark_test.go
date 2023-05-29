package validator

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	k8sClientFake "k8s.io/client-go/kubernetes/fake"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiAccessClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	smiSpecClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	smiSplitClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"

	policyv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/policy/v1alpha1"
	configFake "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/fake"
	pluginFake "github.com/flomesh-io/fsm/pkg/gen/client/plugin/clientset/versioned/fake"
	policyFake "github.com/flomesh-io/fsm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/signals"
	"github.com/flomesh-io/fsm/pkg/tests"
	"github.com/flomesh-io/fsm/pkg/webhook"
)

func BenchmarkDoValidation(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	kubeClient := k8sClientFake.NewSimpleClientset()
	_, cancel := context.WithCancel(context.Background())
	stop := signals.RegisterExitHandlers(cancel)
	msgBroker := messaging.NewBroker(stop)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	policyClient := policyFake.NewSimpleClientset()
	pluginClient := pluginFake.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	informerCollection, err := informers.NewInformerCollection(tests.MeshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		informers.WithConfigClient(configClient, tests.FsmMeshConfigName, tests.FsmNamespace),
		informers.WithPolicyClient(policyClient),
	)
	if err != nil {
		b.Fatalf("Failed to create informer collection: %s", err)
	}
	k8sClient := k8s.NewKubernetesController(informerCollection, policyClient, pluginClient, msgBroker)
	policyController := policy.NewPolicyController(informerCollection, kubeClient, k8sClient, msgBroker)
	kv := &policyValidator{
		policyClient: policyController,
	}

	w := httptest.NewRecorder()
	s := &validatingWebhookServer{
		validators: map[string]validateFunc{
			policyv1alpha1.SchemeGroupVersion.WithKind("IngressBackend").String():         kv.ingressBackendValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("Egress").String():                 egressValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("EgressGateway").String():          kv.egressGatewayValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("UpstreamTrafficSetting").String(): kv.upstreamTrafficSettingValidator,
			smiAccess.SchemeGroupVersion.WithKind("TrafficTarget").String():               trafficTargetValidator,
		},
	}

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		req := &http.Request{
			Header: map[string][]string{
				webhook.HTTPHeaderContentType: {webhook.ContentTypeJSON},
			},
			Body: io.NopCloser(strings.NewReader(`{
				"metadata": {
					"uid": "some-uid"
				},
				"request": {}
			}`)),
		}
		s.doValidation(w, req)
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			b.Fatalf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
		}
	}
	b.StopTimer()
}
