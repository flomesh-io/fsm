package framework

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"

	goversion "github.com/hashicorp/go-version"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
)

const (
	// DefaultFsmGrafanaPort is the default Grafana port
	DefaultFsmGrafanaPort = 3000
	// DefaultFsmPrometheusPort default FSM prometheus port
	DefaultFsmPrometheusPort = 7070

	// FsmGrafanaAppLabel is the FSM Grafana deployment app label
	FsmGrafanaAppLabel = "fsm-grafana"
	// FsmPrometheusAppLabel is the FSM Prometheus deployment app label
	FsmPrometheusAppLabel = "fsm-prometheus"

	// FSM Grafana Dashboard specifics

	// MeshDetails is dashboard uuid and name as we have them load in Grafana
	MeshDetails string = "PLyKJcHGz/mesh-and-sidecar-details"

	// MemRSSPanel is the ID of the MemRSS panel on FSM's MeshDetails dashboard
	MemRSSPanel int = 13

	// CPUPanel is the ID of the CPU panel on FSM's MeshDetails dashboard
	CPUPanel int = 14

	// maxPodCreationRetries determines the max number of retries for creating
	// a Pod (including via a Deployment) upon failure
	maxPodCreationRetries = 2

	// delayIntervalForPodCreationRetries
	delayIntervalForPodCreationRetries = 5 * time.Second
)

var (
	// FsmCtlLabels is the list of app labels for FSM CTL
	FsmCtlLabels = []string{constants.FSMControllerName, FsmGrafanaAppLabel, FsmPrometheusAppLabel, constants.FSMInjectorName, constants.FSMBootstrapName}

	// NginxIngressSvc is the namespaced name of the nginx ingress service
	NginxIngressSvc = types.NamespacedName{Namespace: "ingress-ns", Name: "ingress-nginx-controller"}
)

// CreateServiceAccount is a wrapper to create a service account
// If creating on OpenShift, add privileged SCC
func (td *FsmTestData) CreateServiceAccount(ns string, svcAccount *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
	svcAc, err := td.Client.CoreV1().ServiceAccounts(ns).Create(context.Background(), svcAccount, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Service Account: %w", err)
		return nil, err
	}
	if Td.DeployOnOpenShift {
		err = Td.AddOpenShiftSCC("privileged", svcAc.Name, svcAc.Namespace)
		return svcAc, err
	}
	return svcAc, nil
}

// createRole is a wrapper to create a role
func (td *FsmTestData) createRole(ns string, role *rbacv1.Role) (*rbacv1.Role, error) {
	r, err := td.Client.RbacV1().Roles(ns).Create(context.Background(), role, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Role: %w", err)
		return nil, err
	}

	return r, nil
}

// createRoleBinding is a wrapper to create a role binding
func (td *FsmTestData) createRoleBinding(ns string, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	rb, err := td.Client.RbacV1().RoleBindings(ns).Create(context.Background(), roleBinding, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create RoleBinding: %w", err)
		return nil, err
	}

	return rb, nil
}

func (td *FsmTestData) getMaxPodCreationRetries() int {
	if td.RetryAppPodCreation {
		return maxPodCreationRetries
	}
	return 1
}

// CreatePod is a wrapper to create a pod
func (td *FsmTestData) CreatePod(ns string, pod corev1.Pod) (*corev1.Pod, error) {
	maxRetries := td.getMaxPodCreationRetries()

	for i := 1; i <= maxRetries; i++ {
		if i > 1 {
			// Sleep before next retry
			time.Sleep(delayIntervalForPodCreationRetries)
		}
		podRet, err := td.Client.CoreV1().Pods(ns).Create(context.Background(), &pod, metav1.CreateOptions{})
		if err != nil {
			td.T.Logf("Could not create Pod in attempt %d due to error: %w", i, err)
			continue
		}
		return podRet, nil
	}
	return nil, fmt.Errorf("Error creating pod in namespace %s after %d attempts", ns, maxRetries)
}

