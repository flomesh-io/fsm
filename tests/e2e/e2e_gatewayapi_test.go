package e2e

import (
	. "github.com/flomesh-io/fsm/tests/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FSMDescribe("Test traffic among FSM Gateway",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 13,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("FSMGateway", func() {
			It("allow traffic of multiple protocols through Gateway", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableIngress = false
				installOpts.EnableGateway = true
				installOpts.EnableServiceLB = true

				Expect(Td.InstallFSM(installOpts)).To(Succeed())

				// Create namespaces
				Expect(Td.CreateNs("httpbin", nil)).To(Succeed())

				By("Generating CA private key")
				stdout, stderr, err := Td.RunLocal("openssl", "genrsa", "-out", "ca.key", "2048")
				Td.T.Log(stdout.String())
				if err != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Generating CA certificate")
				stdout, stderr, err = Td.RunLocal("openssl", "req", "-new", "-x509", "-nodes", "-days", "365", "-key", "ca.key", "-out", "ca.crt", "-subj", "/CN=flomesh.io")
				Td.T.Log(stdout.String())
				if err != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating certificate and key for HTTPS")
				stdout, stderr, err = Td.RunLocal("openssl", "req", "-new", "-x509", "-nodes", "-days", "365", "-newkey", "rsa:2048", "-keyout", "https.key", "-out", "https.crt", "-subj", "/CN=httptest.localhost", "-addext", "subjectAltName = DNS:httptest.localhost")
				Td.T.Log(stdout.String())
				if err != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating secret for HTTPS")
				stdout, stderr, err = Td.RunLocal("kubectl", "-n", "httpbin", "create", "secret", "generic", "https-cert", "--from-file=ca.crt=./ca.crt", "--from-file=tls.crt=./https.crt", "--from-file=tls.key=./https.key")
				Td.T.Log(stdout.String())
				if err != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
