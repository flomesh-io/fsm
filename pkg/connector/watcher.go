package connector

import (
	"fmt"
	"reflect"

	"github.com/flomesh-io/fsm/pkg/announcements"
	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/logger"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

var (
	GatewayAPIEnabled = false
)

// WatchMeshConfigUpdated watches update of meshconfig
func WatchMeshConfigUpdated(msgBroker *messaging.Broker, stop <-chan struct{}) {
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

			if prevObj.Spec.GatewayAPI.Enabled != newObj.Spec.GatewayAPI.Enabled {
				GatewayAPIEnabled = newObj.Spec.GatewayAPI.Enabled
			}

			if prevObj.Spec.ClusterSet.Name != newObj.Spec.ClusterSet.Name &&
				prevObj.Spec.ClusterSet.Group != newObj.Spec.ClusterSet.Group &&
				prevObj.Spec.ClusterSet.Zone != newObj.Spec.ClusterSet.Zone &&
				prevObj.Spec.ClusterSet.Region != newObj.Spec.ClusterSet.Region {
				ServiceSourceValue = fmt.Sprintf("%s.%s.%s.%s",
					newObj.Spec.ClusterSet.Name,
					newObj.Spec.ClusterSet.Group,
					newObj.Spec.ClusterSet.Zone,
					newObj.Spec.ClusterSet.Region)
			}

			if !reflect.DeepEqual(prevObj.Spec.Connector, newObj.Spec.Connector) {
				viaGateway := &newObj.Spec.Connector.ViaGateway
				if len(viaGateway.IngressAddr) > 0 && len(viaGateway.EgressAddr) > 0 {
					ViaGateway.ClusterIP = viaGateway.ClusterIP
					ViaGateway.ExternalIP = viaGateway.ExternalIP
					ViaGateway.IngressAddr = viaGateway.IngressAddr
					ViaGateway.Ingress.HTTPPort = viaGateway.IngressHTTPPort
					ViaGateway.Ingress.GRPCPort = viaGateway.IngressGRPCPort
					ViaGateway.EgressAddr = viaGateway.EgressAddr
					ViaGateway.Egress.HTTPPort = viaGateway.EgressHTTPPort
					ViaGateway.Egress.GRPCPort = viaGateway.EgressGRPCPort
				}
			}
		}
	}
}
