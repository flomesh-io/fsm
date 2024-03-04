package e2e

import (
	. "github.com/flomesh-io/fsm/tests/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FSMDescribe("Test traffic among FSM Ingress",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 14,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("FSMIngress", func() {
			It("allow traffic through Ingress", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableIngress = true
				installOpts.EnableGateway = false
				installOpts.EnableServiceLB = true

				Expect(Td.InstallFSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 4, nil)).To(Succeed())
			})
		})
	})
