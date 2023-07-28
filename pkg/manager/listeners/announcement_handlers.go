package listeners

import (
	"context"
	"github.com/flomesh-io/fsm/pkg/announcements"
	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/manager/utils"
	"github.com/flomesh-io/fsm/pkg/messaging"
	repo "github.com/flomesh-io/fsm/pkg/sidecar/providers/pipy/client"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

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
			newCfg, newOk := msg.NewObj.(*configv1alpha3.MeshConfig)
			if !prevOk || !newOk {
				log.Error().Msgf("Error casting to *MeshConfig, got type prev=%T, new=%T", oldCfg, newCfg)
			}

			log.Info().Msgf("Updating basic config ...")

			if isHTTPConfigChanged(oldCfg, newCfg) {
				if err := utils.UpdateIngressHTTPConfig(constants.DefaultIngressBasePath, repoClient, meshConfigToConfigurator(newCfg)); err != nil {
					log.Error().Msgf("Failed to update HTTP config: %s", err)
				}
			}

			if isTLSConfigChanged(oldCfg, newCfg) {
				if newCfg.Spec.Ingress.TLS.Enabled {
					if err := utils.IssueCertForIngress(constants.DefaultIngressBasePath, repoClient, certMgr, meshConfigToConfigurator(newCfg)); err != nil {
						log.Error().Msgf("Failed to update TLS config and issue default cert: %s", err)
					}
				} else {
					if err := utils.UpdateIngressTLSConfig(constants.DefaultIngressBasePath, repoClient, meshConfigToConfigurator(newCfg)); err != nil {
						log.Error().Msgf("Failed to update TLS config: %s", err)
					}
				}
			}

			if shouldUpdateIngressControllerServiceSpec(oldCfg, newCfg) {
				updateIngressControllerSpec(kubeClient, fsmNamespace, oldCfg, newCfg)
			}
		}
	}
}

func updateIngressControllerSpec(kubeClient *kubernetes.Clientset, fsmNamespace string, oldCfg, newCfg *configv1alpha3.MeshConfig) {
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

		if newCfg.Spec.Ingress.HTTP.Enabled {
			httpPort := corev1.ServicePort{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       newCfg.Spec.Ingress.HTTP.Bind,
				TargetPort: intstr.FromInt(int(newCfg.Spec.Ingress.HTTP.Listen)),
			}
			if newCfg.Spec.Ingress.HTTP.NodePort > 0 {
				httpPort.NodePort = newCfg.Spec.Ingress.HTTP.NodePort
			}
			service.Spec.Ports = append(service.Spec.Ports, httpPort)
		}

		if newCfg.Spec.Ingress.TLS.Enabled {
			tlsPort := corev1.ServicePort{
				Name:       "https",
				Protocol:   corev1.ProtocolTCP,
				Port:       newCfg.Spec.Ingress.TLS.Bind,
				TargetPort: intstr.FromInt(int(newCfg.Spec.Ingress.TLS.Listen)),
			}
			if newCfg.Spec.Ingress.TLS.NodePort > 0 {
				tlsPort.NodePort = newCfg.Spec.Ingress.TLS.NodePort
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