// CreateDeployment is a wrapper to create a deployment
func (td *FsmTestData) CreateDeployment(ns string, deployment appsv1.Deployment) (*appsv1.Deployment, error) {
	maxRetries := td.getMaxPodCreationRetries()

	for i := 1; i <= maxRetries; i++ {
		if i > 1 {
			// Sleep before next retry
			time.Sleep(delayIntervalForPodCreationRetries)
		}
		deploymentRet, err := td.Client.AppsV1().Deployments(ns).Create(context.Background(), &deployment, metav1.CreateOptions{})
		if err != nil {
			td.T.Logf("Could not create Deployment in attempt %d due to error: %v", i, err)
			continue
		}
		return deploymentRet, nil
	}
	return nil, fmt.Errorf("Error creating Deployment in namespace %s after %d attempts", ns, maxRetries)
}

// CreateService is a wrapper to create a service
func (td *FsmTestData) CreateService(ns string, svc corev1.Service) (*corev1.Service, error) {
	sv, err := td.Client.CoreV1().Services(ns).Create(context.Background(), &svc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create Service: %w", err)
		return nil, err
	}
	return sv, nil
}

// CreateConfigMap is a wrapper to create a config map
func (td *FsmTestData) CreateConfigMap(ns string, cm corev1.ConfigMap) (*corev1.ConfigMap, error) {
	cmRet, err := td.Client.CoreV1().ConfigMaps(ns).Create(context.Background(), &cm, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create ConfigMap: %w", err)
		return nil, err
	}
	return cmRet, nil
}

// CreateMutatingWebhook is a wrapper to create a mutating webhook configuration
func (td *FsmTestData) CreateMutatingWebhook(mwhc *admissionregv1.MutatingWebhookConfiguration) (*admissionregv1.MutatingWebhookConfiguration, error) {
	mw, err := td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), mwhc, metav1.CreateOptions{})
	if err != nil {
		err := fmt.Errorf("Could not create MutatingWebhook: %w", err)
		return nil, err
	}
	return mw, nil
}

// GetMutatingWebhook is a wrapper to get a mutating webhook configuration
func (td *FsmTestData) GetMutatingWebhook(mwhcName string) (*admissionregv1.MutatingWebhookConfiguration, error) {
	mwhc, err := td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), mwhcName, metav1.GetOptions{})
	if err != nil {
		err := fmt.Errorf("Could not get MutatingWebhook: %w", err)
		return nil, err
	}
	return mwhc, nil
}

// GetPodsForLabel returns the Pods matching a specific `appLabel`
func (td *FsmTestData) GetPodsForLabel(ns string, labelSel metav1.LabelSelector) ([]corev1.Pod, error) {
	// Apparently there has to be a conversion between metav1 and labels
	labelMap, _ := metav1.LabelSelectorAsMap(&labelSel)

	pods, err := Td.Client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})

	if err != nil {
		return nil, err
	}

	return pods.Items, nil
}

/* Application templates
 * The following functions contain high level helpers to create and get test application definitions.
 *
 * These abstractions aim to simplify and avoid tests having to individually type the the same k8s definitions for
 * some common or recurrent deployment forms.
 */

// SimplePodAppDef defines some parametrization to create a pod-based application from template
type SimplePodAppDef struct {
	Namespace          string
	PodName            string
	ServiceName        string
	ServiceAccountName string
	ContainerName      string
	Image              string
	Command            []string
	Args               []string
	Ports              []int
	AppProtocol        string
	OS                 string
	Labels             map[string]string
}

