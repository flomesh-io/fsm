package k8s

import (
	"context"
	"fmt"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
)

// WatchAndUpdateProxyBootstrapSecret watches for new pods being added to the mesh
// and updates their corresponding proxy bootstrap config Secret's OwnerReferences
// to point to the associated pod.
func WatchAndUpdateProxyBootstrapSecret(kubeClient kubernetes.Interface, msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	podAddChan := kubePubSub.Sub(announcements.PodAdded.String())
	defer msgBroker.Unsub(kubePubSub, podAddChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting proxy bootstrap secret update routine")
			return

		case podAddedMsg := <-podAddChan:
			psubMessage, castOk := podAddedMsg.(events.PubSubMessage)
			if !castOk {
				log.Error().Msgf("Error casting to events.PubSubMessage, got type %T", psubMessage)
				continue
			}

			// guaranteed can only be a PodAdded event
			addedPodObj, castOk := psubMessage.NewObj.(*corev1.Pod)
			if !castOk {
				log.Error().Msgf("Error casting to *corev1.Pod: got type %T", addedPodObj)
				continue
			}

			podUID := addedPodObj.GetUID()
			podUUID := addedPodObj.GetLabels()[constants.SidecarUniqueIDLabelName]
			podName := addedPodObj.GetName()
			namespace := addedPodObj.GetNamespace()
			if len(podUUID) == 0 {
				log.Info().Msgf("Ignored Pod %s/%s, not managed by fsm.", namespace, podName)
				continue
			}
			secretName := fmt.Sprintf("sidecar-bootstrap-config-%s", podUUID)

			secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Msgf("Failed to get secret %s/%s mounted to Pod %s/%s", namespace, secretName, namespace, podName)
				continue
			}

			secret.ObjectMeta.OwnerReferences = append(secret.ObjectMeta.OwnerReferences, metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       podName,
				UID:        podUID,
			})

			if _, err = kubeClient.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{}); err != nil {
				// There might be conflicts when multiple controllers try to update the same resource
				// One of the controllers will successfully update the resource, hence conflicts should be ignored and not treated as an error
				if !apierrors.IsConflict(err) {
					log.Error().Err(err).Msgf("Failed to update OwnerReference for Secret %s/%s to reference Pod %s/%s", namespace, secretName, namespace, podName)
				}
			} else {
				log.Debug().Msgf("Updated OwnerReference for Secret %s/%s to reference Pod %s/%s", namespace, secretName, namespace, podName)
			}
		}
	}
}

// WatchAndUpdateLogLevel watches for log level changes and updates the global log level
func WatchAndUpdateLogLevel(msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting log level update routine")
			return

		case event := <-meshCfgUpdateChan:
			msg, ok := event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Error casting to PubSubMessage, got type %T", msg)
				continue
			}

			prevObj, prevOk := msg.OldObj.(*configv1alpha3.MeshConfig)
			newObj, newOk := msg.NewObj.(*configv1alpha3.MeshConfig)
			if !prevOk || !newOk {
				log.Error().Msgf("Error casting to *MeshConfig, got type prev=%T, new=%T", prevObj, newObj)
			}

			// Update the log level if necessary
			if prevObj.Spec.Observability.FSMLogLevel != newObj.Spec.Observability.FSMLogLevel {
				if err := logger.SetLogLevel(newObj.Spec.Observability.FSMLogLevel); err != nil {
					log.Error().Err(err).Msgf("Error setting controller log level to %s", newObj.Spec.Observability.FSMLogLevel)
				}
			}
		}
	}
}

// WatchAndUpdateIngressConfig watches for log level changes and updates the global log level
func WatchAndUpdateIngressConfig(kubeClient *kubernetes.Clientset, msgBroker *messaging.Broker, fsmNamespace string, certMgr *certificate.Manager, repoClient *repo.PipyRepoClient, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting log level update routine")
			return

		case event := <-meshCfgUpdateChan:
			msg, ok := event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Error casting to PubSubMessage, got type %T", msg)
				continue
			}

			oldCfg, prevOk := msg.OldObj.(*configv1alpha3.MeshConfig)
			cfg, newOk := msg.NewObj.(*configv1alpha3.MeshConfig)
			if !prevOk || !newOk {
				log.Error().Msgf("Error casting to *MeshConfig, got type prev=%T, new=%T", oldCfg, cfg)
			}

			log.Info().Msgf("Updating basic config ...")

			if isHTTPConfigChanged(oldCfg, cfg) {
				if err := utils.UpdateIngressHTTPConfig(constants.DefaultIngressBasePath, repoClient, cfg); err != nil {
					log.Error().Msgf("Failed to update HTTP config: %s", err)
				}
			}

			if isTLSConfigChanged(oldCfg, cfg) {
				if cfg.Spec.Ingress.TLS.Enabled {
					if err := utils.IssueCertForIngress(constants.DefaultIngressBasePath, repoClient, certMgr, cfg); err != nil {
						log.Error().Msgf("Failed to update TLS config and issue default cert: %s", err)
					}
				} else {
					if err := utils.UpdateIngressTLSConfig(constants.DefaultIngressBasePath, repoClient, cfg); err != nil {
						log.Error().Msgf("Failed to update TLS config: %s", err)
					}
				}
			}

			if shouldUpdateIngressControllerServiceSpec(oldCfg, cfg) {
				updateIngressControllerSpec(kubeClient, fsmNamespace, oldCfg, cfg)
			}
		}
	}
}

