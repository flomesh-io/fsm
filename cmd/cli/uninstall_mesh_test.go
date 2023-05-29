package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsClientSetFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/injector"
	"github.com/flomesh-io/fsm/pkg/validator"
)

var (
	someOtherNamespace                = "someOtherNamespace"
	someOtherCustomResourceDefinition = "someOtherCustomResourceDefinition"
	someOtherWebhookName              = "someOtherWebhookName"
	someOtherSecretName               = "someOtherSecretName"
	fsmTestVersion                    = "testVersion"
	diffMesh                          = "diffMesh"
	diffNamespace                     = "diffNs"
)

func TestUninstallCmd(t *testing.T) {
	tests := []struct {
		name            string
		meshName        string
		meshNamespace   string
		force           bool
		deleteNamespace bool
		emptyMeshList   bool
		meshExists      bool
		userPromptsYes  bool
	}{
		{
			name:            "no meshes in cluster",
			meshName:        "",
			meshNamespace:   "",
			force:           false,
			deleteNamespace: false,
			userPromptsYes:  true,
			emptyMeshList:   true,
			meshExists:      false,
		},
		{
			name:            "the mesh DOES NOT exist",
			meshName:        diffMesh,
			meshNamespace:   diffNamespace,
			force:           false,
			deleteNamespace: false,
			userPromptsYes:  true,
			emptyMeshList:   false,
			meshExists:      false,
		},
		{
			name:            "the mesh DOES NOT exist and force delete set to true",
			meshName:        diffMesh,
			meshNamespace:   diffNamespace,
			force:           true,
			deleteNamespace: false,
			userPromptsYes:  true,
			emptyMeshList:   false,
			meshExists:      false,
		},
		{
			name:            "the mesh exists",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: false,
			userPromptsYes:  true,
			emptyMeshList:   false,
			meshExists:      true,
		},
		{
			name:            "the mesh exists and force set to true",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           true,
			deleteNamespace: false,
			userPromptsYes:  true,
			emptyMeshList:   false,
			meshExists:      true,
		},
		{
			name:            "the mesh exists, force set to true and delete namespace set to true",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           true,
			deleteNamespace: true,
			userPromptsYes:  true,
			emptyMeshList:   false,
			meshExists:      true,
		},
		{
			name:            "the mesh exists, force set to false, delete namespace set to true and user refuses mesh delete",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: true,
			meshExists:      true,
			emptyMeshList:   false,
			userPromptsYes:  true,
		},
		{
			name:            "the mesh exists, force set to false, delete namespace set to true and user approves mesh delete",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: true,
			emptyMeshList:   false,
			meshExists:      true,
			userPromptsYes:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			if test.userPromptsYes {
				in.Write([]byte("y\n"))
			} else {
				in.Write([]byte("n\n"))
			}

			var existingKubeClientsetObjects []runtime.Object
			existingNamespaces := []runtime.Object{
				// FSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: test.meshNamespace,
					},
				},
			}
			existingCustomResourceDefinitions := []runtime.Object{
				// FSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "egresses.policy.flomesh.io",
						Labels: map[string]string{
							constants.FSMAppNameLabelKey: constants.FSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
			}
			existingMutatingWebhookConfigurations := []runtime.Object{
				// FSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: injector.MutatingWebhookName,
						Labels: map[string]string{
							constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
							constants.FSMAppInstanceLabelKey: test.meshName,
							constants.AppLabel:               constants.FSMInjectorName,
						},
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
			}
			existingValidatingWebhookConfigurations := []runtime.Object{
				// FSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: validator.ValidatingWebhookName,
						Labels: map[string]string{
							constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
							constants.FSMAppInstanceLabelKey: test.meshName,
							constants.AppLabel:               constants.FSMControllerName,
						},
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
			}
			existingSecrets := []runtime.Object{
				// FSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.DefaultCABundleSecretName,
						Namespace: test.meshNamespace,
					},
				},
			}
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingNamespaces...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingMutatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingValidatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingSecrets...)

			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(test.meshNamespace)
			}

			if !test.emptyMeshList {
				rel := release.Mock(&release.MockReleaseOptions{Name: testMeshName})
				err := store.Create(rel)
				assert.Nil(err)
			}

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet := fake.NewSimpleClientset(existingKubeClientsetObjects...)

			if !test.emptyMeshList {
				_, err := addDeployment(fakeClientSet, "fsm-controller-1", testMeshName, testNamespace, fsmTestVersion, true)
				assert.Nil(err)
			}

			uninstall := uninstallMeshCmd{
				in:                  in,
				out:                 out,
				force:               test.force,
				client:              helm.NewUninstall(testConfig),
				meshName:            test.meshName,
				meshNamespace:       test.meshNamespace,
				caBundleSecretName:  constants.DefaultCABundleSecretName,
				clientSet:           fakeClientSet,
				extensionsClientset: extensionsClientSetFake.NewSimpleClientset(existingCustomResourceDefinitions...),
				deleteNamespace:     test.deleteNamespace,
				actionConfig:        new(action.Configuration),
			}

			err := uninstall.run()
			assert.Nil(err)

			if test.emptyMeshList {
				assert.Contains(out.String(), "No FSM control planes found\n")
			} else {
				if !test.force {
					if test.meshExists {
						assert.Contains(out.String(), fmt.Sprintf("\nUninstall FSM [mesh name: %s] in namespace [%s] and/or FSM resources? [y/n]: ", test.meshName, test.meshNamespace))
					} else {
						assert.Contains(out.String(), "List of meshes present in the cluster:\n")
						assert.Contains(out.String(), fmt.Sprintf("Did not find mesh [%s] in namespace [%s]\n", test.meshName, test.meshNamespace))
					}
				}

				if test.userPromptsYes {
					if test.meshExists {
						assert.Contains(out.String(), fmt.Sprintf("FSM [mesh name: %s] in namespace [%s] uninstalled\n", test.meshName, test.meshNamespace))
					}
					if test.deleteNamespace {
						assert.Contains(out.String(), fmt.Sprintf("FSM namespace [%s] deleted successfully\n", test.meshNamespace))
						namespaces, err := uninstall.clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
						assert.Nil(err)
						assert.Equal(0, len(namespaces.Items))
					} else {
						namespaces, err := uninstall.clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
						assert.Nil(err)
						assert.Equal(len(existingNamespaces), len(namespaces.Items))
					}
				}
			}

			// ensure that the FSM CRDs remain in the cluster
			crdsList, err := uninstall.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(crdsList.Items))
			assert.Equal("egresses.policy.flomesh.io", crdsList.Items[0].Name)

			// ensure that the FSM MutatingWebhookConfigurations remain in the cluster
			mutatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(mutatingWebhookConfigurations.Items))
			assert.Equal(injector.MutatingWebhookName, mutatingWebhookConfigurations.Items[0].Name)

			// ensure that FSM ValidatingWebhookConfigurations remain in the cluster
			validatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(validatingWebhookConfigurations.Items))
			assert.Equal(validator.ValidatingWebhookName, validatingWebhookConfigurations.Items[0].Name)

			// ensure that FSM Secrets remain in the cluster
			secrets, err := uninstall.clientSet.CoreV1().Secrets(uninstall.meshNamespace).List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(len(secrets.Items), 1)
			assert.Equal(constants.DefaultCABundleSecretName, secrets.Items[0].Name)
		})
	}
}