// SimplePodApp returns a set of k8s typed definitions for a pod-based k8s definition.
// Includes Pod, Service and ServiceAccount types
func (td *FsmTestData) SimplePodApp(def SimplePodAppDef) (corev1.ServiceAccount, corev1.Pod, corev1.Service, error) {
	if len(def.OS) == 0 {
		return corev1.ServiceAccount{}, corev1.Pod{}, corev1.Service{}, fmt.Errorf("ClusterOS must be explicitly specified")
	}

	if len(def.PodName) == 0 {
		return corev1.ServiceAccount{}, corev1.Pod{}, corev1.Service{}, fmt.Errorf("PodName must be explicitly specified")
	}

	if def.Labels == nil {
		def.Labels = map[string]string{constants.AppLabel: def.PodName}
	}

	serviceAccountName := def.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = RandomNameWithPrefix("serviceaccount")
	}

	serviceName := def.ServiceName
	if serviceName == "" {
		serviceName = RandomNameWithPrefix("service")
	}

	containerName := def.ContainerName
	if containerName == "" {
		containerName = def.PodName
	}

	serviceAccountDefinition := Td.SimpleServiceAccount(serviceAccountName, def.Namespace)

	podDefinition := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.PodName,
			Namespace: def.Namespace,
			Labels:    def.Labels,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: new(int64), // 0
			ServiceAccountName:            serviceAccountName,
			NodeSelector: map[string]string{
				"kubernetes.io/os": def.OS,
			},
			Containers: []corev1.Container{
				{
					Name:            containerName,
					Image:           def.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
					Env: []corev1.EnvVar{
						{
							Name: "POD_IP",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  "status.podIP",
								},
							},
						},
					},
				},
			},
		},
	}

	if td.AreRegistryCredsPresent() {
		podDefinition.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: RegistrySecretName,
			},
		}
	}
	if def.Command != nil && len(def.Command) > 0 {
		podDefinition.Spec.Containers[0].Command = def.Command
	}
	if def.Args != nil && len(def.Args) > 0 {
		podDefinition.Spec.Containers[0].Args = def.Args
	}

	serviceDefinition := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName,
			Labels: def.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: def.Labels,
		},
	}

	if def.Ports != nil && len(def.Ports) > 0 {
		podDefinition.Spec.Containers[0].Ports = []corev1.ContainerPort{}
		serviceDefinition.Spec.Ports = []corev1.ServicePort{}

		for _, p := range def.Ports {
			podDefinition.Spec.Containers[0].Ports = append(podDefinition.Spec.Containers[0].Ports,
				corev1.ContainerPort{
					ContainerPort: int32(p),
				},
			)

			svcPort := corev1.ServicePort{
				Port:       int32(p),
				TargetPort: intstr.FromInt(p),
			}

			if def.AppProtocol != "" {
				if ver, err := td.getKubernetesServerVersionNumber(); err != nil {
					svcPort.Name = fmt.Sprintf("%s-%d", def.AppProtocol, p) // use named port with AppProtocol
				} else {
					// use appProtocol field in servicePort if k8s server version >= 1.19
					if ver[0] >= 1 && ver[1] >= 19 {
						svcPort.AppProtocol = &def.AppProtocol // set the appProtocol field
					} else {
						svcPort.Name = fmt.Sprintf("%s-%d", def.AppProtocol, p) // use named port with AppProtocol
					}
				}
			}

			serviceDefinition.Spec.Ports = append(serviceDefinition.Spec.Ports, svcPort)
		}
	}

	return serviceAccountDefinition, podDefinition, serviceDefinition, nil
}

// SimpleServiceAccount returns a k8s typed definition for a service account.
func (td *FsmTestData) SimpleServiceAccount(name string, namespace string) corev1.ServiceAccount {
	serviceAccountDefinition := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return serviceAccountDefinition
}

// simpleRole returns a k8s typed definition for a role.
func (td *FsmTestData) simpleRole(name string, namespace string) rbacv1.Role {
	roleDefinition := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return roleDefinition
}

// simpleRoleBinding returns a k8s typed definition for a role binding.
func (td *FsmTestData) simpleRoleBinding(name string, namespace string) rbacv1.RoleBinding {
	roleBindingDefinition := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return roleBindingDefinition
}

// getKubernetesServerVersionNumber returns the version number in chunks, ex. v1.19.3 => [1, 19, 3]
func (td *FsmTestData) getKubernetesServerVersionNumber() ([]int, error) {
	version, err := td.Client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("Error getting K8s server version: %w", err)
	}

	ver, err := goversion.NewVersion(version.String())
	if err != nil {
		return nil, fmt.Errorf("Error parsing k8s server version %s: %w", version, err)
	}

	return ver.Segments(), nil
}

// SimpleDeploymentAppDef defines some parametrization to create a deployment-based application from template
type SimpleDeploymentAppDef struct {
	Namespace          string
	DeploymentName     string
	ServiceName        string
	ContainerName      string
	ServiceAccountName string
	Image              string
	ReplicaCount       int32
	Command            PodCommand
	Args               []string
	Ports              []int
	AppProtocol        string
	OS                 string
	Labels             map[string]string
}

// PodCommand describes a command for a pod
type PodCommand []string

// PodCommandDefault is the default pod command (nothing)
var PodCommandDefault = []string{}

