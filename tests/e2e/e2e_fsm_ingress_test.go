package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
)

const nsIngress = "test"

var _ = FSMDescribe("Test traffic among FSM Ingress",
	FSMDescribeInfo{
		Tier:   1,
		Bucket: 7,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Test traffic from client to backend service routing by FSM Ingress", func() {
			It("allow traffic through Ingress", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableIngress = true
				installOpts.EnableIngressTLS = true
				installOpts.EnableGateway = false
				installOpts.EnableServiceLB = true
				installOpts.IngressHTTPPort = 8090
				installOpts.IngressTLSPort = 9443

				Expect(Td.InstallFSM(installOpts)).To(Succeed())

				// Wait for FSM Ingress to be ready
				Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 1, &metav1.LabelSelector{
					MatchLabels: map[string]string{
						constants.AppLabel:                 constants.FSMIngressName,
						"networking.flomesh.io/namespaced": "false",
					},
				})).To(Succeed())

				// Create namespaces
				Expect(Td.CreateNs(nsIngress, nil)).To(Succeed())

				deployAppForTestingIngress()
				testIngressHTTPTraffic()
				testIngressTLSTraffic()
			})
		})
	})

func testIngressTLSTraffic() {
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
	stdout, stderr, err = Td.RunLocal("kubectl", "-n", nsIngress, "create", "secret", "generic", "https-cert", "--from-file=ca.crt=./ca.crt", "--from-file=tls.crt=./https.crt", "--from-file=tls.key=./https.key")
	Td.T.Log(stdout.String())
	if stderr != nil {
		Td.T.Log("stderr:\n" + stderr.String())
	}
	Expect(err).NotTo(HaveOccurred())

	// Create ingress
	By("Creating ingress rule for HTTPs")
	ingTLS := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsIngress,
			Name:      "pipy-tls",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: pointer.String("pipy"),
			Rules: []networkingv1.IngressRule{
				{
					Host: "httptest.localhost",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/ok-tls",
									PathType: func() *networkingv1.PathType {
										pt := networkingv1.PathTypePrefix
										return &pt
									}(),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "pipy",
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"httptest.localhost"},
					SecretName: "https-cert",
				},
			},
		},
	}

	_, err = Td.CreateIngress(nsIngress, ingTLS)
	Expect(err).NotTo(HaveOccurred())

	By("Testing HTTPS traffic")
	httpsReq := HTTPRequestDef{
		Destination: "https://httptest.localhost:9443/ok-tls",
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

func testIngressHTTPTraffic() {
	// Create ingress
	By("Creating ingress rule for HTTP")
	ing := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsIngress,
			Name:      "pipy",
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: pointer.String("pipy"),
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/ok",
									PathType: func() *networkingv1.PathType {
										pt := networkingv1.PathTypePrefix
										return &pt
									}(),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "pipy",
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateIngress(nsIngress, ing)
	Expect(err).NotTo(HaveOccurred())

	// Test http
	By("Testing HTTP traffic")
	httpReq := HTTPRequestDef{
		Destination: "http://httptest.localhost:8090/ok",
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

func deployAppForTestingIngress() {
	// Deploy test app
	By("Deploying app in namespace test")
	pipyDeploy := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsIngress,
			Name:      "pipy",
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
							Image: "flomesh/pipy:1.5.5",
							Ports: []corev1.ContainerPort{
								{
									Name:          "pipy",
									ContainerPort: 8080,
								},
							},
							Command:    []string{"pipy", "-e", "pipy().listen(8080).serveHTTP(new Message('Hi, I am PIPY!'))"},
							WorkingDir: "/tmp",
						},
					},
				},
			},
		},
	}

	_, err := Td.CreateDeployment(nsIngress, pipyDeploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsIngress, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{constants.AppLabel: "pipy"},
	})).To(Succeed())

	// Create svc
	By("Creating svc for pipy")
	pipySvc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsIngress,
			Name:      "pipy",
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
	_, err = Td.CreateService(nsIngress, pipySvc)
	Expect(err).NotTo(HaveOccurred())
}
