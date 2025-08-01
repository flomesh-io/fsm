package v2

import (
	"net"
	"time"
)

// Result is a string used to handle the results for probing container readiness/liveness
type Result string

const (
	// Success Result
	Success Result = "success"
	// Failure Result
	Failure Result = "failure"
)

// doTCPProbe checks that a TCP socket to the address can be opened.
// If the socket can be opened, it returns Success
// If the socket fails to open, it returns Failure.
// This is exported because some other packages may want to do direct TCP probes.
func (s *Server) doTCPProbe(host, port string, timeout time.Duration) Result {
	if conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout); err != nil {
		log.Debug().Msgf("error opening TCP probe socket: %v (%#v)", err, err)
		return Failure
	} else if err = conn.Close(); err != nil {
		log.Debug().Msgf("error closing TCP probe socket: %v (%#v)", err, err)
	}
	return Success
}

// doUDPProbe checks that a UDP socket to the address can be opened.
// If the socket can be opened, it returns Success
// If the socket fails to open, it returns Failure.
// This is exported because some other packages may want to do direct TCP probes.
func (s *Server) doUDPProbe(host, port string, timeout time.Duration) Result {
	if conn, err := net.DialTimeout("udp", net.JoinHostPort(host, port), timeout); err != nil {
		log.Debug().Msgf("error opening UDP probe socket: %v (%#v)", err, err)
		return Failure
	} else if err = conn.Close(); err != nil {
		log.Debug().Msgf("error closing UDP probe socket: %v (%#v)", err, err)
	}
	return Success
}