// SimpleDeploymentApp creates returns a set of k8s typed definitions for a deployment-based k8s definition.
// Includes Deployment, Service and ServiceAccount types
func (td *FsmTestData) SimpleDeploymentApp(def SimpleDeploymentAppDef) (corev1.ServiceAccount, appsv1.Deployment, corev1.Service, error) {
	if len(def.OS) == 0 {
		return corev1.ServiceAccount{}, appsv1.Deployment{}, corev1.Service{}, fmt.Errorf("ClusterOS must be explicitly specified")
	}

	if def.Labels == nil {
		def.Labels = map[string]string{constants.AppLabel: def.DeploymentName}
	}

	serviceAccountName := def.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = RandomNameWithPrefix("serviceaccount")
	}

	serviceName := def.ServiceName
	if serviceName == "" {
		serviceName = RandomNameWithPrefix("service")
	}

	serviceAccountDefinition := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: def.Namespace,
		},
	}
	containerName := def.ContainerName
	if containerName == "" {
		containerName = def.DeploymentName
	}

	// Required, as replica count is a pointer to distinguish between 0 and not specified
	replicaCountExplicitDeclaration := def.ReplicaCount

	deploymentDefinition := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.DeploymentName,
			Namespace: def.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCountExplicitDeclaration,
			Selector: &metav1.LabelSelector{
				MatchLabels: def.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: def.Labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: new(int64), // 0
					ServiceAccountName:            serviceAccountName,
					NodeSelector: map[string]string{
						"kubernetes.io/os": def.OS,
					},
					Containers: []corev1.Container{
						{
							Name:            containerName,
							Image:           def.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}

	if td.AreRegistryCredsPresent() {
		deploymentDefinition.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: RegistrySecretName,
			},
		}
	}

	if def.Command != nil && len(def.Command) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Command = def.Command
	}
	if def.Args != nil && len(def.Args) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Args = def.Args
	}

	serviceDefinition := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: def.Namespace,
			Labels:    def.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: def.Labels,
		},
	}

	if def.Ports != nil && len(def.Ports) > 0 {
		deploymentDefinition.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{}
		serviceDefinition.Spec.Ports = []corev1.ServicePort{}

		for _, p := range def.Ports {
			deploymentDefinition.Spec.Template.Spec.Containers[0].Ports = append(deploymentDefinition.Spec.Template.Spec.Containers[0].Ports,
				corev1.ContainerPort{
					ContainerPort: int32(p),
				},
			)

			svcPort := corev1.ServicePort{
				Port:       int32(p),
				TargetPort: intstr.FromInt(p),
			}

			if def.AppProtocol != "" {
				if ver, err := td.getKubernetesServerVersionNumber(); err != nil {
					svcPort.Name = fmt.Sprintf("%s-%d", def.AppProtocol, p) // use named port with AppProtocol
				} else {
					// use appProtocol field in servicePort if k8s server version >= 1.19
					if ver[0] >= 1 && ver[1] >= 19 {
						svcPort.AppProtocol = &def.AppProtocol // set the appProtocol field
					} else {
						svcPort.Name = fmt.Sprintf("%s-%d", def.AppProtocol, p) // use named port with AppProtocol
					}
				}
			}

			serviceDefinition.Spec.Ports = append(serviceDefinition.Spec.Ports, svcPort)
		}
	}

	return serviceAccountDefinition, deploymentDefinition, serviceDefinition, nil
}

// GetOSSpecificHTTPBinPod returns a OS pod that runs httpbin.
func (td *FsmTestData) GetOSSpecificHTTPBinPod(podName string, namespace string, customCommand ...string) (corev1.ServiceAccount, corev1.Pod, corev1.Service, error) {
	appDef := SimplePodAppDef{
		PodName:   podName,
		Namespace: namespace,
		Image:     "flomesh/httpbin:ken",
		Ports:     []int{80},
		OS:        Td.ClusterOS,
	}

	if len(customCommand) > 0 {
		appDef.Command = customCommand
	}

	return Td.SimplePodApp(appDef)
}

// GetOSSpecificSleepPod returns a simple OS specific busy loop pod.
func (td *FsmTestData) GetOSSpecificSleepPod(sourceNs string) (corev1.ServiceAccount, corev1.Pod, corev1.Service, error) {
	return Td.SimplePodApp(SimplePodAppDef{
		PodName:   RandomNameWithPrefix("pod"),
		Namespace: sourceNs,
		Command:   []string{"/bin/bash", "-c", "--"},
		Args:      []string{"while true; do sleep 30; done;"},
		Image:     "flomesh/alpine-debug",
		Ports:     []int{80},
		OS:        td.ClusterOS,
	})
}

