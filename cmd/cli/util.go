package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	gatewayApiClientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	nsigClientset "github.com/flomesh-io/fsm/pkg/gen/client/namespacedingress/clientset/versioned"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/tidwall/sjson"

	"k8s.io/apimachinery/pkg/types"

	mapset "github.com/deckarep/golang-set"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
)

const (
	restartAtEnvName = "RESTART_AT"
)

// confirm displays a prompt `s` to the user and returns a bool indicating yes / no
// If the lowercased, trimmed input begins with anything other than 'y', it returns false
// It accepts an int `tries` representing the number of attempts before returning false
func confirm(stdin io.Reader, stdout io.Writer, s string, tries int) (bool, error) {
	r := bufio.NewReader(stdin)

	for ; tries > 0; tries-- {
		fmt.Fprintf(stdout, "%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			return false, err
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(res)) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			fmt.Fprintf(stdout, "Invalid input.\n")
			continue
		}
	}

	return false, nil
}

// getPrettyPrintedMeshInfoList returns a pretty printed list of meshes.
func getPrettyPrintedMeshInfoList(meshInfoList []meshInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\tVERSION\tADDED NAMESPACES\n"

	for _, meshInfo := range meshInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\t%s\n",
			meshInfo.name,
			meshInfo.namespace,
			meshInfo.version,
			strings.Join(meshInfo.monitoredNamespaces, ","),
		)
		s += m
	}

	return s
}

// getMeshInfoList returns a list of meshes (including the info of each mesh) within the cluster
func getMeshInfoList(restConfig *rest.Config, clientSet kubernetes.Interface) ([]meshInfo, error) {
	var meshInfoList []meshInfo

	fsmControllerDeployments, err := getControllerDeployments(clientSet)
	if err != nil {
		return meshInfoList, fmt.Errorf("Could not list deployments %w", err)
	}
	if len(fsmControllerDeployments.Items) == 0 {
		return meshInfoList, nil
	}

	for _, fsmControllerDeployment := range fsmControllerDeployments.Items {
		meshName := fsmControllerDeployment.ObjectMeta.Labels["meshName"]
		meshNamespace := fsmControllerDeployment.ObjectMeta.Namespace

		meshVersion := fsmControllerDeployment.ObjectMeta.Labels[constants.FSMAppVersionLabelKey]
		if meshVersion == "" {
			meshVersion = "Unknown"
		}

		var meshMonitoredNamespaces []string
		nsList, err := selectNamespacesMonitoredByMesh(meshName, clientSet)
		if err == nil && len(nsList.Items) > 0 {
			for _, ns := range nsList.Items {
				meshMonitoredNamespaces = append(meshMonitoredNamespaces, ns.Name)
			}
		}

		meshInfoList = append(meshInfoList, meshInfo{
			name:                meshName,
			namespace:           meshNamespace,
			version:             meshVersion,
			monitoredNamespaces: meshMonitoredNamespaces,
		})
	}

	return meshInfoList, nil
}

// getControllerDeployments returns a list of Deployments corresponding to fsm-controller
func getControllerDeployments(clientSet kubernetes.Interface) (*appsv1.DeploymentList, error) {
	deploymentsClient := clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.FSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return deploymentsClient.List(context.TODO(), listOptions)
}

// getControllerPods returns a list of fsm-controller Pods in a specified namespace
func getControllerPods(clientSet kubernetes.Interface, namespace string) (*corev1.PodList, error) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.FSMControllerName}}
	podClient := clientSet.CoreV1().Pods(namespace)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: listOptions.LabelSelector})
}

// getMeshNames returns a set of mesh names corresponding to meshes within the cluster
func getMeshNames(clientSet kubernetes.Interface) mapset.Set {
	meshList := mapset.NewSet()

	deploymentList, _ := getControllerDeployments(clientSet)
	for _, elem := range deploymentList.Items {
		meshList.Add(elem.ObjectMeta.Labels["meshName"])
	}

	return meshList
}

// getPrettyPrintedMeshSmiInfoList returns a pretty printed list
// of meshes with supported smi versions
func getPrettyPrintedMeshSmiInfoList(meshSmiInfoList []meshSmiInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\tSMI SUPPORTED\n"

	for _, mesh := range meshSmiInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\n",
			mesh.name,
			mesh.namespace,
			strings.Join(mesh.smiSupportedVersions, ","),
		)
		s += m
	}

	return s
}

