// Package fake implements Fake's methods.
package fake

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	tresorFake "github.com/flomesh-io/fsm/pkg/certificate/providers/tresor/fake"
	configClientset "github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned"
	kubeFake "github.com/flomesh-io/fsm/pkg/providers/kube/fake"
	smiFake "github.com/flomesh-io/fsm/pkg/smi/fake"

	"github.com/flomesh-io/fsm/pkg/catalog"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/informers"
	"github.com/flomesh-io/fsm/pkg/messaging"
	"github.com/flomesh-io/fsm/pkg/multicluster"
	"github.com/flomesh-io/fsm/pkg/plugin"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/tests"
)

// NewFakeMeshCatalog creates a new struct implementing catalog.MeshCataloger interface used for testing.
func NewFakeMeshCatalog(kubeClient kubernetes.Interface, meshConfigClient configClientset.Interface) *catalog.MeshCatalog {
	mockCtrl := gomock.NewController(ginkgo.GinkgoT())
	mockKubeController := k8s.NewMockController(mockCtrl)
	mockPolicyController := policy.NewMockController(mockCtrl)
	mockPluginController := plugin.NewMockController(mockCtrl)
	mockMultiClusterController := multicluster.NewMockController(mockCtrl)

	meshSpec := smiFake.NewFakeMeshSpecClient()

	stop := make(<-chan struct{})

	provider := kubeFake.NewFakeProvider()

	endpointProviders := []endpoint.Provider{
		provider,
	}
	serviceProviders := []service.Provider{
		provider,
	}

	fsmNamespace := "-test-fsm-namespace-"
	fsmMeshConfigName := "-test-fsm-mesh-config-"
	ic, err := informers.NewInformerCollection("fsm", stop, informers.WithKubeClient(kubeClient), informers.WithConfigClient(meshConfigClient, fsmMeshConfigName, fsmNamespace))
	if err != nil {
		return nil
	}

	cfg := configurator.NewConfigurator(ic, fsmNamespace, fsmMeshConfigName, nil)

	certManager := tresorFake.NewFake(nil, 1*time.Hour)

	// #1683 tracks potential improvements to the following dynamic mocks
	mockKubeController.EXPECT().ListServices(true, true).DoAndReturn(func() []*corev1.Service {
		// play pretend this call queries a controller cache
		var services []*corev1.Service

		// This assumes that catalog tests use monitored namespaces at all times
		svcList, _ := kubeClient.CoreV1().Services("").List(context.Background(), metav1.ListOptions{})
		for idx := range svcList.Items {
			services = append(services, &svcList.Items[idx])
		}

		return services
	}).AnyTimes()
	mockKubeController.EXPECT().ListServiceAccounts(true).DoAndReturn(func() []*corev1.ServiceAccount {
		// play pretend this call queries a controller cache
		var serviceAccounts []*corev1.ServiceAccount

		// This assumes that catalog tests use monitored namespaces at all times
		svcAccountList, _ := kubeClient.CoreV1().ServiceAccounts("").List(context.Background(), metav1.ListOptions{})
		for idx := range svcAccountList.Items {
			serviceAccounts = append(serviceAccounts, &svcAccountList.Items[idx])
		}

		return serviceAccounts
	}).AnyTimes()
	mockKubeController.EXPECT().GetService(gomock.Any()).DoAndReturn(func(msh service.MeshService) *corev1.Service {
		// play pretend this call queries a controller cache
		vv, err := kubeClient.CoreV1().Services(msh.Namespace).Get(context.Background(), msh.Name, metav1.GetOptions{})
		if err != nil {
			return nil
		}

		return vv
	}).AnyTimes()
	mockKubeController.EXPECT().ListPods().DoAndReturn(func() []*corev1.Pod {
		vv, err := kubeClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil
		}

		podRet := []*corev1.Pod{}
		for idx := range vv.Items {
			podRet = append(podRet, &vv.Items[idx])
		}
		return podRet
	}).AnyTimes()

	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreV1Service.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookstoreV2Service.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookbuyerService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().IsMonitoredNamespace(tests.BookwarehouseService.Namespace).Return(true).AnyTimes()
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookstoreV1Service).Return([]identity.K8sServiceAccount{tests.BookstoreServiceAccount}, nil).AnyTimes()
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookstoreV2Service).Return([]identity.K8sServiceAccount{tests.BookstoreV2ServiceAccount}, nil).AnyTimes()
	mockKubeController.EXPECT().ListServiceIdentitiesForService(tests.BookbuyerService).Return([]identity.K8sServiceAccount{tests.BookbuyerServiceAccount}, nil).AnyTimes()

	mockPolicyController.EXPECT().ListEgressPoliciesForSourceIdentity(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetIngressBackendPolicy(gomock.Any()).Return(nil).AnyTimes()
	mockPolicyController.EXPECT().GetUpstreamTrafficSetting(gomock.Any()).Return(nil).AnyTimes()

	mockKubeController.EXPECT().GetTargetPortForServicePort(gomock.Any(), gomock.Any()).DoAndReturn(
		func(namespacedSvc types.NamespacedName, port uint16) (uint16, error) {
			return port, nil
		}).AnyTimes()
	mockMultiClusterController.EXPECT().GetTargetPortForServicePort(gomock.Any(), gomock.Any()).DoAndReturn(
		func(namespacedSvc types.NamespacedName, port uint16) map[uint16]bool {
			return nil
		}).AnyTimes()

	return catalog.NewMeshCatalog(mockKubeController, meshSpec, certManager,
		mockPolicyController, mockPluginController, mockMultiClusterController, stop, cfg, serviceProviders, endpointProviders, messaging.NewBroker(stop))
}