func TestUninstallClusterWideResources(t *testing.T) {
	tests := []struct {
		name                                    string
		existingNamespaces                      []runtime.Object
		existingCustomResourceDefinitions       []runtime.Object
		existingMutatingWebhookConfigurations   []runtime.Object
		existingValidatingWebhookConfigurations []runtime.Object
		existingSecrets                         []runtime.Object
	}{
		{
			name: "fsm/smi resources EXIST before run, should NOT ERROR, fsm/smi resources should BE DELETED, non-fsm/non-smi resources should NOT BE DELETED",
			existingNamespaces: []runtime.Object{
				// FSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
				// non-FSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherNamespace,
					},
				},
			},
			existingCustomResourceDefinitions: []runtime.Object{
				// FSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "egresses.policy.flomesh.io",
						Labels: map[string]string{
							constants.FSMAppNameLabelKey: constants.FSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// FSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingressbackends.policy.flomesh.io",
						Labels: map[string]string{
							constants.FSMAppNameLabelKey: constants.FSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "httproutegroups.specs.smi-spec.io",
						Labels: map[string]string{
							constants.FSMAppNameLabelKey: constants.FSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "tcproutes.specs.smi-spec.io",
						Labels: map[string]string{
							constants.FSMAppNameLabelKey: constants.FSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "trafficsplits.split.smi-spec.io",
						Labels: map[string]string{
							constants.FSMAppNameLabelKey: constants.FSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// non-FSM/non-SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherCustomResourceDefinition,
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
			},
			existingMutatingWebhookConfigurations: []runtime.Object{
				// FSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: injector.MutatingWebhookName,
						Labels: map[string]string{
							constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
							constants.FSMAppInstanceLabelKey: testMeshName,
							constants.AppLabel:               constants.FSMInjectorName,
						},
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
				// non-FSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
			},
			existingValidatingWebhookConfigurations: []runtime.Object{
				// FSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: validator.ValidatingWebhookName,
						Labels: map[string]string{
							constants.FSMAppNameLabelKey:     constants.FSMAppNameLabelValue,
							constants.FSMAppInstanceLabelKey: testMeshName,
							constants.AppLabel:               constants.FSMControllerName,
						},
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
				// non-FSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
			},
			existingSecrets: []runtime.Object{
				// FSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.DefaultCABundleSecretName,
						Namespace: testNamespace,
					},
				},
				// non-FSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      someOtherSecretName,
						Namespace: testNamespace,
					},
				},
			},
		},
		{
			name: "fsm/smi resources DO NOT EXIST before run, should NOT ERROR, non-fsm/non-smi resources should NOT BE DELETED",
			existingNamespaces: []runtime.Object{
				// non-FSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherNamespace,
					},
				},
			},
			existingCustomResourceDefinitions: []runtime.Object{
				// non-FSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherCustomResourceDefinition,
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
			},
			existingMutatingWebhookConfigurations: []runtime.Object{
				// non-FSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
			},
			existingValidatingWebhookConfigurations: []runtime.Object{
				// non-FSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
			},
			existingSecrets: []runtime.Object{
				// non-FSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      someOtherSecretName,
						Namespace: testNamespace,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			var existingKubeClientsetObjects []runtime.Object
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingNamespaces...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingMutatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingValidatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingSecrets...)

			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(testNamespace)
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: testMeshName})
			err := store.Create(rel)
			assert.Nil(err)

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet := fake.NewSimpleClientset(existingKubeClientsetObjects...)
			_, err = addDeployment(fakeClientSet, "fsm-controller-1", testMeshName, testNamespace, fsmTestVersion, true)
			assert.Nil(err)

			uninstall := uninstallMeshCmd{
				in:                 in,
				out:                out,
				force:              true,
				client:             helm.NewUninstall(testConfig),
				meshName:           testMeshName,
				meshNamespace:      testNamespace,
				caBundleSecretName: constants.DefaultCABundleSecretName,
				clientSet:          fakeClientSet,
				// CustomResourceDefinitions belong to the extensionsClientset
				extensionsClientset:        extensionsClientSetFake.NewSimpleClientset(test.existingCustomResourceDefinitions...),
				deleteClusterWideResources: true,
				actionConfig:               new(action.Configuration),
			}

			err = uninstall.run()
			assert.Nil(err)

			// ensure that only the non-FSM CRDs remain in the cluster
			crdsList, err := uninstall.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(crdsList.Items))
			assert.Equal(someOtherCustomResourceDefinition, crdsList.Items[0].Name)

			// ensure that only the non-FSM MutatingWebhookConfigurations remain in the cluster
			mutatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(mutatingWebhookConfigurations.Items))
			assert.Equal(someOtherWebhookName, mutatingWebhookConfigurations.Items[0].Name)

			// ensure that only the non-FSM ValidatingWebhookConfigurations remain in the cluster
			validatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(validatingWebhookConfigurations.Items))
			assert.Equal(someOtherWebhookName, validatingWebhookConfigurations.Items[0].Name)

			// ensure that only the non-FSM Secrets remain in the cluster
			secrets, err := uninstall.clientSet.CoreV1().Secrets(uninstall.meshNamespace).List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(secrets.Items))
			assert.Equal(someOtherSecretName, secrets.Items[0].Name)

			// ensure that existing namespaces are not deleted as deleting namespaces could be disastrous (for example, if FSM was installed in namespace kube-system)
			namespaces, err := uninstall.clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(len(test.existingNamespaces), len(namespaces.Items))
		})
	}
}