func updateIngressControllerSpec(kubeClient *kubernetes.Clientset, fsmNamespace string, oldCfg, cfg *configv1alpha3.MeshConfig) {
	selector := labels.SelectorFromSet(
		map[string]string{
			"app":                           "fsm-ingress",
			"ingress.flomesh.io/namespaced": "false",
		},
	)
	svcList, err := kubeClient.CoreV1().
		Services(fsmNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})

	if err != nil {
		log.Error().Msgf("Failed to list all ingress-pipy services: %s", err)
		return
	}

	// as container port of pod is informational, only change svc spec is enough
	for _, svc := range svcList.Items {
		service := svc.DeepCopy()
		service.Spec.Ports = nil

		if cfg.Spec.Ingress.HTTP.Enabled {
			httpPort := corev1.ServicePort{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       cfg.Spec.Ingress.HTTP.Bind,
				TargetPort: intstr.FromInt(int(cfg.Spec.Ingress.HTTP.Listen)),
			}
			if cfg.Spec.Ingress.HTTP.NodePort > 0 {
				httpPort.NodePort = cfg.Spec.Ingress.HTTP.NodePort
			}
			service.Spec.Ports = append(service.Spec.Ports, httpPort)
		}

		if cfg.Spec.Ingress.TLS.Enabled {
			tlsPort := corev1.ServicePort{
				Name:       "https",
				Protocol:   corev1.ProtocolTCP,
				Port:       cfg.Spec.Ingress.TLS.Bind,
				TargetPort: intstr.FromInt(int(cfg.Spec.Ingress.TLS.Listen)),
			}
			if cfg.Spec.Ingress.TLS.NodePort > 0 {
				tlsPort.NodePort = cfg.Spec.Ingress.TLS.NodePort
			}
			service.Spec.Ports = append(service.Spec.Ports, tlsPort)
		}

		if len(service.Spec.Ports) > 0 {
			if _, err := kubeClient.CoreV1().
				Services(fsmNamespace).
				Update(context.TODO(), service, metav1.UpdateOptions{}); err != nil {
				log.Error().Msgf("Failed update spec of ingress-pipy service: %s", err)
			}
		} else {
			log.Warn().Msgf("Both HTTP and TLS are disabled, ignore updating ingress-pipy service")
		}
	}
}

func isHTTPConfigChanged(oldCfg, cfg *configv1alpha3.MeshConfig) bool {
	return cfg.Spec.Ingress.Enabled &&
		(oldCfg.Spec.Ingress.HTTP.Enabled != cfg.Spec.Ingress.HTTP.Enabled ||
			oldCfg.Spec.Ingress.HTTP.Listen != cfg.Spec.Ingress.HTTP.Listen)
}

func isTLSConfigChanged(oldCfg, cfg *configv1alpha3.MeshConfig) bool {
	return cfg.Spec.Ingress.Enabled &&
		(oldCfg.Spec.Ingress.TLS.Enabled != cfg.Spec.Ingress.TLS.Enabled ||
			oldCfg.Spec.Ingress.TLS.Listen != cfg.Spec.Ingress.TLS.Listen ||
			oldCfg.Spec.Ingress.TLS.MTLS != cfg.Spec.Ingress.TLS.MTLS)
}

func shouldUpdateIngressControllerServiceSpec(oldCfg, cfg *configv1alpha3.MeshConfig) bool {
	return cfg.Spec.Ingress.Enabled &&
		(oldCfg.Spec.Ingress.TLS.Enabled != cfg.Spec.Ingress.TLS.Enabled ||
			oldCfg.Spec.Ingress.TLS.Listen != cfg.Spec.Ingress.TLS.Listen ||
			oldCfg.Spec.Ingress.TLS.Bind != cfg.Spec.Ingress.TLS.Bind ||
			oldCfg.Spec.Ingress.TLS.NodePort != cfg.Spec.Ingress.TLS.NodePort ||
			oldCfg.Spec.Ingress.HTTP.Enabled != cfg.Spec.Ingress.HTTP.Enabled ||
			oldCfg.Spec.Ingress.HTTP.Listen != cfg.Spec.Ingress.HTTP.Listen ||
			oldCfg.Spec.Ingress.HTTP.NodePort != cfg.Spec.Ingress.HTTP.NodePort ||
			oldCfg.Spec.Ingress.HTTP.Bind != cfg.Spec.Ingress.HTTP.Bind)
}
