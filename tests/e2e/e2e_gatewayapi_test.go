package e2e

import (
	"fmt"
	"strings"

	gwpav1alpha2 "github.com/flomesh-io/fsm/pkg/apis/policyattachment/v1alpha2"

	extv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/extension/v1alpha1"

	gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"k8s.io/utils/ptr"

	gwv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
)

const (
	nsGateway    = "test"
	nsHTTPRoute  = "http-route"
	nsHTTPSvc    = "http"
	nsGRPCRoute  = "grpc-route"
	nsGRPCSvc    = "grpc"
	nsTCPRoute   = "tcp-route"
	nsTCPSvc     = "tcp"
	nsUDPRoute   = "udp-route"
	nsUDPSvc     = "udp"
	nsKubeSystem = "kube-system"
	nsDefault    = "default"
)

var _ = FSMDescribe("Test traffic routing by FSM Gateway with trafficInterceptionMode(PodLevel)",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 6,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Test traffic from client to backend service routing by FSM Gateway", func() {
			It("allow traffic of multiple protocols through Gateway", func() {
				testGatewayAPI(constants.TrafficInterceptionModePodLevel)
			})
		})
	})

var _ = FSMDescribe("Test traffic routing by FSM Gateway with trafficInterceptionMode(NodeLevel)",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 7,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Test traffic from client to backend service routing by FSM Gateway", func() {
			It("allow traffic of multiple protocols through Gateway", func() {
				testGatewayAPI(constants.TrafficInterceptionModeNodeLevel)
			})
		})
	})

func testGatewayAPI(trafficInterceptionMode string) {
	// Install FSM
	installOpts := Td.GetFSMInstallOpts()
	installOpts.EnableIngress = false
	installOpts.EnableGateway = true
	installOpts.EnableServiceLB = Td.InstType == KindCluster
	installOpts.TrafficInterceptionMode = trafficInterceptionMode

	Expect(Td.InstallFSM(installOpts)).To(Succeed())
	Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 3, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/instance": "fsm",
			"app.kubernetes.io/name":     "flomesh.io",
		},
	})).To(Succeed())

	testDeployFSMGateway()

	testFSMGatewayHTTPTrafficSameNamespace()
	testFSMGatewayHTTPTrafficCrossNamespace()
	testFSMGatewayHTTPSTraffic()
	testFSMGatewayTLSTerminate()
	testFSMGatewayTLSPassthrough()

	testFSMGatewayGRPCTrafficSameNamespace()
	testFSMGatewayGRPCTrafficCrossNamespace()
	testFSMGatewayGRPCSTraffic()

	testFSMGatewayTCPTrafficSameNamespace()
	testFSMGatewayTCPTrafficCrossNamespace()
	testFSMGatewayUDPTrafficSameNamespace()
	testFSMGatewayUDPTrafficCrossNamespace()
	testFSMGatewayDNSTraffic()
	testFSMGatewayDNSModifierFilterTraffic()
}

