package e2e

import (
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/tests/framework"
	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("TCP server-first traffic",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 1,
	},
	func() {
		var (
			sourceNs = framework.RandomNameWithPrefix("client")
			destNs   = framework.RandomNameWithPrefix("server")
			ns       = []string{sourceNs, destNs}
		)

		It("TCP server-first traffic", func() {
			// Install FSM
			installOpts := Td.GetFSMInstallOpts()
			installOpts.EnablePermissiveMode = true
			Expect(Td.InstallFSM(installOpts)).To(Succeed())

			sidecarClass := Td.GetSidecarClass(Td.FsmNamespace)
			if strings.EqualFold(strings.ToLower(constants.SidecarClassPipy), strings.ToLower(sidecarClass)) {
				Skip("Pending")
			}

			// Create Test NS
			for _, n := range ns {
				Expect(Td.CreateNs(n, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, n)).To(Succeed())
			}

			destinationPort := 80

			// Get simple pod definitions for the TCP server
			svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
				SimplePodAppDef{
					PodName:     framework.RandomNameWithPrefix("server"),
					Namespace:   destNs,
					Image:       "busybox",
					Command:     []string{"nc", "-lkp", strconv.Itoa(destinationPort), "-e", "sh", "-c", "while yes; do :; done"},
					Ports:       []int{destinationPort},
					AppProtocol: constants.ProtocolTCPServerFirst,
					OS:          Td.ClusterOS,
				},
			)

			Expect(err).NotTo(HaveOccurred())

			_, err = Td.CreateServiceAccount(destNs, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(destNs, podDef)
			Expect(err).NotTo(HaveOccurred())
			dstSvc, err := Td.CreateService(destNs, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(destNs, 1, nil)).To(Succeed())

			svcAccDef, podDef, _, err = Td.SimplePodApp(SimplePodAppDef{
				PodName:   framework.RandomNameWithPrefix("client"),
				Namespace: sourceNs,
				Command:   []string{"nc", dstSvc.Name + "." + dstSvc.Namespace, strconv.Itoa(destinationPort)},
				Image:     "busybox",
				OS:        Td.ClusterOS,
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateServiceAccount(sourceNs, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(sourceNs, podDef)
			Expect(err).NotTo(HaveOccurred())

			Expect(Td.WaitForPodsRunningReady(sourceNs, 1, nil)).To(Succeed())

			Eventually(func() (string, error) {
				return getPodLogs(sourceNs, podDef.Name, podDef.Name)
			}, 5*time.Second).Should(ContainSubstring("\ny\n"), "Didn't get expected response from server")
		})
	})