// GetOSSpecificTCPEchoPod returns a simple OS specific tcp-echo pod.
func (td *FsmTestData) GetOSSpecificTCPEchoPod(podName string, namespace string, destinationPort int) (corev1.ServiceAccount, corev1.Pod, corev1.Service, error) {
	var image string
	var command string
	installOpts := Td.GetFSMInstallOpts()
	image = fmt.Sprintf("%s/fsm-demo-tcp-echo-server:%s", installOpts.ContainerRegistryLoc, installOpts.FsmImagetag)
	command = "/tcp-echo-server"
	return Td.SimplePodApp(
		SimplePodAppDef{
			PodName:     podName,
			Namespace:   namespace,
			Image:       image,
			Command:     []string{command},
			Args:        []string{"--port", fmt.Sprintf("%d", destinationPort)},
			Ports:       []int{destinationPort},
			AppProtocol: constants.ProtocolTCP,
			OS:          Td.ClusterOS,
		})
}

// GetGrafanaPodHandle generic func to forward a grafana pod and returns a handler pointing to the locally forwarded resource
func (td *FsmTestData) GetGrafanaPodHandle(ns string, grafanaPodName string, port uint16) (*Grafana, error) {
	dialer, err := k8s.DialerToPod(td.RestConfig, td.Client, grafanaPodName, ns)
	if err != nil {
		return nil, err
	}
	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", port, port))
	if err != nil {
		return nil, fmt.Errorf("Error setting up port forwarding: %w", err)
	}

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Could not start forwarding: %w", err)
	}

	return &Grafana{
		Schema:   "http",
		Hostname: "localhost",
		Port:     port,
		User:     "admin", // default value of grafana deployment
		Password: "admin", // default value of grafana deployment
		pfwd:     portForwarder,
	}, nil
}

// GetPrometheusPodHandle generic func to forward a prometheus pod and returns a handler pointing to the locally forwarded resource
func (td *FsmTestData) GetPrometheusPodHandle(ns string, prometheusPodName string, port uint16) (*Prometheus, error) {
	dialer, err := k8s.DialerToPod(td.RestConfig, td.Client, prometheusPodName, ns)
	if err != nil {
		return nil, err
	}
	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", port, port))
	if err != nil {
		return nil, fmt.Errorf("Error setting up port forwarding: %w", err)
	}

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Could not start forwarding: %w", err)
	}

	client, err := api.NewClient(api.Config{
		Address: fmt.Sprintf("http://localhost:%d", port),
	})
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(client)

	return &Prometheus{
		Client: client,
		API:    v1api,
		pfwd:   portForwarder,
	}, nil
}

func (td *FsmTestData) waitForFSMControlPlane() error {
	var errController, errInjector, errBootstrap error
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(3)

	go func() {
		defer GinkgoRecover()
		errController = td.WaitForPodsRunningReady(td.FsmNamespace, 1, &metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.AppLabel: constants.FSMControllerName,
			},
		})
		waitGroup.Done()
	}()

	go func() {
		defer GinkgoRecover()
		errInjector = td.WaitForPodsRunningReady(td.FsmNamespace, 1, &metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.AppLabel: constants.FSMInjectorName,
			},
		})
		waitGroup.Done()
	}()

	go func() {
		defer GinkgoRecover()
		errBootstrap = td.WaitForPodsRunningReady(td.FsmNamespace, 1, &metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.AppLabel: constants.FSMBootstrapName,
			},
		})
		waitGroup.Done()
	}()

	waitGroup.Wait()

	if errController != nil || errInjector != nil {
		return fmt.Errorf("FSM Control plane was not ready in time (%v, %v, %v)", errController, errInjector, errBootstrap)
	}

	return nil
}

