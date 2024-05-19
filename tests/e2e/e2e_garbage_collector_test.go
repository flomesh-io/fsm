package e2e

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/flomesh-io/fsm/pkg/constants"
	. "github.com/flomesh-io/fsm/tests/framework"
)

var _ = FSMDescribe("Test garbage collection for unused sidecar bootstrap config secrets",
	FSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("Garbage Collection", func() {
			userService := "app"
			userReplicaSet := 1

			It("Tests garbage collection", func() {
				// Install FSM
				Expect(Td.InstallFSM(Td.GetFSMInstallOpts())).To(Succeed())

				// Create NSs
				Expect(Td.CreateNs(userService, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, userService)).To(Succeed())

				// User app
				svcAccDef, deploymentDef, svcDef, err := Td.SimpleDeploymentApp(
					SimpleDeploymentAppDef{
						DeploymentName: userService,
						Namespace:      userService,
						ReplicaCount:   int32(userReplicaSet),
						Command:        []string{"/bin/bash", "-c", "--"},
						Args:           []string{"while true; do sleep 30; done;"},
						Image:          "flomesh/alpine-debug",
						Ports:          []int{80},
						OS:             Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(userService, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateDeployment(userService, deploymentDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(userService, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(userService, userReplicaSet, nil)).To(Succeed())

				By("Verifying the secrets have been patched with OwnerReference")

				podSelector := constants.SidecarUniqueIDLabelName

				pods, err := Td.Client.CoreV1().Pods(userService).List(context.Background(), metav1.ListOptions{LabelSelector: podSelector})
				Expect(err).To(BeNil())

				for _, pod := range pods.Items {
					podUUID := pod.GetLabels()[podSelector]
					secretName := fmt.Sprintf("sidecar-bootstrap-config-%s", podUUID)
					secret, err := Td.Client.CoreV1().Secrets(userService).Get(context.Background(), secretName, metav1.GetOptions{})
					Expect(err).To(BeNil())

					ownerReferences := secret.GetOwnerReferences()
					Expect(ownerReferences).ToNot(BeNil())

					expectedOwnerReference := v1.OwnerReference{
						APIVersion: "v1",
						Kind:       "Pod",
						Name:       pod.GetName(),
						UID:        pod.GetUID(),
					}

					foundReference := false
					for _, ownerReference := range ownerReferences {
						if reflect.DeepEqual(expectedOwnerReference, ownerReference) {
							foundReference = true
						}
					}
					Expect(foundReference).To(BeTrue())
				}

				By("Verifying unused secrets are deleted when the referenced owner is deleted")

				pods, err = Td.Client.CoreV1().Pods(userService).List(context.Background(), metav1.ListOptions{LabelSelector: podSelector})
				Expect(err).To(BeNil())

				policy := metav1.DeletePropagationForeground
				err = Td.Client.CoreV1().Pods(userService).DeleteCollection(context.Background(), metav1.DeleteOptions{PropagationPolicy: &policy}, metav1.ListOptions{LabelSelector: podSelector})
				Expect(err).To(BeNil())

				Expect(Td.WaitForPodsDeleted(pods, userService, 200*time.Second)).To(Succeed())

				for _, pod := range pods.Items {
					podUUID := pod.GetLabels()[podSelector]
					secretName := fmt.Sprintf("sidecar-bootstrap-config-%s", podUUID)
					_, err := Td.Client.CoreV1().Secrets(userService).Get(context.Background(), secretName, metav1.GetOptions{})
					Expect(err).ToNot(BeNil())
				}
			})
		})
	})
