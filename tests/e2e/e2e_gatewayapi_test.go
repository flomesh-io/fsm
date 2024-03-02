package e2e

import (
	"fmt"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"k8s.io/apimachinery/pkg/api/resource"

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

				testDeployGateway()

				testHTTP()
				testHTTPS()
				testTLSTerminate()

				testGRPC()
				testGRPCS()

				testTCP()
				testUDP()
				testTLSPassthrough()
			})
		})
	})

func testDeployGateway() {
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
}

func testHTTP() {
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

	_, err := Td.CreateDeployment(nsHttpbin, httpbinDeploy)
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
}

func testGRPC() {
	By("Deploying app in namespace grpcbin")
	grpcDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGrpcbin,
			Name:      "grpcbin",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "grpcbin"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "grpcbin"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "grpcbin",
							Image: "flomesh/grpcbin",
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpcbin",
									ContainerPort: 9000,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("50Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsGrpcbin, grpcDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsGrpcbin, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "grpcbin"},
	})).To(Succeed())

	By("Creating svc for grpcbin")
	grpcbinSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGrpcbin,
			Name:      "grpcbin",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       9000,
					TargetPort: intstr.FromInt32(9000),
				},
			},
			Selector: map[string]string{"app": "grpcbin"},
		},
	}
	_, err = Td.CreateService(nsGrpcbin, grpcbinSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating GRPCRoute for testing GRPC protocol")
	grpcRoute := gwv1alpha2.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGrpcbin,
			Name:      "grpc-app-1",
		},
		Spec: gwv1alpha2.GRPCRouteSpec{
			CommonRouteSpec: gwv1alpha2.CommonRouteSpec{
				ParentRefs: []gwv1alpha2.ParentReference{
					{
						Namespace: namespacePtr(corev1.NamespaceDefault),
						Name:      "test-gw-1",
						Port:      portPtr(8090),
					},
				},
			},
			Hostnames: []gwv1alpha2.Hostname{"grpctest.localhost"},
			Rules: []gwv1alpha2.GRPCRouteRule{
				{
					Matches: []gwv1alpha2.GRPCRouteMatch{
						{
							Method: &gwv1alpha2.GRPCMethodMatch{
								Type:    grpcMethodMatchTypePtr(gwv1alpha2.GRPCMethodMatchExact),
								Service: pointer.String("hello.HelloService"),
								Method:  pointer.String("SayHello"),
							},
						},
					},
					BackendRefs: []gwv1alpha2.GRPCBackendRef{
						{
							BackendRef: gwv1alpha2.BackendRef{
								BackendObjectReference: gwv1alpha2.BackendObjectReference{
									Name: "grpcbin",
									Port: portPtr(9000),
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPIGRPCRoute(nsGrpcbin, grpcRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing GRPCRoute")
	grpcReq := GRPCRequestDef{
		Destination: "grpctest.localhost:8090",
		Symbol:      "hello.HelloService/SayHello",
		JSONRequest: `{"greeting":"Flomesh"}`,
	}
	srcToDestStr := fmt.Sprintf("%s -> %s/%s", "grpcurl", grpcReq.Destination, grpcReq.Symbol)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalGRPCRequest(grpcReq)

		if result.Err != nil {
			Td.T.Logf("> (%s) gRPC req failed, response: %s, err: %s",
				srcToDestStr, result.Response, result.Err)
			return false
		}

		Td.T.Logf("> (%s) gRPC req succeeded, response: %s", srcToDestStr, result.Response)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing GRPC traffic from grpcurl(localhost) to destination %s/%s", grpcReq.Destination, grpcReq.Symbol)
}

func testTCP() {
	By("Deploying app in namespace tcproute")
	tcpDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTcproute,
			Name:      "tcp-echo",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "tcp-echo"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "tcp-echo"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tcp",
							Image: "istio/fortio:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "tcp",
									ContainerPort: 8078,
								},
							},
							Command: []string{"fortio", "tcp-echo"},
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsTcproute, tcpDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsTcproute, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "tcp-echo"},
	})).To(Succeed())

	By("Creating svc for tcproute")
	tcpSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTcproute,
			Name:      "tcp-echo",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "tcp",
					Protocol:   corev1.ProtocolTCP,
					Port:       8078,
					TargetPort: intstr.FromInt32(8078),
				},
			},
			Selector: map[string]string{constants.AppLabel: "tcp-echo"},
		},
	}
	_, err = Td.CreateService(nsTcproute, tcpSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating TCPRoute for testing TCP protocol")
	tcpRoute := gwv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTcproute,
			Name:      "tcp-app-1",
		},
		Spec: gwv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Namespace: namespacePtr(corev1.NamespaceDefault),
						Name:      "test-gw-1",
						Port:      portPtr(3000),
					},
				},
			},
			Rules: []gwv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Name: "tcp-echo",
								Port: portPtr(8078),
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPITCPRoute(nsTcproute, tcpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing TCPRoute")
	tcpReq := TCPRequestDef{
		DestinationHost: "localhost",
		DestinationPort: 3000,
		Message:         "Hi, I am TCP!",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s:%d", "client", tcpReq.DestinationHost, tcpReq.DestinationPort)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalTCPRequest(tcpReq)

		if result.Err != nil {
			Td.T.Logf("> (%s) TCP req failed, response: %s, err: %s", srcToDestStr, result.Response, result.Err)
			return false
		}

		Td.T.Logf("> (%s) TCP req succeeded, response: %s", srcToDestStr, result.Response)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing TCP traffic from echo/nc(localhost) to destination %s:%d", tcpReq.DestinationHost, tcpReq.DestinationPort)
}

func testUDP() {
	By("Deploying app in namespace udproute")
}

func testHTTPS() {
	By("Creating HTTPRoute for testing HTTPs protocol")
	httpRoute := gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsHttpbin,
			Name:      "https-app-1",
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Namespace: namespacePtr(corev1.NamespaceDefault),
						Name:      "test-gw-1",
						Port:      portPtr(7443),
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
	_, err := Td.CreateGatewayAPIHTTPRoute(nsHttpbin, httpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing HTTPRoute(HTTPS)")
	httpsReq := HTTPRequestDef{
		Destination: "https://httptest.localhost:7443/bar",
		UseTLS:      true,
		CertFile:    "https.crt",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s", "curl", httpsReq.Destination)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalHTTPRequest(httpsReq)

		if result.Err != nil || result.StatusCode != 200 {
			Td.T.Logf("> (%s) HTTPs Req failed %d %v",
				srcToDestStr, result.StatusCode, result.Err)
			return false
		}
		Td.T.Logf("> (%s) HTTPs Req succeeded: %d", srcToDestStr, result.StatusCode)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing HTTPs traffic from curl(localhost) to destination %s", httpsReq.Destination)
}

func testGRPCS() {
	By("Creating GRPCRoute for testing GRPCs protocol")
	grpcRoute := gwv1alpha2.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGrpcbin,
			Name:      "grpcs-app-1",
		},
		Spec: gwv1alpha2.GRPCRouteSpec{
			CommonRouteSpec: gwv1alpha2.CommonRouteSpec{
				ParentRefs: []gwv1alpha2.ParentReference{
					{
						Namespace: namespacePtr(corev1.NamespaceDefault),
						Name:      "test-gw-1",
						Port:      portPtr(7443),
					},
				},
			},
			Hostnames: []gwv1alpha2.Hostname{"grpctest.localhost"},
			Rules: []gwv1alpha2.GRPCRouteRule{
				{
					Matches: []gwv1alpha2.GRPCRouteMatch{
						{
							Method: &gwv1alpha2.GRPCMethodMatch{
								Type:    grpcMethodMatchTypePtr(gwv1alpha2.GRPCMethodMatchExact),
								Service: pointer.String("hello.HelloService"),
								Method:  pointer.String("SayHello"),
							},
						},
					},
					BackendRefs: []gwv1alpha2.GRPCBackendRef{
						{
							BackendRef: gwv1alpha2.BackendRef{
								BackendObjectReference: gwv1alpha2.BackendObjectReference{
									Name: "grpcbin",
									Port: portPtr(9000),
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := Td.CreateGatewayAPIGRPCRoute(nsGrpcbin, grpcRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing GRPCRoute")
	grpcReq := GRPCRequestDef{
		Destination: "grpctest.localhost:8090",
		Symbol:      "hello.HelloService/SayHello",
		JSONRequest: `{"greeting":"Flomesh"}`,
		UseTLS:      true,
		CertFile:    "grpc.crt",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s/%s", "grpcurl", grpcReq.Destination, grpcReq.Symbol)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalGRPCRequest(grpcReq)

		if result.Err != nil {
			Td.T.Logf("> (%s) gRPCs req failed, response: %s, err: %s",
				srcToDestStr, result.Response, result.Err)
			return false
		}

		Td.T.Logf("> (%s) gRPCs req succeeded, response: %s", srcToDestStr, result.Response)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing GRPCs traffic from grpcurl(localhost) to destination %s/%s", grpcReq.Destination, grpcReq.Symbol)

}

func testTLSPassthrough() {

}

func testTLSTerminate() {

}

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

func grpcMethodMatchTypePtr(grpcMatch gwv1alpha2.GRPCMethodMatchType) *gwv1alpha2.GRPCMethodMatchType {
	ret := grpcMatch
	return &ret
}
