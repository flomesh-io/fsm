package e2e

import (
	. "github.com/flomesh-io/fsm/tests/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FSMDescribe("Test traffic among FSM NamespacedIngress",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 14,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("NamespacedIngress", func() {
			It("allow traffic through NamespacedIngress", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableIngress = true
				installOpts.EnableNamespacedIngress = true
				installOpts.EnableGateway = false
				installOpts.EnableServiceLB = true

				Expect(Td.InstallFSM(installOpts)).To(Succeed())
				//Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 3, nil)).To(Succeed())
			})
		})
	})
