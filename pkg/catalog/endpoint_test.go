package catalog

import (
	"context"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/endpoint"
	"github.com/flomesh-io/fsm/pkg/identity"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/policy"
	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/smi"
	"github.com/flomesh-io/fsm/pkg/tests"
)

func TestListAllowedUpstreamEndpointsForService(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                     string
		proxyIdentity            identity.ServiceIdentity
		upstreamSvc              service.MeshService
		trafficTargets           []*access.TrafficTarget
		services                 []service.MeshService
		outboundServices         map[identity.ServiceIdentity][]service.MeshService
		outboundServiceEndpoints map[service.MeshService][]endpoint.Endpoint
		permissiveMode           bool
		expectedEndpoints        []endpoint.Endpoint
	}{
		{
			name: `Traffic target defined for bookstore ServiceAccount.
			This service account has only bookstore-v1 service on it.
			Hence endpoints returned for bookstore-v1`,
			proxyIdentity:  tests.BookbuyerServiceIdentity,
			upstreamSvc:    tests.BookstoreV1Service,
			trafficTargets: []*access.TrafficTarget{&tests.TrafficTarget},
			services:       []service.MeshService{tests.BookstoreV1Service},
			outboundServices: map[identity.ServiceIdentity][]service.MeshService{
				tests.BookstoreServiceIdentity: {tests.BookstoreV1Service},
			},
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
			},
			permissiveMode:    false,
			expectedEndpoints: []endpoint.Endpoint{tests.Endpoint},
		},
		{
			name: `Traffic target defined for bookstore ServiceAccount.
			This service account has bookstore-v1 bookstore-v2 services,
			but bookstore-v2 pod has service account bookstore-v2.
			Hence no endpoints returned for bookstore-v2`,
			proxyIdentity:  tests.BookbuyerServiceIdentity,
			upstreamSvc:    tests.BookstoreV2Service,
			trafficTargets: []*access.TrafficTarget{&tests.TrafficTarget},
			services:       []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			outboundServices: map[identity.ServiceIdentity][]service.MeshService{
				tests.BookstoreServiceIdentity: {tests.BookstoreV1Service, tests.BookstoreV2Service},
			},
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
				tests.BookstoreV2Service: {endpoint.Endpoint{
					IP:   net.ParseIP("9.9.9.9"),
					Port: endpoint.Port(tests.ServicePort),
				}},
			},
			permissiveMode:    false,
			expectedEndpoints: []endpoint.Endpoint{},
		},
		{
			name: `Traffic target defined for bookstore ServiceAccount.
			This service account has bookstore-v1 bookstore-v2 services,
			since bookstore-v2 pod has service account bookstore-v2 which is allowed in the traffic target.
			Hence endpoints returned for bookstore-v2`,
			proxyIdentity:  tests.BookbuyerServiceIdentity,
			upstreamSvc:    tests.BookstoreV2Service,
			trafficTargets: []*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget},
			services:       []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			outboundServices: map[identity.ServiceIdentity][]service.MeshService{
				tests.BookstoreServiceIdentity:   {tests.BookstoreV1Service},
				tests.BookstoreV2ServiceIdentity: {tests.BookstoreV2Service},
			},
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
				tests.BookstoreV2Service: {endpoint.Endpoint{
					IP:   net.ParseIP("9.9.9.9"),
					Port: endpoint.Port(tests.ServicePort),
				}},
			},
			permissiveMode: false,
			expectedEndpoints: []endpoint.Endpoint{{
				IP:   net.ParseIP("9.9.9.9"),
				Port: endpoint.Port(tests.ServicePort),
			}},
		},
		{
			name:          `Permissive mode should return all endpoints for a service without filtering them`,
			proxyIdentity: tests.BookbuyerServiceIdentity,
			upstreamSvc:   tests.BookstoreV2Service,
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV2Service: {
					{
						IP:   net.ParseIP("1.1.1.1"),
						Port: 80,
					},
					{
						IP:   net.ParseIP("2.2.2.2"),
						Port: 80,
					},
				},
			},
			permissiveMode: true,
			expectedEndpoints: []endpoint.Endpoint{
				{
					IP:   net.ParseIP("1.1.1.1"),
					Port: 80,
				},
				{
					IP:   net.ParseIP("2.2.2.2"),
					Port: 80,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			kubeClient := testclient.NewSimpleClientset()
			defer mockCtrl.Finish()

			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockKubeController := k8s.NewMockController(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)
			mockServiceProvider := service.NewMockProvider(mockCtrl)
			mockPolicyController := policy.NewMockController(mockCtrl)
			mockMeshSpec := smi.NewMockMeshSpec(mockCtrl)

			mc := MeshCatalog{
				kubeController:     mockKubeController,
				meshSpec:           mockMeshSpec,
				endpointsProviders: []endpoint.Provider{mockEndpointProvider},
				serviceProviders:   []service.Provider{mockServiceProvider},
				policyController:   mockPolicyController,
				configurator:       mockConfigurator,
			}

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(tc.permissiveMode).AnyTimes()
			mockPolicyController.EXPECT().ListIsolationPolicies().Return(nil).AnyTimes()

			for svc, endpoints := range tc.outboundServiceEndpoints {
				mockEndpointProvider.EXPECT().ListEndpointsForService(svc).Return(endpoints).AnyTimes()
			}

			if tc.permissiveMode {
				actual := mc.ListAllowedUpstreamEndpointsForService(tc.proxyIdentity, tc.upstreamSvc)
				assert.ElementsMatch(actual, tc.expectedEndpoints)
				return
			}

			mockMeshSpec.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).AnyTimes()

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			for sa, services := range tc.outboundServices {
				for _, svc := range services {
					k8sService := tests.NewServiceFixture(svc.Name, svc.Namespace, map[string]string{})
					mockKubeController.EXPECT().GetService(svc).Return(k8sService).AnyTimes()
				}
				mockServiceProvider.EXPECT().GetServicesForServiceIdentity(sa).Return(services).AnyTimes()
			}

			var pods []*corev1.Pod
			for serviceIdentity, services := range tc.outboundServices {
				// TODO(draychev): use ServiceIdentity in the rest of the tests [https://github.com/flomesh-io/fsm/issues/2218]
				sa := serviceIdentity.ToK8sServiceAccount()
				for _, svc := range services {
					podlabels := map[string]string{
						constants.AppLabel:                 tests.SelectorValue,
						constants.SidecarUniqueIDLabelName: uuid.New().String(),
					}
					pod := tests.NewPodFixture(tests.Namespace, svc.Name, sa.Name, podlabels)
					podEndpoints := tc.outboundServiceEndpoints[svc]
					var podIps []corev1.PodIP
					for _, ep := range podEndpoints {
						podIps = append(podIps, corev1.PodIP{IP: ep.IP.String()})
					}
					pod.Status.PodIPs = podIps
					pod.Spec.ServiceAccountName = sa.Name
					_, err := kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
					assert.Nil(err)
					pods = append(pods, pod)
				}
			}
			mockKubeController.EXPECT().ListPods().Return(pods).AnyTimes()

			for sa, services := range tc.outboundServices {
				for _, svc := range services {
					podEndpoints := tc.outboundServiceEndpoints[svc]
					mockEndpointProvider.EXPECT().ListEndpointsForIdentity(sa).Return(podEndpoints).AnyTimes()
				}
			}

			actual := mc.ListAllowedUpstreamEndpointsForService(tc.proxyIdentity, tc.upstreamSvc)
			assert.ElementsMatch(actual, tc.expectedEndpoints)
		})
	}
}