func TestFindSpecifiedMesh(t *testing.T) {
	tests := []struct {
		name                  string
		specifiedMesh         string
		meshList              []meshInfo
		expSpecifiedMeshFound bool
	}{
		{
			name:          "mesh flag not specified",
			specifiedMesh: "",
			meshList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			expSpecifiedMeshFound: false,
		},
		{
			name:          "mesh flag specified and not in mesh list",
			specifiedMesh: "notInMesh",
			meshList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			expSpecifiedMeshFound: false,
		},
		{
			name:          "mesh flag specified and in mesh list",
			specifiedMesh: testMeshName,
			meshList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			expSpecifiedMeshFound: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(testNamespace)
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: testMeshName})
			err := store.Create(rel)
			assert.Nil(err)

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet := fake.NewSimpleClientset()
			_, err = addDeployment(fakeClientSet, "fsm-controller-1", testMeshName, testNamespace, fsmTestVersion, true)
			assert.Nil(err)

			uninstall := uninstallMeshCmd{
				in:                         in,
				out:                        out,
				force:                      true,
				client:                     helm.NewUninstall(testConfig),
				meshName:                   test.specifiedMesh,
				meshNamespace:              testNamespace,
				caBundleSecretName:         constants.DefaultCABundleSecretName,
				clientSet:                  fakeClientSet,
				extensionsClientset:        extensionsClientSetFake.NewSimpleClientset(),
				deleteClusterWideResources: true,
				actionConfig:               new(action.Configuration),
			}
			specifiedMeshFound := uninstall.findSpecifiedMesh(test.meshList)
			assert.Equal(specifiedMeshFound, test.expSpecifiedMeshFound)
		})
	}
}