func testDeployFSMGateway() {
	// Create namespaces
	Expect(Td.CreateNs(nsGateway, nil)).To(Succeed())
	Expect(Td.CreateNs(nsHTTPRoute, map[string]string{"app": "http-cross"})).To(Succeed())
	Expect(Td.CreateNs(nsHTTPSvc, nil)).To(Succeed())
	Expect(Td.CreateNs(nsGRPCRoute, map[string]string{"app": "grpc-cross"})).To(Succeed())
	Expect(Td.CreateNs(nsGRPCSvc, nil)).To(Succeed())
	Expect(Td.CreateNs(nsTCPRoute, map[string]string{"app": "tcp-cross"})).To(Succeed())
	Expect(Td.CreateNs(nsTCPSvc, nil)).To(Succeed())
	Expect(Td.CreateNs(nsUDPRoute, map[string]string{"app": "udp-cross"})).To(Succeed())
	Expect(Td.CreateNs(nsUDPSvc, nil)).To(Succeed())

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
	stdout, stderr, err = Td.RunLocal("kubectl", "-n", nsGateway, "create", "secret", "tls", "https-cert", "--key", "https.key", "--cert", "https.crt")
	Td.T.Log(stdout.String())
	if stderr != nil {
		Td.T.Log("stderr:\n" + stderr.String())
	}
	Expect(err).NotTo(HaveOccurred())

	By("Creating ConfigMap for HTTPS CA Certificate")
	stdout, stderr, err = Td.RunLocal("kubectl", "-n", nsGateway, "create", "cm", "https-ca", "--from-file=ca.crt=./ca.crt")
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
	stdout, stderr, err = Td.RunLocal("kubectl", "-n", nsGRPCSvc, "create", "secret", "tls", "grpc-cert", "--key", "grpc.key", "--cert", "grpc.crt")
	Td.T.Log(stdout.String())
	if stderr != nil {
		Td.T.Log("stderr:\n" + stderr.String())
	}
	Expect(err).NotTo(HaveOccurred())

	By("Creating ConfigMap for configuring Gateway")
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "gateway-config",
		},
		Data: map[string]string{
			"values.yaml": `
fsm:
  gateway:
    replicas: 2
    resources:
      requests:
        cpu: 123m
        memory: 257Mi
      limits:
        cpu: 1314m
        memory: 2048Mi
`,
		},
	}
	_, err = Td.CreateConfigMap(nsGateway, cm)
	Expect(err).NotTo(HaveOccurred())

	By("Creating ReferenceGrant for testing Secret reference cross namespace")
	referenceGrant := gwv1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGRPCSvc,
			Name:      "secret-cross-1",
		},

		Spec: gwv1beta1.ReferenceGrantSpec{
			From: []gwv1beta1.ReferenceGrantFrom{
				{Group: gwv1.GroupName, Kind: "Gateway", Namespace: nsGateway},
			},
			To: []gwv1beta1.ReferenceGrantTo{
				{Group: corev1.GroupName, Kind: "Secret", Name: ptr.To(gwv1.ObjectName("grpc-cert"))},
			},
		},
	}
	_, err = Td.CreateGatewayAPIReferenceGrant(nsGRPCSvc, referenceGrant)
	Expect(err).NotTo(HaveOccurred())

	By("Deploy Gateway")
	gateway := gwv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "test-gw-1",
		},

		Spec: gwv1.GatewaySpec{
			GatewayClassName: "fsm",
			Listeners: []gwv1.Listener{
				{
					Port:     8090,
					Name:     "http",
					Protocol: gwv1.HTTPProtocolType,
				},
				{
					Port:     9090,
					Name:     "http-cross-ns",
					Protocol: gwv1.HTTPProtocolType,
					AllowedRoutes: &gwv1.AllowedRoutes{
						Namespaces: &gwv1.RouteNamespaces{
							From: ptr.To(gwv1.NamespacesFromAll),
						},
					},
				},
				{
					Port:     3000,
					Name:     "tcp",
					Protocol: gwv1.TCPProtocolType,
				},
				{
					Port:     3001,
					Name:     "tcp-cross-ns",
					Protocol: gwv1.TCPProtocolType,
					AllowedRoutes: &gwv1.AllowedRoutes{
						Namespaces: &gwv1.RouteNamespaces{
							From: ptr.To(gwv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "tcp-cross"},
							},
						},
					},
				},
				{
					Port:     4000,
					Name:     "udp",
					Protocol: gwv1.UDPProtocolType,
				},
				{
					Port:     5053,
					Name:     "dns",
					Protocol: gwv1.UDPProtocolType,
					AllowedRoutes: &gwv1.AllowedRoutes{
						Namespaces: &gwv1.RouteNamespaces{
							From: ptr.To(gwv1.NamespacesFromAll),
						},
					},
				},
				{
					Port:     4001,
					Name:     "udp-cross-ns",
					Protocol: gwv1.UDPProtocolType,
					AllowedRoutes: &gwv1.AllowedRoutes{
						Namespaces: &gwv1.RouteNamespaces{
							From: ptr.To(gwv1.NamespacesFromAll),
						},
					},
				},
				{
					Port:     7443,
					Name:     "https",
					Protocol: gwv1.HTTPSProtocolType,
					TLS: &gwv1.GatewayTLSConfig{
						CertificateRefs: []gwv1.SecretObjectReference{
							{
								Group: ptr.To(gwv1.Group(corev1.GroupName)),
								Kind:  ptr.To(gwv1.Kind("Secret")),
								Name:  "https-cert",
							},
							{
								Group:     ptr.To(gwv1.Group(corev1.GroupName)),
								Kind:      ptr.To(gwv1.Kind("Secret")),
								Namespace: namespacePtr(nsGRPCSvc),
								Name:      "grpc-cert",
							},
						},
						//FrontendValidation: &gwv1.FrontendTLSValidation{
						//	CACertificateRefs: []gwv1.ObjectReference{
						//		{Group: corev1.GroupName, Kind: "ConfigMap", Name: "https-ca"},
						//	},
						//},
						Options: map[gwv1.AnnotationKey]gwv1.AnnotationValue{},
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
								Group: ptr.To(gwv1.Group(corev1.GroupName)),
								Kind:  ptr.To(gwv1.Kind("Secret")),
								Name:  "https-cert",
							},
							{
								Group:     ptr.To(gwv1.Group(corev1.GroupName)),
								Kind:      ptr.To(gwv1.Kind("Secret")),
								Namespace: namespacePtr(nsGRPCSvc),
								Name:      "grpc-cert",
							},
						},
						//FrontendValidation: &gwv1.FrontendTLSValidation{
						//	CACertificateRefs: []gwv1.ObjectReference{
						//		{Group: corev1.GroupName, Kind: "ConfigMap", Name: "https-ca"},
						//	},
						//},
						Options: map[gwv1.AnnotationKey]gwv1.AnnotationValue{},
					},
				},
			},
			Infrastructure: &gwv1.GatewayInfrastructure{
				Annotations: map[gwv1.AnnotationKey]gwv1.AnnotationValue{"xyz": "abc"},
				Labels:      map[gwv1.LabelKey]gwv1.LabelValue{"test": "demo"},
				ParametersRef: &gwv1.LocalParametersReference{
					Group: corev1.GroupName,
					Kind:  constants.KubernetesConfigMapKind,
					Name:  "gateway-config",
				},
			},
		},
	}
	_, err = Td.CreateGateway(nsGateway, gateway)
	Expect(err).NotTo(HaveOccurred())
	// Expect it to be up and running in default namespace
	Expect(Td.WaitForPodsRunningReady(nsGateway, 2, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: constants.FSMGatewayName},
	})).To(Succeed())
}

