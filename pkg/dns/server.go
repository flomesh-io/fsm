package dns

import (
	"fmt"
	"time"

	"github.com/miekg/dns"

	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/constants"
)

// Server type
type Server struct {
	host      string
	rTimeout  time.Duration
	wTimeout  time.Duration
	handler   *DNSHandler
	udpServer *dns.Server
	tcpServer *dns.Server
}

// Run starts the server
func (s *Server) Run(config *Config) {
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
	log.Info().Msgf("start %s listener on %s\n", ds.Net, s.host)

	if err := ds.ListenAndServe(); err != nil {
		log.Error().Msgf("start %s listener on %s failed: %s\n", ds.Net, s.host, err.Error())
	}
}

// Stop stops the server
func (s *Server) Stop() {
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

func Start(cfg configurator.Configurator) {
	config := &Config{cfg: cfg}

	server := &Server{
		host:     fmt.Sprintf(":%d", constants.FSMDNSProxyPort),
		rTimeout: 5 * time.Second,
		wTimeout: 5 * time.Second,
	}

	server.Run(config)
}