// GetFSMPrometheusHandle convenience wrapper, will get the Prometheus instance regularly deployed
// by FSM installation in test <FsmNamespace>
func (td *FsmTestData) GetFSMPrometheusHandle() (*Prometheus, error) {
	prometheusPod, err := Td.GetPodsForLabel(Td.FsmNamespace, metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.AppLabel: FsmPrometheusAppLabel,
		},
	})
	if err != nil || len(prometheusPod) == 0 {
		return nil, fmt.Errorf("Error getting Prometheus pods: %w (prom pods len: %d)", err, len(prometheusPod))
	}
	pHandle, err := Td.GetPrometheusPodHandle(prometheusPod[0].Namespace, prometheusPod[0].Name, DefaultFsmPrometheusPort)
	if err != nil {
		return nil, err
	}

	return pHandle, nil
}

// GetFSMGrafanaHandle convenience wrapper, will get the Grafana instance regularly deployed
// by FSM installation in test <FsmNamespace>
func (td *FsmTestData) GetFSMGrafanaHandle() (*Grafana, error) {
	grafanaPod, err := Td.GetPodsForLabel(Td.FsmNamespace, metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.AppLabel: FsmGrafanaAppLabel,
		},
	})
	if err != nil || len(grafanaPod) == 0 {
		return nil, fmt.Errorf("Error getting Grafana pods: %w (graf pods len: %d)", err, len(grafanaPod))
	}
	gHandle, err := Td.GetGrafanaPodHandle(grafanaPod[0].Namespace, grafanaPod[0].Name, DefaultFsmGrafanaPort)
	if err != nil {
		return nil, err
	}
	return gHandle, nil
}

// InstallNginxIngress installs the k8s Nginx Ingress controller and returns the IP address
// that clients can send traffic to for ingress
func (td *FsmTestData) InstallNginxIngress() (string, error) {
	// Check the node's provider so this works for preprovisioned kind clusters
	nodes, err := td.Client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("Error listing nodes to install nginx ingress: %w", err)
	}

	providerID := nodes.Items[0].Spec.ProviderID
	isKind := strings.HasPrefix(providerID, "kind://")
	var vals map[string]interface{}
	if isKind {
		vals = map[string]interface{}{
			"controller": map[string]interface{}{
				"hostPort": map[string]interface{}{
					"enabled": true,
				},
				"nodeSelector": map[string]interface{}{
					"ingress-ready": "true",
				},
				"service": map[string]interface{}{
					"type": "NodePort",
				},
			},
		}
	}

	if err := td.CreateNs(NginxIngressSvc.Namespace, nil); err != nil {
		return "", fmt.Errorf("Error creating namespace for nginx ingress: %w", err)
	}

	helmConfig := &action.Configuration{}
	if err := helmConfig.Init(Td.Env.RESTClientGetter(), NginxIngressSvc.Namespace, "secret", Td.T.Logf); err != nil {
		return "", fmt.Errorf("Error initializing Helm config for nginx ingress: %w", err)
	}

	helmConfig.KubeClient.(*kube.Client).Namespace = NginxIngressSvc.Namespace

	install := action.NewInstall(helmConfig)
	install.RepoURL = "https://kubernetes.github.io/ingress-nginx"
	install.Namespace = NginxIngressSvc.Namespace
	install.ReleaseName = "ingress-nginx"
	install.Version = "4.0.18"
	install.Wait = true
	install.Timeout = 5 * time.Minute

	chartPath, err := install.LocateChart("ingress-nginx", helmcli.New())
	if err != nil {
		return "", fmt.Errorf("Error locating ingress-nginx Helm chart: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("Error loading ingress-nginx chart %s: %w", chartPath, err)
	}

	if _, err = install.Run(chart, vals); err != nil {
		return "", fmt.Errorf("Error installing ingress-nginx: %w", err)
	}

	ingressAddr := "localhost"
	if !isKind {
		svc, err := Td.Client.CoreV1().Services(NginxIngressSvc.Namespace).Get(context.Background(), NginxIngressSvc.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("Error getting service: %s/%s: %w", NginxIngressSvc.Namespace, NginxIngressSvc.Name, err)
		}

		ingressAddr = svc.Status.LoadBalancer.Ingress[0].IP
		if len(ingressAddr) == 0 {
			ingressAddr = svc.Status.LoadBalancer.Ingress[0].Hostname
		}
	}

	return ingressAddr, nil
}

// RandomNameWithPrefix generates a random string with the given prefix.
//
//	If the prefix is empty, the default prefix "test" will be used
func RandomNameWithPrefix(prefix string) string {
	if prefix == "" || len(prefix) > 100 {
		prefix = "test"
	}
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String())
}
