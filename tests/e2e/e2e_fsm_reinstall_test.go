package e2e

import (
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test reinstalling FSM in the same namespace with the same mesh name",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 3,
	},
	func() {
		It("Becomes ready after being reinstalled", func() {
			opts := Td.GetFSMInstallOpts()
			Expect(Td.InstallFSM(opts)).To(Succeed())

			By("Uninstalling FSM")
			stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/fsm"), "uninstall", "mesh", "-f", "--fsm-namespace", opts.ControlPlaneNS)
			Td.T.Log(stdout)
			if err != nil {
				Td.T.Logf("stderr:\n%s", stderr)
			}
			Expect(err).NotTo(HaveOccurred())

			By("Reinstalling FSM")
			// Invoke the CLI directly because Td.InstallFSM unconditionally
			// creates the namespace which fails when it already exists.
			stdout, stderr, err = Td.RunLocal(filepath.FromSlash("../../bin/fsm"), "install", "--verbose", "--timeout=5m", "--fsm-namespace", opts.ControlPlaneNS, "--set", "fsm.image.registry="+opts.ContainerRegistryLoc+",fsm.image.tag="+opts.FsmImagetag, "--set", "fsm.fsmIngress.enabled=false")
			Td.T.Log(stdout)
			if err != nil {
				Td.T.Logf("stderr:\n%s", stderr)
			}
			Expect(err).NotTo(HaveOccurred())
		})
	})
