package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/mesh"
)

// GetSidecarProxyConfig returns the sidecar proxy config of a pod
func GetSidecarProxyConfig(clientSet kubernetes.Interface, config *rest.Config, namespace string, podName string, localPort uint16, query string) ([]byte, error) {
	// Check if the pod belongs to a mesh
	pod, err := clientSet.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Could not find pod %s in namespace %s", podName, namespace)
	}
	if !mesh.ProxyLabelExists(*pod) {
		return nil, fmt.Errorf("Pod %s in namespace %s is not a part of a mesh", podName, namespace)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("Pod %s in namespace %s is not running", podName, namespace)
	}

	dialer, err := k8s.DialerToPod(config, clientSet, podName, namespace)
	if err != nil {
		return nil, err
	}

	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", localPort, constants.SidecarAdminPort))
	if err != nil {
		return nil, fmt.Errorf("Error setting up port forwarding: %w", err)
	}

	var sidecarProxyConfig []byte
	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()
		url := fmt.Sprintf("http://localhost:%d/%s", localPort, query)

		// #nosec G107: Potential HTTP request made with variable url
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("Error fetching url %s: %w", url, err)
		}
		//nolint: errcheck
		//#nosec G307
		defer resp.Body.Close()

		sidecarProxyConfig, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Error rendering HTTP response: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Error retrieving proxy config for pod %s in namespace %s: %w", podName, namespace, err)
	}

	return sidecarProxyConfig, nil
}