func TestPromptMeshUninstall(t *testing.T) {
	tests := []struct {
		name                 string
		userPromptsYes       bool
		meshInfoList         []meshInfo
		specifiedMeshName    string
		expMeshesToUninstall []meshInfo
	}{
		{
			name:           "prompt no to uninstall mesh",
			userPromptsYes: false,
			meshInfoList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			specifiedMeshName:    "",
			expMeshesToUninstall: []meshInfo{},
		},
		{
			name:           "prompt yes to uninstall mesh",
			userPromptsYes: true,
			meshInfoList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			specifiedMeshName: "",
			expMeshesToUninstall: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
		},
		{
			name:           "prompt no to uninstall mesh for specified mesh",
			userPromptsYes: false,
			meshInfoList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			specifiedMeshName:    testMeshName,
			expMeshesToUninstall: []meshInfo{},
		},
		{
			name:           "prompt yes to uninstall mesh for specified mesh",
			userPromptsYes: true,
			meshInfoList: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
			specifiedMeshName: testMeshName,
			expMeshesToUninstall: []meshInfo{
				{
					name:      testMeshName,
					namespace: testNamespace,
				},
			},
		},
	}

	meshesToUninstall := []meshInfo{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			if test.userPromptsYes {
				in.Write([]byte("y\n"))
			} else {
				in.Write([]byte("n\n"))
			}

			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(testNamespace)
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: testMeshName})
			err := store.Create(rel)
			assert.Nil(err)

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet := fake.NewSimpleClientset()
			_, err = addDeployment(fakeClientSet, "fsm-controller-1", testMeshName, testNamespace, fsmTestVersion, true)
			assert.Nil(err)

			uninstall := uninstallMeshCmd{
				in:                         in,
				out:                        out,
				force:                      true,
				client:                     helm.NewUninstall(testConfig),
				meshName:                   test.specifiedMeshName,
				meshNamespace:              testNamespace,
				caBundleSecretName:         constants.DefaultCABundleSecretName,
				clientSet:                  fakeClientSet,
				extensionsClientset:        extensionsClientSetFake.NewSimpleClientset(),
				deleteClusterWideResources: true,
				actionConfig:               new(action.Configuration),
			}

			meshList, err := uninstall.promptMeshUninstall(test.meshInfoList, meshesToUninstall)
			assert.Nil(err)
			assert.ElementsMatch(test.expMeshesToUninstall, meshList)
		})
	}
}
