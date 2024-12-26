package e2e

import (
	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test traffic routing by FSM Gateway with trafficInterceptionMode(NodeLevel)",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 7,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Test traffic from client to backend service routing by FSM Gateway", func() {
			It("allow traffic of multiple protocols through Gateway", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableIngress = false
				installOpts.EnableGateway = true
				installOpts.EnableServiceLB = true
				installOpts.TrafficInterceptionMode = constants.TrafficInterceptionModeNodeLevel

				Expect(Td.InstallFSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 3, &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/instance": "fsm",
						"app.kubernetes.io/name":     "flomesh.io",
					},
				})).To(Succeed())

				testDeployFSMGateway()

				testFSMGatewayHTTPTrafficSameNamespace()
				testFSMGatewayHTTPTrafficCrossNamespace()
				testFSMGatewayHTTPSTraffic()
				testFSMGatewayTLSTerminate()
				testFSMGatewayTLSPassthrough()

				testFSMGatewayGRPCTrafficSameNamespace()
				testFSMGatewayGRPCTrafficCrossNamespace()
				testFSMGatewayGRPCSTraffic()

				testFSMGatewayTCPTrafficSameNamespace()
				testFSMGatewayTCPTrafficCrossNamespace()
				testFSMGatewayUDPTrafficSameNamespace()
				testFSMGatewayUDPTrafficCrossNamespace()
			})
		})
	})