func testFSMGatewayHTTPTrafficSameNamespace() {
	By("Deploying app for testing HTTP traffic in the same namespace")
	httpbinDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "httpbin",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "httpbin"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "httpbin"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pipy",
							Image: "flomesh/pipy:1.5.9",
							Ports: []corev1.ContainerPort{
								{
									Name:          "pipy",
									ContainerPort: 8080,
								},
							},
							Command:    []string{"pipy", "-e", "pipy().listen(8080).serveHTTP(new Message('Hi, I am HTTPRoute!'))"},
							WorkingDir: "/tmp",
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsGateway, httpbinDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsGateway, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "httpbin"},
	})).To(Succeed())

	By("Creating svc for HTTPRoute in the same namespace")
	httpbinSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
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
			Selector: map[string]string{"app": "httpbin"},
		},
	}
	_, err = Td.CreateService(nsGateway, httpbinSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating HTTPRoute for testing HTTP protocol in the same namespace")
	httpRoute := gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "http-app-1",
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(8090),
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
	_, err = Td.CreateGatewayAPIHTTPRoute(nsGateway, httpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing HTTPRoute - same namespace")
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

func testFSMGatewayHTTPTrafficCrossNamespace() {
	By("Deploying app for testing HTTP traffic cross namespace")
	httpbinDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsHTTPSvc,
			Name:      "httpbin-cross",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "httpbin-cross"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "httpbin-cross"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "pipy",
							Image: "flomesh/pipy:1.5.9",
							Ports: []corev1.ContainerPort{
								{
									Name:          "pipy",
									ContainerPort: 8080,
								},
							},
							Command:    []string{"pipy", "-e", "pipy().listen(8080).serveHTTP(new Message('Hi, I am HTTPRoute!'))"},
							WorkingDir: "/tmp",
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsHTTPSvc, httpbinDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsHTTPSvc, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "httpbin-cross"},
	})).To(Succeed())

	By("Creating svc for HTTPRoute cross namespace")
	httpbinSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsHTTPSvc,
			Name:      "httpbin-cross",
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
			Selector: map[string]string{"app": "httpbin-cross"},
		},
	}
	_, err = Td.CreateService(nsHTTPSvc, httpbinSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating HTTPRoute for testing HTTP protocol cross namespace")
	httpRoute := gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsHTTPRoute,
			Name:      "http-app-cross-1",
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Namespace: namespacePtr(nsGateway),
						Name:      "test-gw-1",
						Port:      portPtr(9090),
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
								Value: pointer.String("/cross"),
							},
						},
					},
					BackendRefs: []gwv1.HTTPBackendRef{
						{
							BackendRef: gwv1.BackendRef{
								BackendObjectReference: gwv1.BackendObjectReference{
									Namespace: namespacePtr(nsHTTPSvc),
									Name:      "httpbin-cross",
									Port:      portPtr(8080),
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPIHTTPRoute(nsHTTPRoute, httpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Creating ReferenceGrant for testing HTTP traffic cross namespace")
	referenceGrant := gwv1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsHTTPSvc,
			Name:      "http-cross-1",
		},

		Spec: gwv1beta1.ReferenceGrantSpec{
			From: []gwv1beta1.ReferenceGrantFrom{
				{Group: gwv1.GroupName, Kind: "HTTPRoute", Namespace: nsHTTPRoute},
			},
			To: []gwv1beta1.ReferenceGrantTo{
				{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gwv1.ObjectName("httpbin-cross"))},
			},
		},
	}
	_, err = Td.CreateGatewayAPIReferenceGrant(nsHTTPSvc, referenceGrant)
	Expect(err).NotTo(HaveOccurred())

	By("Testing HTTPRoute - cross namespace")
	httpReq := HTTPRequestDef{
		Destination: "http://httptest.localhost:9090/cross",
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

func testFSMGatewayGRPCTrafficSameNamespace() {
	By("Deploying app for testing gRPC traffic in the same namespace")
	grpcDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
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

	_, err := Td.CreateDeployment(nsGateway, grpcDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsGateway, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "grpcbin"},
	})).To(Succeed())

	By("Creating svc for GRPCRoute in the same namespace")
	grpcbinSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
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
	_, err = Td.CreateService(nsGateway, grpcbinSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating GRPCRoute for testing GRPC protocol in the same namespace")
	grpcRoute := gwv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "grpc-app-1",
		},
		Spec: gwv1.GRPCRouteSpec{
			CommonRouteSpec: gwv1alpha2.CommonRouteSpec{
				ParentRefs: []gwv1alpha2.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(8090),
					},
				},
			},
			Hostnames: []gwv1alpha2.Hostname{"grpctest.localhost"},
			Rules: []gwv1.GRPCRouteRule{
				{
					Matches: []gwv1.GRPCRouteMatch{
						{
							Method: &gwv1.GRPCMethodMatch{
								Type:    grpcMethodMatchTypePtr(gwv1.GRPCMethodMatchExact),
								Service: pointer.String("hello.HelloService"),
								Method:  pointer.String("SayHello"),
							},
						},
					},
					BackendRefs: []gwv1.GRPCBackendRef{
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
	_, err = Td.CreateGatewayAPIGRPCRoute(nsGateway, grpcRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing GRPCRoute - same namespace")
	grpcReq := GRPCRequestDef{
		Destination: "grpctest.localhost:8090",
		Symbol:      "hello.HelloService/SayHello",
		JSONRequest: `{"greeting":"Flomesh"}`,
		ProtoFile:   "../../scripts/hello.proto",
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

func testFSMGatewayGRPCTrafficCrossNamespace() {
	By("Deploying app for testing gRPC traffic cross namespace")
	grpcDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGRPCSvc,
			Name:      "grpcbin-cross",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "grpcbin-cross"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "grpcbin-cross"},
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

	_, err := Td.CreateDeployment(nsGRPCSvc, grpcDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsGRPCSvc, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "grpcbin-cross"},
	})).To(Succeed())

	By("Creating svc for GRPCRoute cross namespace")
	grpcbinSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGRPCSvc,
			Name:      "grpcbin-cross",
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
			Selector: map[string]string{"app": "grpcbin-cross"},
		},
	}
	_, err = Td.CreateService(nsGRPCSvc, grpcbinSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating GRPCRoute for testing GRPC protocol cross namespace")
	grpcRoute := gwv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGRPCRoute,
			Name:      "grpc-cross-1",
		},
		Spec: gwv1.GRPCRouteSpec{
			CommonRouteSpec: gwv1alpha2.CommonRouteSpec{
				ParentRefs: []gwv1alpha2.ParentReference{
					{
						Namespace: namespacePtr(nsGateway),
						Name:      "test-gw-1",
						Port:      portPtr(9090),
					},
				},
			},
			Hostnames: []gwv1alpha2.Hostname{"grpctest.localhost"},
			Rules: []gwv1.GRPCRouteRule{
				{
					Matches: []gwv1.GRPCRouteMatch{
						{
							Method: &gwv1.GRPCMethodMatch{
								Type:    grpcMethodMatchTypePtr(gwv1.GRPCMethodMatchExact),
								Service: pointer.String("hello.HelloService"),
								Method:  pointer.String("SayHello"),
							},
						},
					},
					BackendRefs: []gwv1.GRPCBackendRef{
						{
							BackendRef: gwv1alpha2.BackendRef{
								BackendObjectReference: gwv1alpha2.BackendObjectReference{
									Namespace: namespacePtr(nsGRPCSvc),
									Name:      "grpcbin-cross",
									Port:      portPtr(9000),
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPIGRPCRoute(nsGRPCRoute, grpcRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Creating ReferenceGrant for testing GRPC traffic cross namespace")
	referenceGrant := gwv1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGRPCSvc,
			Name:      "grpc-cross-1",
		},

		Spec: gwv1beta1.ReferenceGrantSpec{
			From: []gwv1beta1.ReferenceGrantFrom{
				{Group: gwv1.GroupName, Kind: "GRPCRoute", Namespace: nsGRPCRoute},
			},
			To: []gwv1beta1.ReferenceGrantTo{
				{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gwv1.ObjectName("grpcbin-cross"))},
			},
		},
	}
	_, err = Td.CreateGatewayAPIReferenceGrant(nsGRPCSvc, referenceGrant)
	Expect(err).NotTo(HaveOccurred())

	By("Testing GRPCRoute - cross namespace")
	grpcReq := GRPCRequestDef{
		Destination: "grpctest.localhost:9090",
		Symbol:      "hello.HelloService/SayHello",
		JSONRequest: `{"greeting":"Flomesh"}`,
		ProtoFile:   "../../scripts/hello.proto",
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

func testFSMGatewayTCPTrafficSameNamespace() {
	By("Deploying app for testing TCP traffic in the same namespace")
	tcpDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
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

	_, err := Td.CreateDeployment(nsGateway, tcpDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsGateway, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "tcp-echo"},
	})).To(Succeed())

	By("Creating svc for TCPRoute in the same namespace")
	tcpSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
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
	_, err = Td.CreateService(nsGateway, tcpSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating TCPRoute for testing TCP protocol in the same namespace")
	tcpRoute := gwv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "tcp-app-1",
		},
		Spec: gwv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(3000),
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
	_, err = Td.CreateGatewayAPITCPRoute(nsGateway, tcpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing TCPRoute - same namespace")
	tcpReq := TCPRequestDef{
		DestinationHost: "tcptest.localhost",
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

func testFSMGatewayTCPTrafficCrossNamespace() {
	By("Deploying app for testing TCP traffic cross namespace")
	tcpDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTCPSvc,
			Name:      "tcp-echo-cross",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "tcp-echo-cross"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "tcp-echo-cross"},
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

	_, err := Td.CreateDeployment(nsTCPSvc, tcpDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsTCPSvc, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "tcp-echo-cross"},
	})).To(Succeed())

	By("Creating svc for TCPRoute cross namespace")
	tcpSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTCPSvc,
			Name:      "tcp-echo-cross",
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
			Selector: map[string]string{constants.AppLabel: "tcp-echo-cross"},
		},
	}
	_, err = Td.CreateService(nsTCPSvc, tcpSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating TCPRoute for testing TCP protocol cross namespace")
	tcpRoute := gwv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTCPRoute,
			Name:      "tcp-cross-1",
		},
		Spec: gwv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Namespace: namespacePtr(nsGateway),
						Name:      "test-gw-1",
						Port:      portPtr(3001),
					},
				},
			},
			Rules: []gwv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Namespace: namespacePtr(nsTCPSvc),
								Name:      "tcp-echo-cross",
								Port:      portPtr(8078),
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPITCPRoute(nsTCPRoute, tcpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Creating ReferenceGrant for testing TCP traffic cross namespace")
	referenceGrant := gwv1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsTCPSvc,
			Name:      "tcp-cross-1",
		},

		Spec: gwv1beta1.ReferenceGrantSpec{
			From: []gwv1beta1.ReferenceGrantFrom{
				{Group: gwv1.GroupName, Kind: "TCPRoute", Namespace: nsTCPRoute},
			},
			To: []gwv1beta1.ReferenceGrantTo{
				{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gwv1.ObjectName("tcp-echo-cross"))},
			},
		},
	}
	_, err = Td.CreateGatewayAPIReferenceGrant(nsTCPSvc, referenceGrant)
	Expect(err).NotTo(HaveOccurred())

	By("Testing TCPRoute - cross namespace")
	tcpReq := TCPRequestDef{
		DestinationHost: "tcptest.localhost",
		DestinationPort: 3001,
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

func testFSMGatewayUDPTrafficSameNamespace() {
	By("Deploying app for testing UDP traffic in the same namespace")
	udpDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "udp-echo",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "udp-echo"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "udp-echo"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "udp",
							Image: "istio/fortio:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "udp",
									ContainerPort: 8078,
								},
							},
							Command: []string{"fortio", "udp-echo"},
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsGateway, udpDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsGateway, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "udp-echo"},
	})).To(Succeed())

	By("Creating svc for UDPRoute")
	udpSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "udp-echo",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "udp",
					Protocol:   corev1.ProtocolUDP,
					Port:       8078,
					TargetPort: intstr.FromInt32(8078),
				},
			},
			Selector: map[string]string{constants.AppLabel: "udp-echo"},
		},
	}
	_, err = Td.CreateService(nsGateway, udpSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating UDPRoute for testing UDP protocol in the same namespace")
	udpRoute := gwv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "udp-app-1",
		},
		Spec: gwv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(4000),
					},
				},
			},
			Rules: []gwv1alpha2.UDPRouteRule{
				{
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Name: "udp-echo",
								Port: portPtr(8078),
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPIUDPRoute(nsGateway, udpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing UDPRoute - same namespace")
	udpReq := UDPRequestDef{
		DestinationHost: "udptest.localhost",
		DestinationPort: 4000,
		Message:         "Hi, I am UDP!",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s:%d", "client", udpReq.DestinationHost, udpReq.DestinationPort)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalUDPRequest(udpReq)

		if result.Err != nil {
			Td.T.Logf("> (%s) UDP req failed, response: %s, err: %s", srcToDestStr, result.Response, result.Err)
			return false
		}

		Td.T.Logf("> (%s) UDP req succeeded, response: %s", srcToDestStr, result.Response)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing UDP traffic from echo/nc(localhost) to destination %s:%d", udpReq.DestinationHost, udpReq.DestinationPort)
}

func testFSMGatewayDNSTraffic() {
	By("Creating UDPRoute for exposing DNS service")
	udpRoute := gwv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsKubeSystem,
			Name:      "udp-dns-1",
		},
		Spec: gwv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Namespace: namespacePtr(nsGateway),
						Name:      "test-gw-1",
						Port:      portPtr(5053),
					},
				},
			},
			Rules: []gwv1alpha2.UDPRouteRule{
				{
					Name: ptr.To(gwv1.SectionName("dns")),
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Name: "kube-dns",
								Port: portPtr(53),
							},
						},
					},
				},
			},
		},
	}
	_, err := Td.CreateGatewayAPIUDPRoute(nsKubeSystem, udpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing UDPRoute - DIG DNS query")
	dnsReq := DNSRequestDef{
		DNSServer: "udptest.localhost",
		DNSPort:   5053,
		QueryHost: "kubernetes.default.svc.cluster.local",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s:%d", "client", dnsReq.DNSServer, dnsReq.DNSPort)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalDIGDNSRequest(dnsReq)

		response := strings.TrimSpace(result.Response)
		if result.Err != nil {
			Td.T.Logf("> (%s) DNS req failed, response: %s, err: %s", srcToDestStr, response, result.Err)
			return false
		}

		if response == "10.43.0.1" {
			Td.T.Logf("> (%s) DNS req succeeded, response: %s", srcToDestStr, response)
			return true
		}

		Td.T.Logf("> (%s) DNS req failed, expect: 10.43.0.1, response: %q", srcToDestStr, response)
		return false
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing DNS traffic from dig(localhost) to destination %s:%d", dnsReq.DNSServer, dnsReq.DNSPort)
}

func testFSMGatewayDNSModifierFilterTraffic() {
	By("Creating DNSModifier configuration for modifying DNS response")
	dnsModifier := extv1alpha1.DNSModifier{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsKubeSystem,
			Name:      "dns-mod-1",
		},

		Spec: extv1alpha1.DNSModifierSpec{
			Domains: []extv1alpha1.DNSDomain{
				{
					Name: gwv1.PreciseHostname("test.flomesh.io"),
					Answer: extv1alpha1.DNSAnswer{
						RData: "1.11.11.111",
					},
				},
			},
		},
	}

	_, err := Td.CreateGatewayAPIDNSModifier(nsKubeSystem, dnsModifier)
	Expect(err).NotTo(HaveOccurred())

	By("Creating Filter for modifying DNS response")
	filter := extv1alpha1.Filter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsKubeSystem,
			Name:      "test-dns-1",
		},

		Spec: extv1alpha1.FilterSpec{
			Type: "DNSModifier",
			ConfigRef: &gwv1.LocalObjectReference{
				Group: extv1alpha1.GroupName,
				Kind:  "DNSModifier",
				Name:  "dns-mod-1",
			},
		},
	}

	_, err = Td.CreateGatewayAPIFilter(nsKubeSystem, filter)
	Expect(err).NotTo(HaveOccurred())

	By("Creating RouteRuleFilterPolicy for attaching Filter to RouteRule")

	policy := gwpav1alpha2.RouteRuleFilterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsKubeSystem,
			Name:      "dns-policy-1",
		},

		Spec: gwpav1alpha2.RouteRuleFilterPolicySpec{
			TargetRefs: []gwpav1alpha2.LocalFilterPolicyTargetReference{
				{
					Group: gwv1.GroupName,
					Kind:  constants.GatewayAPIUDPRouteKind,
					Name:  "udp-dns-1",
					Rule:  "dns",
				},
			},
			FilterRefs: []gwpav1alpha2.LocalFilterReference{
				{
					Group: extv1alpha1.GroupName,
					Kind:  constants.GatewayAPIExtensionFilterKind,
					Name:  "test-dns-1",
				},
			},
		},
	}

	_, err = Td.CreateGatewayAPIRouteRuleFilterPolicy(nsKubeSystem, policy)
	Expect(err).NotTo(HaveOccurred())

	By("Testing UDPRoute - DIG DNS query")
	dnsReq := DNSRequestDef{
		DNSServer: "udptest.localhost",
		DNSPort:   5053,
		QueryHost: "test.flomesh.io",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s:%d", "client", dnsReq.DNSServer, dnsReq.DNSPort)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalDIGDNSRequest(dnsReq)

		response := strings.TrimSpace(result.Response)
		if result.Err != nil {
			Td.T.Logf("> (%s) DNS req failed, response: %s, err: %s", srcToDestStr, response, result.Err)
			return false
		}

		if response == "1.11.11.111" {
			Td.T.Logf("> (%s) DNS req succeeded, response: %s", srcToDestStr, response)
			return true
		}

		Td.T.Logf("> (%s) DNS req failed, expect: 1.11.11.111, response: %q", srcToDestStr, response)
		return false
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing DNS traffic from dig(localhost) to destination %s:%d", dnsReq.DNSServer, dnsReq.DNSPort)
}