// getSupportedSmiInfoForMeshList returns a meshSmiInfo list showing
// the supported smi versions for each fsm mesh in the mesh list
func getSupportedSmiInfoForMeshList(meshInfoList []meshInfo, clientSet kubernetes.Interface, config *rest.Config, localPort uint16) []meshSmiInfo {
	var meshSmiInfoList []meshSmiInfo

	for _, mesh := range meshInfoList {
		meshControllerPods := k8s.GetFSMControllerPods(clientSet, mesh.namespace)

		meshSmiSupportedVersions := []string{"Unknown"}
		if len(meshControllerPods.Items) > 0 {
			// for listing mesh information, checking info using the first fsm-controller pod should suffice
			controllerPod := meshControllerPods.Items[0]
			smiMap, err := getSupportedSmiForControllerPod(controllerPod.Name, mesh.namespace, config, clientSet, localPort)
			if err == nil {
				meshSmiSupportedVersions = []string{}
				for smi, version := range smiMap {
					meshSmiSupportedVersions = append(meshSmiSupportedVersions, fmt.Sprintf("%s:%s", smi, version))
				}
			}
		}
		sort.Strings(meshSmiSupportedVersions)

		meshSmiInfoList = append(meshSmiInfoList, meshSmiInfo{
			name:                 mesh.name,
			namespace:            mesh.namespace,
			smiSupportedVersions: meshSmiSupportedVersions,
		})
	}

	return meshSmiInfoList
}

// getSupportedSmiForControllerPod returns the supported smi versions
// for a given fsm controller pod in a namespace
func getSupportedSmiForControllerPod(pod string, namespace string, restConfig *rest.Config, clientSet kubernetes.Interface, localPort uint16) (map[string]string, error) {
	dialer, err := k8s.DialerToPod(restConfig, clientSet, pod, namespace)
	if err != nil {
		return nil, err
	}

	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", localPort, constants.FSMHTTPServerPort))
	if err != nil {
		return nil, fmt.Errorf("Error setting up port forwarding: %w", err)
	}

	var smiSupported map[string]string

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()
		url := fmt.Sprintf("http://localhost:%d%s", localPort, constants.FSMControllerSMIVersionPath)

		// #nosec G107: Potential HTTP request made with variable url
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("Error fetching url %s: %s", url, err)
		}

		if err := json.NewDecoder(resp.Body).Decode(&smiSupported); err != nil {
			return fmt.Errorf("Error rendering HTTP response: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Error retrieving supported SMI versions for pod %s in namespace %s: %s", pod, namespace, err)
	}

	for smiAPI, smiAPIVersion := range smiSupported {
		// smiApi looks like HTTPRouteGroup
		// smiApiVersion looks like specs.smi-spec.io/v1alpha4
		// leave out the API group and only keep the version after "/"
		splitVersionInfo := strings.SplitN(smiAPIVersion, "/", 2)
		if len(splitVersionInfo) >= 2 {
			smiSupported[smiAPI] = splitVersionInfo[1]
		}
	}

	return smiSupported, nil
}

func annotateErrorMessageWithFsmNamespace(errMsgFormat string, args ...interface{}) error {
	fsmNamespaceErrorMsg := fmt.Sprintf(
		"Note: The command failed when run in the FSM namespace [%s].\n"+
			"Use the global flag --fsm-namespace if [%s] is not the intended FSM namespace.",
		settings.Namespace(), settings.Namespace())

	return annotateErrorMessageWithActionableMessage(fsmNamespaceErrorMsg, errMsgFormat, args...)
}

func annotateErrorMessageWithActionableMessage(actionableMessage string, errMsgFormat string, args ...interface{}) error {
	if !strings.HasSuffix(errMsgFormat, "\n") {
		errMsgFormat += "\n"
	}

	if !strings.HasSuffix(errMsgFormat, "\n\n") {
		errMsgFormat += "\n"
	}

	return fmt.Errorf(errMsgFormat+actionableMessage, args...)
}

func restartFSMController(ctx context.Context, kubeClient kubernetes.Interface, fsmNamespace string) error {
	debug("Restarting fsm-controller ...")
	// Rollout restart fsm-controller
	// patch the deployment spec template triggers the action of rollout restart like with kubectl
	patch := fmt.Sprintf(
		`{"spec": {"template":{"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`,
		time.Now().Format("20060102-150405.0000"),
	)

	if _, err := kubeClient.AppsV1().
		Deployments(fsmNamespace).
		Patch(ctx, constants.FSMControllerName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		return err
	}

	return nil
}

