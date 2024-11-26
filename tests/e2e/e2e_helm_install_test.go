package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test fsm control plane installation with Helm",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 3,
	},
	func() {
		Context("Helm install using default values", func() {
			It("installs fsm control plane successfully", func() {
				if Td.InstType == NoInstall {
					Skip("Test is not going through InstallFSM, hence cannot be automatically skipped with NoInstall (#1908)")
				}

				namespace := "helm-install-namespace"
				release := "helm-install-fsm"

				// Install FSM with Helm
				Expect(Td.HelmInstallFSM(release, namespace)).To(Succeed())

				meshConfig, err := Td.GetMeshConfig(namespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate fsm MeshConfig
				spec := meshConfig.Spec
				Expect(spec.Traffic.EnablePermissiveTrafficPolicyMode).To(BeTrue())
				Expect(spec.Traffic.EnableEgress).To(BeTrue())
				Expect(spec.Sidecar.LogLevel).To(Equal("error"))
				Expect(spec.Observability.Tracing.Enable).To(BeFalse())
				Expect(spec.Certificate.ServiceCertValidityDuration).To(Equal("24h"))

				Expect(Td.DeleteHelmRelease(release, namespace)).To(Succeed())
			})
		})
	})
