package e2e

import (
	"fmt"

	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	nsHttpbin  = "httpbin"
	nsGrpcbin  = "grpcbin"
	nsTcproute = "tcproute"
	nsUdproute = "udproute"
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
				Expect(Td.CreateNs(nsHttpbin, nil)).To(Succeed())
				Expect(Td.CreateNs(nsGrpcbin, nil)).To(Succeed())
				Expect(Td.CreateNs(nsTcproute, nil)).To(Succeed())
				Expect(Td.CreateNs(nsUdproute, nil)).To(Succeed())

				By("Generating CA private key")
				stdout, stderr, err := Td.RunLocal("openssl", "genrsa", "-out", "ca.key", "2048")
				Td.T.Log(stdout.String())
				if stderr != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Generating CA certificate")
				stdout, stderr, err = Td.RunLocal("openssl", "req", "-new", "-x509", "-nodes", "-days", "365", "-key", "ca.key", "-out", "ca.crt", "-subj", "/CN=flomesh.io")
				Td.T.Log(stdout.String())
				if stderr != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating certificate and key for HTTPS")
				stdout, stderr, err = Td.RunLocal("openssl", "req", "-new", "-x509", "-nodes", "-days", "365", "-newkey", "rsa:2048", "-keyout", "https.key", "-out", "https.crt", "-subj", "/CN=httptest.localhost", "-addext", "subjectAltName = DNS:httptest.localhost")
				Td.T.Log(stdout.String())
				if stderr != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating secret for HTTPS")
				stdout, stderr, err = Td.RunLocal("kubectl", "-n", nsHttpbin, "create", "secret", "generic", "https-cert", "--from-file=ca.crt=./ca.crt", "--from-file=tls.crt=./https.crt", "--from-file=tls.key=./https.key")
				Td.T.Log(stdout.String())
				if stderr != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating certificate and key for gRPC")
				stdout, stderr, err = Td.RunLocal("openssl", "req", "-new", "-x509", "-nodes", "-days", "365", "-newkey", "rsa:2048", "-keyout", "grpc.key", "-out", "grpc.crt", "-subj", "/CN=grpctest.localhost", "-addext", "subjectAltName = DNS:grpctest.localhost")
				Td.T.Log(stdout.String())
				if stderr != nil {
					Td.T.Log("stderr:\n" + stderr.String())
				}
				Expect(err).NotTo(HaveOccurred())

				By("Creating secret for gRPC")
				stdout, stderr, err = Td.RunLocal("kubectl", "-n", nsGrpcbin, "create", "secret", "tls", "grpc-cert", "--key", "grpc.key", "--cert", "grpc.crt")
				Td.T.Log(stdout.String())
				if stderr != nil {
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
											Namespace: namespacePtr(nsHttpbin),
										},
										{
											Name:      "grpc-cert",
											Namespace: namespacePtr(nsGrpcbin),
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
											Namespace: namespacePtr(nsHttpbin),
										},
										{
											Name:      "grpc-cert",
											Namespace: namespacePtr(nsGrpcbin),
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

				By("Deploying app in namespace httpbin")
				httpbinDeploy := appv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: nsHttpbin,
						Name:      "httpbin",
					},
					Spec: appv1.DeploymentSpec{
						Replicas: pointer.Int32(1),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{constants.AppLabel: "pipy"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{constants.AppLabel: "pipy"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "pipy",
										Image: "flomesh/pipy:latest",
										Ports: []corev1.ContainerPort{
											{
												Name:          "pipy",
												ContainerPort: 8080,
											},
										},
										Command: []string{"pipy", "-e", "pipy().listen(8080).serveHTTP(new Message('Hi, I am HTTPRoute!'))"},
									},
								},
							},
						},
					},
				}

				_, err = Td.CreateDeployment(nsHttpbin, httpbinDeploy)
				Expect(err).NotTo(HaveOccurred())
				Expect(Td.WaitForPodsRunningReady(nsHttpbin, 1, &metav1.LabelSelector{
					MatchLabels: map[string]string{constants.AppLabel: "pipy"},
				})).To(Succeed())

				By("Creating svc for httpbin")
				httpbinSvc := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: nsHttpbin,
						Name:      "httpbin",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "pipy",
								Protocol:   corev1.ProtocolTCP,
								Port:       8080,
								TargetPort: intstr.FromInt32(8080),
							},
						},
						Selector: map[string]string{"app": "pipy"},
					},
				}
				_, err = Td.CreateService(nsHttpbin, httpbinSvc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating HTTPRoute for testing HTTP protocol")
				httpRoute := gwv1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: nsHttpbin,
						Name:      "http-app-1",
					},
					Spec: gwv1.HTTPRouteSpec{
						CommonRouteSpec: gwv1.CommonRouteSpec{
							ParentRefs: []gwv1.ParentReference{
								{
									Namespace: namespacePtr(corev1.NamespaceDefault),
									Name:      "test-gw-1",
									Port:      portPtr(8090),
								},
							},
						},
						Hostnames: []gwv1.Hostname{"httptest.localhost"},
						Rules: []gwv1.HTTPRouteRule{
							{
								Matches: []gwv1.HTTPRouteMatch{
									{
										Path: &gwv1.HTTPPathMatch{
											Type:  pathMatchTypePtr(gwv1.PathMatchPathPrefix),
											Value: pointer.String("/bar"),
										},
									},
								},
								BackendRefs: []gwv1.HTTPBackendRef{
									{
										BackendRef: gwv1.BackendRef{
											BackendObjectReference: gwv1.BackendObjectReference{
												Name: "httpbin",
												Port: portPtr(8080),
											},
										},
									},
								},
							},
						},
					},
				}
				_, err = Td.CreateGatewayAPIHTTPRoute(nsHttpbin, httpRoute)
				Expect(err).NotTo(HaveOccurred())

				By("Testing HTTPRoute")
				httpReq := HTTPRequestDef{
					Destination: "http://httptest.localhost:8090/bar",
				}
				srcToDestStr := fmt.Sprintf("%s -> %s", "curl", httpReq.Destination)

				cond := Td.WaitForRepeatedSuccess(func() bool {
					result := Td.LocalHTTPRequest(httpReq)

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("> (%s) HTTP Req failed %d %v",
							srcToDestStr, result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
					return true
				}, 5, Td.ReqSuccessTimeout)

				Expect(cond).To(BeTrue(), "Failed testing HTTP traffic from curl(localhost) to destination %s", httpReq.Destination)
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

func portPtr(port int32) *gwv1.PortNumber {
	ret := gwv1.PortNumber(port)
	return &ret
}

func pathMatchTypePtr(pathMatch gwv1.PathMatchType) *gwv1.PathMatchType {
	ret := pathMatch
	return &ret
}
