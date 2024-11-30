package dns

import (
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"

	"github.com/flomesh-io/fsm/pkg/announcements"
	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/k8s"
	"github.com/flomesh-io/fsm/pkg/k8s/events"
	"github.com/flomesh-io/fsm/pkg/messaging"
)

// Server type
type Server struct {
	running   bool
	lock      sync.Mutex
	host      string
	rTimeout  time.Duration
	wTimeout  time.Duration
	handler   *DNSHandler
	udpServer *dns.Server
	tcpServer *dns.Server
}

// Run starts the server
func (s *Server) run(config *Config) {
	s.handler = NewHandler(config)

	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", s.handler.DoTCP)

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", s.handler.DoUDP)

	for _, record := range NewCustomDNSRecordsFromText(config.CustomDNSRecords) {
		handleFunc := record.serve(s.handler)
		tcpHandler.HandleFunc(record.name, handleFunc)
		udpHandler.HandleFunc(record.name, handleFunc)
	}

	s.tcpServer = &dns.Server{Addr: s.host,
		Net:          "tcp",
		Handler:      tcpHandler,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	s.udpServer = &dns.Server{Addr: s.host,
		Net:          "udp",
		Handler:      udpHandler,
		UDPSize:      65535,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	go s.start(s.udpServer)
	go s.start(s.tcpServer)
}

func (s *Server) start(ds *dns.Server) {
	log.Info().Msgf("start %s listener on %s", ds.Net, s.host)

	if err := ds.ListenAndServe(); err != nil {
		log.Error().Msgf("start %s listener on %s failed: %s", ds.Net, s.host, err.Error())
	}
}

// Stop stops the server
func (s *Server) stop() {
	if s.handler != nil {
		s.handler.muActive.Lock()
		s.handler.active = false
		close(s.handler.requestChannel)
		s.handler.muActive.Unlock()
	}
	if s.udpServer != nil {
		err := s.udpServer.Shutdown()
		if err != nil {
			log.Error().Err(err)
		}
	}
	if s.tcpServer != nil {
		err := s.tcpServer.Shutdown()
		if err != nil {
			log.Error().Err(err)
		}
	}
}

var (
	k8sClient k8s.Controller
	cfg       configurator.Configurator

	server = &Server{
		host:     fmt.Sprintf(":%d", constants.FSMDNSProxyPort),
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}
)

func Init(kubeController k8s.Controller, configurator configurator.Configurator) {
	k8sClient = kubeController
	cfg = configurator
	if cfg.IsLocalDNSProxyEnabled() {
		Start()
	}
}

func Start() {
	server.lock.Lock()
	defer server.lock.Unlock()

	if !server.running {
		config := &Config{cfg: cfg}
		server.run(config)
		server.running = true
	}
}

func Stop() {
	server.lock.Lock()
	defer server.lock.Unlock()

	if server.running {
		server.stop()
		server.running = false
	}
}

func WatchAndUpdateLocalDNSProxy(msgBroker *messaging.Broker, stop <-chan struct{}) {
	kubePubSub := msgBroker.GetKubeEventPubSub()
	meshCfgUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer msgBroker.Unsub(kubePubSub, meshCfgUpdateChan)

	for {
		select {
		case <-stop:
			log.Info().Msg("Received stop signal, exiting local dns proxy update routine")
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

			if prevObj.Spec.Sidecar.LocalDNSProxy.Enable != newObj.Spec.Sidecar.LocalDNSProxy.Enable {
				if newObj.Spec.Sidecar.LocalDNSProxy.Enable {
					Start()
				} else {
					Stop()
				}
			}
		}
	}
}
