package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/flomesh-io/fsm/pkg/constants"
)

const (
	testRegistrySecret  = "test-registry-secret"
	testVaultHost       = "vault.fsm.svc.cluster.local"
	testVaultToken      = "token"
	testVaultSecretName = "secret"
	testVaultSecretKey  = "key"
	testChartPath       = "testdata/test-chart"
	kubeVersionMajor    = 1
	kubeVersionMinor    = 22
	kubeVersionPatch    = 9
)

func helmCapabilities() *chartutil.Capabilities {
	defaultCapabilities := chartutil.DefaultCapabilities.Copy()
	// Intentionally avoiding charutil.ParseKubeVersion so we don't have to
	// deal with error handling when generating the capabilities.
	defaultCapabilities.KubeVersion = chartutil.KubeVersion{
		Version: fmt.Sprintf(
			"v%d.%d.%d",
			kubeVersionMajor,
			kubeVersionMinor,
			kubeVersionPatch,
		),
		Major: strconv.Itoa(kubeVersionMajor),
		Minor: strconv.Itoa(kubeVersionMinor),
	}
	return defaultCapabilities
}

var _ = Describe("Running the install command", func() {

	Describe("with default parameters", func() {
		var (
			out    *bytes.Buffer
			store  *storage.Storage
			config *helm.Configuration
			err    error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.FsmNamespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := getDefaultInstallCmd(out)

			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("FSM installed successfully in namespace [fsm-system] with mesh name [fsm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				defaultValues := getDefaultValues()
				Expect(rel.Config).To(BeEquivalentTo(defaultValues))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.FsmNamespace()))
			})
		})
	})

	Describe("with a default Helm chart", func() {
		var (
			out    *bytes.Buffer
			store  *storage.Storage
			config *helm.Configuration
			err    error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.FsmNamespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := getDefaultInstallCmd(out)
			installCmd.chartPath = "testdata/test-chart"

			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("FSM installed successfully in namespace [fsm-system] with mesh name [fsm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				defaultValues := getDefaultValues()
				Expect(rel.Config).To(BeEquivalentTo(defaultValues))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.FsmNamespace()))
			})
		})
	})

	Describe("with the vault cert manager using vault token", func() {
		var (
			out    *bytes.Buffer
			store  *storage.Storage
			config *helm.Configuration
			err    error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.FsmNamespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := getDefaultInstallCmd(out)

			installCmd.setOptions = []string{
				"fsm.certificateProvider.kind=vault",
				fmt.Sprintf("fsm.vault.host=%s", testVaultHost),
				fmt.Sprintf("fsm.vault.token=%s", testVaultToken),
			}
			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("FSM installed successfully in namespace [fsm-system] with mesh name [fsm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				expectedValues := getDefaultValues()
				valuesConfig := []string{
					fmt.Sprintf("fsm.certificateProvider.kind=%s", "vault"),
					fmt.Sprintf("fsm.vault.host=%s", testVaultHost),
					fmt.Sprintf("fsm.vault.token=%s", testVaultToken),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(rel.Config).To(BeEquivalentTo(expectedValues))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.FsmNamespace()))
			})
		})
	})

	Describe("with the vault cert manager using token secret ref", func() {
		var (
			out    *bytes.Buffer
			store  *storage.Storage
			config *helm.Configuration
			err    error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.FsmNamespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := getDefaultInstallCmd(out)

			installCmd.setOptions = []string{
				"fsm.certificateProvider.kind=vault",
				fmt.Sprintf("fsm.vault.host=%s", testVaultHost),
				"fsm.vault.token=",
				fmt.Sprintf("fsm.vault.secret.name=%s", testVaultSecretName),
				fmt.Sprintf("fsm.vault.secret.key=%s", testVaultSecretKey),
			}
			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("FSM installed successfully in namespace [fsm-system] with mesh name [fsm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				expectedValues := getDefaultValues()
				valuesConfig := []string{
					fmt.Sprintf("fsm.certificateProvider.kind=%s", "vault"),
					fmt.Sprintf("fsm.vault.host=%s", testVaultHost),
					"fsm.vault.token=",
					fmt.Sprintf("fsm.vault.secret.name=%s", testVaultSecretName),
					fmt.Sprintf("fsm.vault.secret.key=%s", testVaultSecretKey),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(rel.Config).To(BeEquivalentTo(expectedValues))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.FsmNamespace()))
			})
		})
	})

	Describe("without required vault parameters", func() {
		var (
			installCmd installCmd
			config     *helm.Configuration
		)

		BeforeEach(func() {
			out := new(bytes.Buffer)
			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.FsmNamespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd = getDefaultInstallCmd(out)
			installCmd.chartPath = filepath.FromSlash("../../charts/fsm")
			installCmd.setOptions = []string{
				"fsm.certificateProvider.kind=vault",
			}
		})

		It("should error when host isn't set", func() {
			err := installCmd.run(config)
			Expect(err.Error()).To(ContainSubstring("fsm.vault.host is required"))
		})

		It("should error when token and token secret key are not set", func() {
			installCmd.setOptions = append(installCmd.setOptions,
				"fsm.vault.host=my-host",
				"fsm.vault.secret.name=secret",
			)
			err := installCmd.run(config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fsm.vault.secret.key is required"))
		})

		It("should error when token and token secret name are not set", func() {
			installCmd.setOptions = append(installCmd.setOptions,
				"fsm.vault.host=my-host",
				"fsm.vault.secret.key=key",
			)
			err := installCmd.run(config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fsm.vault.secret.name is required"))
		})

		It("should error when token and token secret name and key are not set", func() {
			installCmd.setOptions = append(installCmd.setOptions,
				"fsm.vault.host=my-host",
			)
			err := installCmd.run(config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fsm.vault.token is required"))
		})
	})

	Describe("with the cert-manager certificate manager", func() {
		var (
			out    *bytes.Buffer
			store  *storage.Storage
			config *helm.Configuration
			err    error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.FsmNamespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: helmCapabilities(),
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := getDefaultInstallCmd(out)
			installCmd.setOptions = []string{
				"fsm.certificateProvider.kind=cert-manager",
			}
			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("FSM installed successfully in namespace [fsm-system] with mesh name [fsm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				expectedValues := getDefaultValues()
				valuesConfig := []string{
					fmt.Sprintf("fsm.certificateProvider.kind=%s", "cert-manager"),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(rel.Config).To(BeEquivalentTo(expectedValues))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.FsmNamespace()))
			})
		})
	})
})

