package debugger

import (
	"net/http"

	configv1alpha2 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha2"

	"github.com/flomesh-io/fsm/pkg/announcements"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/httpserver"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
)

// StartDebugServerConfigListener registers a go routine to listen to configuration and configure debug server as needed
func (d *DebugConfig) StartDebugServerConfigListener(httpDebugHandlers map[string]http.Handler, stop chan struct{}) {
	// This is the Debug server
	httpDebugServer := httpserver.NewHTTPServer(constants.DebugPort)
	httpDebugServer.AddHandlers(d.GetHandlers(httpDebugHandlers))

	kubePubSub := d.msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer d.msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	started := false
	if d.configurator.IsDebugServerEnabled() {
		if err := httpDebugServer.Start(); err != nil {
			log.Error().Err(err).Msgf("error starting debug server")
		}
		started = true
	}

	for {
		select {
		case event := <-meshCfgUpdateChan:
			msg, ok := event.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Error casting to PubSubMessage, got type %T", msg)
				continue
			}

			prevSpec := msg.OldObj.(*configv1alpha2.MeshConfig).Spec
			newSpec := msg.NewObj.(*configv1alpha2.MeshConfig).Spec

			if prevSpec.Observability.EnableDebugServer == newSpec.Observability.EnableDebugServer {
				continue
			}

			enableDbgServer := newSpec.Observability.EnableDebugServer
			if enableDbgServer && !started {
				if err := httpDebugServer.Start(); err != nil {
					log.Error().Err(err).Msgf("error starting debug server")
				} else {
					started = true
				}
			} else if !enableDbgServer && started {
				if err := httpDebugServer.Stop(); err != nil {
					log.Error().Err(err).Msgf("error stopping debug server")
				} else {
					started = false
				}
			}

		case <-stop:
			log.Info().Msg("Received stop signal, exiting debug server config listener")
			return
		}
	}
}
