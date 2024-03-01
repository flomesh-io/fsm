package e2e

import (
	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
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
				Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 3, nil)).To(Succeed())

				// Create namespaces
				Expect(Td.CreateNs("httpbin", nil)).To(Succeed())
				Expect(Td.CreateNs("grpcbin", nil)).To(Succeed())
				Expect(Td.CreateNs("tcproute", nil)).To(Succeed())
				Expect(Td.CreateNs("udproute", nil)).To(Succeed())

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

				By("Creating certificate and key for gRPC")
				stdout, stderr, err = Td.RunLocal("openssl", "req", "-new", "-x509", "-nodes", "-days", "365", "-newkey", "rsa:2048", "-keyout", "grpc.key", "-out", "grpc.crt", "-subj", "/CN=grpctest.localhost", "-addext", "subjectAltName = DNS:grpctest.localhost")
				Td.T.Log(stdout.String())
				if err != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating secret for gRPC")
				stdout, stderr, err = Td.RunLocal("kubectl", "-n", "grpcbin", "create", "secret", "tls", "grpc-cert", "--key", "grpc.key", "--cert", "grpc.crt")
				Td.T.Log(stdout.String())
				if err != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Deploy Gateway")
				gateway := gwv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-gw-1",
						Annotations: map[string]string{
							"gateway.flomesh.io/replicas":     "2",
							"gateway.flomesh.io/cpu":          "100m",
							"gateway.flomesh.io/cpu-limit":    "1000m",
							"gateway.flomesh.io/memory":       "256Mi",
							"gateway.flomesh.io/memory-limit": "1024Mi",
						},
					},

					Spec: gwv1.GatewaySpec{
						GatewayClassName: "fsm-gateway-cls",
						Listeners: []gwv1.Listener{
							{
								Port:     8090,
								Name:     "http",
								Protocol: gwv1.HTTPProtocolType,
							},
							{
								Port:     3000,
								Name:     "tcp",
								Protocol: gwv1.TCPProtocolType,
							},
							{
								Port:     4000,
								Name:     "udp",
								Protocol: gwv1.UDPProtocolType,
							},
							{
								Port:     7443,
								Name:     "https",
								Protocol: gwv1.HTTPSProtocolType,
								TLS: &gwv1.GatewayTLSConfig{
									CertificateRefs: []gwv1.SecretObjectReference{
										{
											Name:      "https-cert",
											Namespace: namespacePtr("httpbin"),
										},
										{
											Name:      "grpc-cert",
											Namespace: namespacePtr("grpcbin"),
										},
									},
								},
							},
							{
								Port:     8443,
								Name:     "tlsp",
								Protocol: gwv1.TLSProtocolType,
								Hostname: hostnamePtr("httptest.localhost"),
								TLS: &gwv1.GatewayTLSConfig{
									Mode: tlsModePtr(gwv1.TLSModePassthrough),
								},
							},
							{
								Port:     9443,
								Name:     "tlst",
								Protocol: gwv1.TLSProtocolType,
								Hostname: hostnamePtr("httptest.localhost"),
								TLS: &gwv1.GatewayTLSConfig{
									Mode: tlsModePtr(gwv1.TLSModeTerminate),
									CertificateRefs: []gwv1.SecretObjectReference{
										{
											Name:      "https-cert",
											Namespace: namespacePtr("httpbin"),
										},
										{
											Name:      "grpc-cert",
											Namespace: namespacePtr("grpcbin"),
										},
									},
								},
							},
						},
						Infrastructure: &gwv1.GatewayInfrastructure{
							Annotations: map[gwv1.AnnotationKey]gwv1.AnnotationValue{"xyz": "abc"},
							Labels:      map[gwv1.AnnotationKey]gwv1.AnnotationValue{"test": "demo"},
						},
					},
				}
				_, err = Td.CreateGateway(corev1.NamespaceDefault, gateway)
				Expect(err).NotTo(HaveOccurred())
				// Expect it to be up and running in default namespace
				Expect(Td.WaitForPodsRunningReady(corev1.NamespaceDefault, 1, &metav1.LabelSelector{
					MatchLabels: map[string]string{constants.AppLabel: constants.FSMGatewayName},
				})).To(Succeed())

				//Td.CreateDeployment()
				//Td.CreateService()
			})
		})
	})

func namespacePtr(ns string) *gwv1.Namespace {
	ret := gwv1.Namespace(ns)
	return &ret
}

func hostnamePtr(hostname string) *gwv1.Hostname {
	ret := gwv1.Hostname(hostname)
	return &ret
}

func tlsModePtr(mode gwv1.TLSModeType) *gwv1.TLSModeType {
	ret := mode
	return &ret
}