var _ = Describe("deployPrometheus is true", func() {
	var (
		out    *bytes.Buffer
		store  *storage.Storage
		config *helm.Configuration
		err    error
	)

	BeforeEach(func() {
		out = new(bytes.Buffer)
		store = storage.Init(driver.NewMemory())
		if mem, ok := store.Driver.(*driver.Memory); ok {
			mem.SetNamespace(settings.FsmNamespace())
		}

		config = &helm.Configuration{
			Releases: store,
			KubeClient: &kubefake.PrintingKubeClient{
				Out: ioutil.Discard},
			Capabilities: helmCapabilities(),
			Log:          func(format string, v ...interface{}) {},
		}

		installCmd := getDefaultInstallCmd(out)
		installCmd.setOptions = []string{
			"fsm.deployPrometheus=true",
		}

		err = installCmd.run(config)
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})
})

func TestResolveValues(t *testing.T) {
	tests := []struct {
		name        string
		installCmd  installCmd
		expected    map[string]interface{}
		expectedErr error
	}{
		{
			name: "default",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)

				// Fill out fields which are empty by default
				installCmd.setOptions = []string{
					fmt.Sprintf("fsm.imagePullSecrets[0].name=%s", testRegistrySecret),
					fmt.Sprintf("fsm.vault.token=%s", testVaultToken),
					fmt.Sprintf("fsm.vault.host=%s", testVaultHost),
				}
				return installCmd
			}(),
			expected: func() map[string]interface{} {
				expectedValues := getDefaultValues()

				// Fill out fields which are empty by default
				valuesConfig := []string{
					fmt.Sprintf("fsm.imagePullSecrets[0].name=%s", testRegistrySecret),
					fmt.Sprintf("fsm.vault.host=%s", testVaultHost),
					fmt.Sprintf("fsm.vault.token=%s", testVaultToken),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					tassert.Nil(t, err)
				}
				return expectedValues
			}(),
		},
		{
			name: "--set creates additional values",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)
				installCmd.setOptions = []string{"new=from set", "key1=val1,key2=val2"}
				return installCmd
			}(),
			expected: func() map[string]interface{} {
				vals := getDefaultValues()
				vals["new"] = "from set"
				vals["key1"] = "val1"
				vals["key2"] = "val2"
				return vals
			}(),
		},
		{
			name: "--set for an existing parameter as no effect",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)
				installCmd.setOptions = []string{"fsm.meshName=set"}
				return installCmd
			}(),
			expected: getDefaultValues(),
		},
		{
			name: "invalid --set format",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)
				installCmd.setOptions = []string{"can't set this"}
				return installCmd
			}(),
			expectedErr: errors.New("invalid format for --set: key \"can't set this\" has no value"),
		},
	}

	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			actual, err := test.installCmd.resolveValues()
			if err != nil && test.expectedErr != nil {
				assert.Equal(err.Error(), test.expectedErr.Error())
			} else {
				assert.Equal(err, test.expectedErr)
			}
			assert.Equal(actual, test.expected, "Test at index %d failed", idx)
		})
	}
}

func createDeploymentSpec(namespace, meshName string) *v1.Deployment {
	labelMap := make(map[string]string)
	if meshName != "" {
		labelMap["meshName"] = meshName
		labelMap[constants.AppLabel] = constants.FSMControllerName
	}
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.FSMControllerName,
			Namespace: namespace,
			Labels:    labelMap,
		},
	}
}

func getDefaultInstallCmd(writer io.Writer) installCmd {
	installCmd := installCmd{
		out:               writer,
		chartPath:         defaultChartPath,
		meshName:          defaultMeshName,
		clientSet:         fake.NewSimpleClientset(),
		enforceSingleMesh: defaultEnforceSingleMesh,
		disableSpinner:    true,
	}

	return installCmd
}

func getDefaultValues() map[string]interface{} {
	return map[string]interface{}{
		"fsm": map[string]interface{}{
			"meshName":          defaultMeshName,
			"enforceSingleMesh": defaultEnforceSingleMesh,
		}}
}
