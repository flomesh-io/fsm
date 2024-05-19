package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flomesh-io/fsm/tests/framework"
	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test SMI TrafficTarget Validation",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 4,
	},
	func() {
		Context("With SMI Traffic Target validation enabled", func() {
			var (
				source                  = framework.RandomNameWithPrefix("source")
				destination             = framework.RandomNameWithPrefix("dest")
				namespaceOutsideTheMesh = framework.RandomNameWithPrefix("outside-mesh")
				namespaces              = []string{source, destination}
			)

			It("only allows SMI traffic target to be created when traffic target namespace matches destination namespace and only validates SMI Traffic Targets in a monitored namespace",
				func() {
					// Install FSM
					Expect(Td.InstallFSM(Td.GetFSMInstallOpts())).To(Succeed())

					// Create Test NS
					for _, n := range namespaces {
						Expect(Td.CreateNs(n, nil)).To(Succeed())
						Expect(Td.AddNsToMesh(true, n)).To(Succeed())
					}
					Expect(Td.CreateNs(namespaceOutsideTheMesh, nil)).To(Succeed())

					httpRouteGroup, trafficTarget := createPolicyForRoutePath(source, source, destination, destination, "/")

					// try creating traffic target and http route group in source namespace
					_, err := Td.CreateHTTPRouteGroup(source, httpRouteGroup)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateTrafficTarget(source, trafficTarget)
					Expect(err).To(HaveOccurred())

					// create traffic target in destination namespace
					_, err = Td.CreateHTTPRouteGroup(destination, httpRouteGroup)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateTrafficTarget(destination, trafficTarget)
					Expect(err).NotTo(HaveOccurred())

					// create traffic target and http route group in a namespace outside of the mesh
					_, err = Td.CreateHTTPRouteGroup(namespaceOutsideTheMesh, httpRouteGroup)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateTrafficTarget(namespaceOutsideTheMesh, trafficTarget)
					Expect(err).NotTo(HaveOccurred())
				})
		})

		Context("With SMI validation disabled ", func() {
			It("allows SMI traffic target to be created regardless of whether the namespace matches the destination namespace in any namespace", func() {
				if Td.InstType == NoInstall {
					Skip("SMI Validation is not configurable via MeshConfig so cannot be tested with NoInstall")
				}
				var (
					source                  = framework.RandomNameWithPrefix("source")
					destination             = framework.RandomNameWithPrefix("destination")
					namespaceOutsideTheMesh = framework.RandomNameWithPrefix("outside-mesh")
					namespaces              = []string{source, destination}
				)

				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.SetOverrides = []string{
					// turn off traffic target validation
					"smi.validateTrafficTarget=false",
				}
				Expect(Td.InstallFSM(installOpts)).To(Succeed())

				// Create Test NS
				for _, n := range namespaces {
					Expect(Td.CreateNs(n, nil)).To(Succeed())
					Expect(Td.AddNsToMesh(true, n)).To(Succeed())
				}

				Expect(Td.CreateNs(namespaceOutsideTheMesh, nil)).To(Succeed())

				httpRouteGroup, trafficTarget := createPolicyForRoutePath(source, source, destination, destination, "/")

				_, err := Td.CreateHTTPRouteGroup(source, httpRouteGroup)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(source, trafficTarget)
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateHTTPRouteGroup(destination, httpRouteGroup)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(destination, trafficTarget)
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateHTTPRouteGroup(namespaceOutsideTheMesh, httpRouteGroup)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(namespaceOutsideTheMesh, trafficTarget)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