func testFSMGatewayUDPTrafficCrossNamespace() {
	By("Deploying app for testing UDP traffic cross namespace")
	udpDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsUDPSvc,
			Name:      "udp-echo-cross",
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{constants.AppLabel: "udp-echo-cross"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{constants.AppLabel: "udp-echo-cross"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "udp",
							Image: "istio/fortio:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "udp",
									ContainerPort: 8078,
								},
							},
							Command: []string{"fortio", "udp-echo"},
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsUDPSvc, udpDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsUDPSvc, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "udp-echo-cross"},
	})).To(Succeed())

	By("Creating svc for UDPRoute")
	udpSvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsUDPSvc,
			Name:      "udp-echo-cross",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "udp",
					Protocol:   corev1.ProtocolUDP,
					Port:       8078,
					TargetPort: intstr.FromInt32(8078),
				},
			},
			Selector: map[string]string{constants.AppLabel: "udp-echo-cross"},
		},
	}
	_, err = Td.CreateService(nsUDPSvc, udpSvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating UDPRoute for testing UDP protocol cross namespace")
	udpRoute := gwv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsUDPRoute,
			Name:      "udp-cross-1",
		},
		Spec: gwv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Namespace: namespacePtr(nsGateway),
						Name:      "test-gw-1",
						Port:      portPtr(4001),
					},
				},
			},
			Rules: []gwv1alpha2.UDPRouteRule{
				{
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Namespace: namespacePtr(nsUDPSvc),
								Name:      "udp-echo-cross",
								Port:      portPtr(8078),
							},
						},
					},
				},
			},
		},
	}
	_, err = Td.CreateGatewayAPIUDPRoute(nsUDPRoute, udpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Creating ReferenceGrant for testing UDP traffic cross namespace")
	referenceGrant := gwv1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsUDPSvc,
			Name:      "udp-cross-1",
		},

		Spec: gwv1beta1.ReferenceGrantSpec{
			From: []gwv1beta1.ReferenceGrantFrom{
				{Group: gwv1.GroupName, Kind: "UDPRoute", Namespace: nsUDPRoute},
			},
			To: []gwv1beta1.ReferenceGrantTo{
				{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gwv1.ObjectName("udp-echo-cross"))},
			},
		},
	}
	_, err = Td.CreateGatewayAPIReferenceGrant(nsUDPSvc, referenceGrant)
	Expect(err).NotTo(HaveOccurred())

	By("Testing UDPRoute - cross namespace")
	udpReq := UDPRequestDef{
		DestinationHost: "udptest.localhost",
		DestinationPort: 4001,
		Message:         "Hi, I am UDP!",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s:%d", "client", udpReq.DestinationHost, udpReq.DestinationPort)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalUDPRequest(udpReq)

		if result.Err != nil {
			Td.T.Logf("> (%s) UDP req failed, response: %s, err: %s", srcToDestStr, result.Response, result.Err)
			return false
		}

		Td.T.Logf("> (%s) UDP req succeeded, response: %s", srcToDestStr, result.Response)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing UDP traffic from echo/nc(localhost) to destination %s:%d", udpReq.DestinationHost, udpReq.DestinationPort)
}

