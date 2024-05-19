package e2e

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test deployment of Fluent Bit sidecar",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("Fluent Bit deployment", func() {
			It("Deploys a Fluent Bit sidecar only when enabled", func() {
				if Td.DeployOnOpenShift {
					Skip("Skipping test: FluentBit not supported on OpenShift")
				}
				// Install FSM with Fluentbit
				installOpts := Td.GetFSMInstallOpts()
				installOpts.DeployFluentbit = true
				Expect(Td.InstallFSM(installOpts)).To(Succeed())

				pods, err := Td.Client.CoreV1().Pods(Td.FsmNamespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{constants.AppLabel: constants.FSMControllerName}).String(),
				})

				Expect(err).NotTo(HaveOccurred())
				cond := false
				for _, pod := range pods.Items {
					for _, container := range pod.Spec.Containers {
						if strings.Contains(container.Image, "fluent-bit") {
							cond = true
						}
					}
				}
				Expect(cond).To(BeTrue())

				err = Td.DeleteNs(Td.FsmNamespace)
				Expect(err).NotTo(HaveOccurred())
				err = Td.WaitForNamespacesDeleted([]string{Td.FsmNamespace}, 60*time.Second)
				Expect(err).NotTo(HaveOccurred())

				// Install FSM without Fluentbit (default)
				installOpts = Td.GetFSMInstallOpts()
				Expect(Td.InstallFSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.FsmNamespace, 2 /* controller, injector */, nil)).To(Succeed())

				pods, err = Td.Client.CoreV1().Pods(Td.FsmNamespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{constants.AppLabel: constants.FSMControllerName}).String(),
				})

				Expect(err).NotTo(HaveOccurred())
				cond = false
				for _, pod := range pods.Items {
					for _, container := range pod.Spec.Containers {
						if strings.Contains(container.Image, "fluent-bit") {
							cond = true
						}
					}
				}
				Expect(cond).To(BeFalse())
			})
		})
	})
