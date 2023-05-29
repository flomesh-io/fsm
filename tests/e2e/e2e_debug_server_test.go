package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flomesh-io/fsm/tests/framework"
	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test Debug Server by toggling enableDebugServer",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("DebugServer", func() {
			var sourceNs = framework.RandomNameWithPrefix("client")

			It("Starts debug server only when enableDebugServer flag is enabled", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableDebugServer = false
				Expect(Td.InstallFSM(installOpts)).To(Succeed())
				meshConfig, _ := Td.GetMeshConfig(Td.FsmNamespace)

				// Create Test NS
				Expect(Td.CreateNs(sourceNs, nil)).To(Succeed())

				// Get simple Pod definitions for the client
				svcAccDef, podDef, svcDef, err := Td.SimplePodApp(SimplePodAppDef{
					PodName:   "client",
					Namespace: sourceNs,
					Command:   []string{"/bin/bash", "-c", "--"},
					Args:      []string{"while true; do sleep 30; done;"},
					Image:     "flomesh/alpine-debug",
					Ports:     []int{80},
					OS:        Td.ClusterOS,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(sourceNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				srcPod, err := Td.CreatePod(sourceNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(sourceNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(sourceNs, 1, nil)).To(Succeed())

				controllerDest := "fsm-controller." + Td.FsmNamespace + ":9092/debug"

				req := HTTPRequestDef{
					SourceNs:        srcPod.Namespace,
					SourcePod:       srcPod.Name,
					SourceContainer: srcPod.Name,

					Destination: controllerDest,
				}

				iterations := 2
				for i := 1; i <= iterations; i++ {
					By(fmt.Sprintf("(%d/%d) Ensuring debug server is available when enableDebugServer is enabled", i, iterations))

					meshConfig.Spec.Observability.EnableDebugServer = true
					meshConfig, err = Td.UpdateFSMConfig(meshConfig)
					Expect(err).NotTo(HaveOccurred())

					cond := Td.WaitForRepeatedSuccess(func() bool {
						result := Td.HTTPRequest(req)

						if result.Err != nil || result.StatusCode != 200 {
							Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
							return false
						}
						Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
						return true
					}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
					Expect(cond).To(BeTrue())

					By(fmt.Sprintf("(%d/%d) Ensuring debug server is unavailable when enableDebugServer is disabled", i, iterations))

					meshConfig.Spec.Observability.EnableDebugServer = false
					meshConfig, err = Td.UpdateFSMConfig(meshConfig)
					Expect(err).NotTo(HaveOccurred())

					cond = Td.WaitForRepeatedSuccess(func() bool {
						result := Td.HTTPRequest(req)

						if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
							Td.T.Logf("> REST req received unexpected response (status: %d) %v", result.StatusCode, result.Err)
							return false
						}
						Td.T.Logf("> REST req succeeded, got expected error: %v", result.Err)
						return true
					}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
					Expect(cond).To(BeTrue())
				}
			})
		})
	})