func testFSMGatewayHTTPSTraffic() {
	By("Creating HTTPRoute for testing HTTPs protocol")
	httpRoute := gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "https-app-1",
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(7443),
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
	_, err := Td.CreateGatewayAPIHTTPRoute(nsGateway, httpRoute)
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

func testFSMGatewayGRPCSTraffic() {
	By("Creating GRPCRoute for testing GRPCs protocol")
	grpcRoute := gwv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "grpcs-app-1",
		},
		Spec: gwv1.GRPCRouteSpec{
			CommonRouteSpec: gwv1alpha2.CommonRouteSpec{
				ParentRefs: []gwv1alpha2.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(7443),
					},
				},
			},
			Hostnames: []gwv1alpha2.Hostname{"grpctest.localhost"},
			Rules: []gwv1.GRPCRouteRule{
				{
					Matches: []gwv1.GRPCRouteMatch{
						{
							Method: &gwv1.GRPCMethodMatch{
								Type:    grpcMethodMatchTypePtr(gwv1.GRPCMethodMatchExact),
								Service: pointer.String("hello.HelloService"),
								Method:  pointer.String("SayHello"),
							},
						},
					},
					BackendRefs: []gwv1.GRPCBackendRef{
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
	_, err := Td.CreateGatewayAPIGRPCRoute(nsGateway, grpcRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing GRPCRoute(GRPCs)")
	grpcReq := GRPCRequestDef{
		Destination: "grpctest.localhost:7443",
		Symbol:      "hello.HelloService/SayHello",
		JSONRequest: `{"greeting":"Flomesh"}`,
		UseTLS:      true,
		CertFile:    "grpc.crt",
		ProtoFile:   "../../scripts/hello.proto",
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

func testFSMGatewayTLSPassthrough() {
	By("Creating TLSRoute for testing TLS passthrough")
	tlsRoute := gwv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "tlsp-app-1",
		},
		Spec: gwv1alpha2.TLSRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(8443),
					},
				},
			},
			Rules: []gwv1alpha2.TLSRouteRule{
				{
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Name: "bing.com",
								Port: portPtr(443),
							},
						},
					},
				},
			},
		},
	}
	_, err := Td.CreateGatewayAPITLSRoute(nsGateway, tlsRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing TLS Passthrough")
	httpsReq := HTTPRequestDef{
		Destination:      "https://bing.com",
		UseTLS:           true,
		IsTLSPassthrough: true,
		PassthroughHost:  "httptest.localhost",
		PassthroughPort:  8443,
	}
	srcToDestStr := fmt.Sprintf("%s -> %s", "curl", httpsReq.Destination)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalHTTPRequest(httpsReq)

		if result.Err != nil || result.StatusCode != 200 {
			Td.T.Logf("> (%s) TLS passthrough Req failed %d %v",
				srcToDestStr, result.StatusCode, result.Err)
			return false
		}
		Td.T.Logf("> (%s) TLS passthrough Req succeeded: %d", srcToDestStr, result.StatusCode)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing TLS passthrough traffic from curl(localhost) to destination %s", httpsReq.Destination)
}

