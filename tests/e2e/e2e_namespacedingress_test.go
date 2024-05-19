package e2e

import (
	nsigv1alpha1 "github.com/flomesh-io/fsm/pkg/apis/namespacedingress/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var _ = FSMDescribe("Test traffic among FSM NamespacedIngress",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 7,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Test traffic from client to backend service routing by FSM NamespacedIngress", func() {
			It("allow traffic through NamespacedIngress", func() {
				// Install FSM
				installOpts := Td.GetFSMInstallOpts()
				installOpts.EnableIngress = true
				installOpts.EnableNamespacedIngress = true
				installOpts.EnableGateway = false
				installOpts.EnableServiceLB = true

				Expect(Td.InstallFSM(installOpts)).To(Succeed())

				// Create namespaces
				Expect(Td.CreateNs(nsIngress, nil)).To(Succeed())

				deployNamespacedIngress()
				deployAppForTestingIngress()
				testIngressHTTPTraffic()
				testIngressTLSTraffic()
			})
		})
	})

func deployNamespacedIngress() {
	By("Creating NamespacedIngress")
	nsig := nsigv1alpha1.NamespacedIngress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsIngress,
			Name:      "test-namespacedingress",
		},
		Spec: nsigv1alpha1.NamespacedIngressSpec{
			ServiceType: corev1.ServiceTypeLoadBalancer,
			Replicas:    pointer.Int32(1),
			HTTP: nsigv1alpha1.HTTP{
				Enabled: true,
				Port: nsigv1alpha1.ServicePort{
					Name:     "http",
					Protocol: "TCP",
					Port:     8090,
				},
			},
			TLS: nsigv1alpha1.TLS{
				Enabled: true,
				Port: nsigv1alpha1.ServicePort{
					Name:     "https",
					Protocol: "TCP",
					Port:     9443,
				},
			},
		},
	}
	_, err := Td.CreateNamespacedIngress(nsIngress, nsig)
	Expect(err).NotTo(HaveOccurred())
	Expect(Td.WaitForPodsRunningReady(nsIngress, 1, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.AppLabel:                 constants.FSMIngressName,
			"networking.flomesh.io/namespaced": "true",
			"networking.flomesh.io/ns":         nsIngress,
		},
	})).To(Succeed())
}
