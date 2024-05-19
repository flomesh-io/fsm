package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test multiple service ports",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 3,
	},
	func() {
		Context("Multiple service ports", func() {
			testMultipleServicePorts()
		})
	})

// testMultipleServicePorts makes consecutive HTTP requests from a client
// application to a server application. The server application's service
// has multiple ports. This test uncovers a bug in FSM that is caused by
// FSM configuring a proxy upstream cluster to have multiple endpoints in
// the event that there is a service with multiple ports.
//
// Related GitHub issue: https://github.com/flomesh-io/fsm/issues/3777
func testMultipleServicePorts() {
	It("Tests traffic to a service with multiple ports", func() {
		// Install FSM.
		installOpts := Td.GetFSMInstallOpts()
		installOpts.EnablePermissiveMode = true
		Expect(Td.InstallFSM(installOpts)).To(Succeed())

		// Create test namespaces.
		const serverName = "server"
		const clientName = "client"
		var ns = []string{serverName, clientName}

		for _, namespace := range ns {
			Expect(Td.CreateNs(namespace, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, namespace)).To(Succeed())
		}

		// Create an HTTP server that clients will send requests to.
		svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
			SimplePodAppDef{
				PodName:   serverName,
				Namespace: serverName,
				Image:     "flomesh/httpbin:ken",
				// To test multiple ports per service, an additional port 90 is exposed
				// on the service.
				Ports:   []int{DefaultUpstreamServicePort, 90},
				Command: HttpbinCmd,
				OS:      Td.ClusterOS,
			})
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(serverName, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(serverName, podDef)
		Expect(err).NotTo(HaveOccurred())

		// When multiple ports are specified for a server, they must be named.
		// We will also test that when Port and TargetPort are different, traffic
		// works as expected. The httpbin server app listens on 'DefaultUpstreamServicePort',
		// so we will use this as the TargetPort, and change the ServicePort to a different
		// port number.
		servicePort := int32(DefaultUpstreamServicePort) + 1
		svcDef.Spec.Ports[0].Name = "http"
		svcDef.Spec.Ports[0].Port = servicePort
		svcDef.Spec.Ports[1].Name = "tcp"
		serverService, err := Td.CreateService(serverName, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Create the client application.
		srcPod := setupSource(clientName, false)

		Expect(Td.WaitForPodsRunningReady(serverName, 1, nil)).To(Succeed())
		Expect(Td.WaitForPodsRunningReady(clientName, 1, nil)).To(Succeed())

		clientToServerRequest := HTTPRequestDef{
			SourceNs:        clientName,
			SourcePod:       srcPod.Name,
			SourceContainer: srcPod.Name,
			Destination:     fmt.Sprintf("%s.%s:%d", serverService.Name, serverService.Namespace, servicePort),
		}

		srcToDestStr := fmt.Sprintf("%s -> %s",
			fmt.Sprintf("%s/%s", clientToServerRequest.SourceNs, clientToServerRequest.SourcePod),
			clientToServerRequest.Destination)

		cond := Td.WaitForSuccessAfterInitialFailure(func() bool {
			result := Td.HTTPRequest(clientToServerRequest)
			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> (%s) HTTP request failed, statusCode=%d, err=%s", srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP request succeeded, statusCode=%d", srcToDestStr, result.StatusCode)
			return true
		}, 10, 90*time.Second)
		Expect(cond).To(BeTrue())
	})
}