func testFSMGatewayTLSTerminate() {
	By("Creating TCPRoute for testing TLS terminate")
	tcpRoute := gwv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsGateway,
			Name:      "tlst-app-1",
		},
		Spec: gwv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: "test-gw-1",
						Port: portPtr(9443),
					},
				},
			},
			Rules: []gwv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gwv1alpha2.BackendRef{
						{
							BackendObjectReference: gwv1.BackendObjectReference{
								Name: "httpbin",
								Port: portPtr(8080),
							},
						},
					},
				},
			},
		},
	}
	_, err := Td.CreateGatewayAPITCPRoute(nsGateway, tcpRoute)
	Expect(err).NotTo(HaveOccurred())

	By("Testing TLS Terminate")
	httpsReq := HTTPRequestDef{
		Destination: "https://httptest.localhost:9443",
		UseTLS:      true,
		CertFile:    "https.crt",
	}
	srcToDestStr := fmt.Sprintf("%s -> %s", "curl", httpsReq.Destination)

	cond := Td.WaitForRepeatedSuccess(func() bool {
		result := Td.LocalHTTPRequest(httpsReq)

		if result.Err != nil || result.StatusCode != 200 {
			Td.T.Logf("> (%s) TLS Req failed %d %v",
				srcToDestStr, result.StatusCode, result.Err)
			return false
		}
		Td.T.Logf("> (%s) TLS Req succeeded: %d", srcToDestStr, result.StatusCode)
		return true
	}, 5, Td.ReqSuccessTimeout)

	Expect(cond).To(BeTrue(), "Failed testing TLS traffic from curl(localhost) to destination %s", httpsReq.Destination)
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

func grpcMethodMatchTypePtr(grpcMatch gwv1.GRPCMethodMatchType) *gwv1.GRPCMethodMatchType {
	ret := grpcMatch
	return &ret
}
