package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test fsm-mesh-config functionalities",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("When FSM is Installed", func() {
			It("create default MeshConfig resource", func() {

				if Td.InstType == "NoInstall" {
					Skip("Skipping test: NoInstall marked on a test that requires fresh installation")
				}
				instOpts := Td.GetFSMInstallOpts()

				// Install FSM
				Expect(Td.InstallFSM(instOpts)).To(Succeed())
				meshConfig, err := Td.GetMeshConfig(Td.FsmNamespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate fsm MeshConfig
				Expect(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode).Should(BeFalse())
				Expect(meshConfig.Spec.Traffic.EnableEgress).Should(BeFalse())
				Expect(meshConfig.Spec.Sidecar.LogLevel).Should(Equal("debug"))
				Expect(meshConfig.Spec.Observability.Tracing.Enable).Should(BeFalse())
				Expect(meshConfig.Spec.Certificate.ServiceCertValidityDuration).Should(Equal("24h"))
			})
		})
	})