func restartFSMControllerContainer(ctx context.Context, kubeClient kubernetes.Interface, fsmNamespace string) error {
	debug("Restarting fsm-controller container ...")

	// Get fsm-controller deployment
	deployment, err := kubeClient.AppsV1().Deployments(fsmNamespace).Get(ctx, constants.FSMControllerName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Get fsm-controller pods by selector of the deployment
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return err
	}
	pods, err := kubeClient.CoreV1().Pods(fsmNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}

	// Update env of fsm-controller container to trigger restart ONLY fsm-controller container,
	// no impact on other containers in the pod
	for idxp, pod := range pods.Items {
		for idxc, c := range pod.Spec.Containers {
			if c.Name != constants.FSMControllerName {
				continue
			}

			found := false
			for idxe, env := range c.Env {
				if env.Name == restartAtEnvName {
					pods.Items[idxp].Spec.Containers[idxc].Env[idxe] = corev1.EnvVar{
						Name:  restartAtEnvName,
						Value: time.Now().Format("20060102-150405.0000"),
					}
					found = true
					break
				}
			}

			if !found {
				pods.Items[idxp].Spec.Containers[idxc].Env = append(pods.Items[idxp].Spec.Containers[idxc].Env, corev1.EnvVar{
					Name:  restartAtEnvName,
					Value: time.Now().Format("20060102-150405.0000"),
				})
			}
		}

		if _, err := kubeClient.CoreV1().Pods(fsmNamespace).Update(ctx, &pods.Items[idxp], metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func updatePresetMeshConfigMap(ctx context.Context, kubeClient kubernetes.Interface, fsmNamespace string, values map[string]interface{}) error {
	debug("Getting configmap preset-mesh-config ...")
	// get configmap preset-mesh-config
	cm, err := kubeClient.CoreV1().ConfigMaps(fsmNamespace).Get(ctx, presetMeshConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	debug("Updating configmap preset-mesh-config ...")
	// update content data of preset-mesh-config.json
	presetMeshConfigJSON := cm.Data[presetMeshConfigJSONKey]
	for path, value := range values {
		presetMeshConfigJSON, err = sjson.Set(presetMeshConfigJSON, path, value)
		if err != nil {
			return err
		}
	}

	// update configmap preset-mesh-config
	cm.Data[presetMeshConfigJSONKey] = presetMeshConfigJSON
	_, err = kubeClient.CoreV1().ConfigMaps(fsmNamespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func deleteIngressResources(ctx context.Context, kubeClient kubernetes.Interface, fsmNamespace, meshName string) error {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.AppLabel:              constants.FSMIngressName,
			"meshName":                      meshName,
			"ingress.flomesh.io/namespaced": "false",
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	serviceList, err := kubeClient.CoreV1().Services(fsmNamespace).List(ctx, listOptions)
	for _, service := range serviceList.Items {
		err = kubeClient.CoreV1().Services(fsmNamespace).Delete(ctx, service.Name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	deploymentList, err := kubeClient.AppsV1().Deployments(fsmNamespace).List(ctx, listOptions)
	for _, deployment := range deploymentList.Items {
		err = kubeClient.AppsV1().Deployments(fsmNamespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	err = kubeClient.NetworkingV1().IngressClasses().Delete(ctx, constants.IngressPipyClass, metav1.DeleteOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func deleteNamespacedIngressResources(ctx context.Context, nsigClient nsigClientset.Interface) error {
	nsigList, err := nsigClient.FlomeshV1alpha1().NamespacedIngresses(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, nsig := range nsigList.Items {
		if err := nsigClient.FlomeshV1alpha1().NamespacedIngresses(nsig.GetNamespace()).Delete(ctx, nsig.GetName(), metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func deleteGatewayResources(ctx context.Context, gatewayAPIClient gatewayApiClientset.Interface) error {
	// delete gateways
	debug("Deleting gateways ...")
	gatewayList, err := gatewayAPIClient.GatewayV1beta1().Gateways(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, gateway := range gatewayList.Items {
		if err := gatewayAPIClient.GatewayV1beta1().Gateways(gateway.GetNamespace()).Delete(ctx, gateway.GetName(), metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	// delete gatewayclasses
	debug("Deleting gatewayclasses ...")
	gatewayClassList, err := gatewayAPIClient.GatewayV1beta1().GatewayClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, gatewayClass := range gatewayClassList.Items {
		if err := gatewayAPIClient.GatewayV1beta1().GatewayClasses().Delete(ctx, gatewayClass.GetName(), metav1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}

	// TODO: delete XXXRoute resources and ReferenceGrant?

	return nil
}

func deleteEgressGatewayResources(ctx context.Context, kubeClient kubernetes.Interface, fsmNamespace, meshName string) error {
	if err := kubeClient.CoreV1().Services(fsmNamespace).Delete(ctx, constants.FSMEgressGatewayName, metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err := kubeClient.AppsV1().Deployments(fsmNamespace).Delete(ctx, constants.FSMEgressGatewayName, metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err := kubeClient.CoreV1().ConfigMaps(fsmNamespace).Delete(ctx, "fsm-egress-gateway-pjs", metav1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
