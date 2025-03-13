package connector

import (
	"net"

	ctv1 "github.com/flomesh-io/fsm/pkg/apis/connector/v1alpha1"
	"github.com/flomesh-io/fsm/pkg/constants"
	"github.com/flomesh-io/fsm/pkg/logger"
)

var (
	log = logger.New("connector")
)

const (
	ProtocolHTTP = MicroServiceProtocol(constants.ProtocolHTTP)
	ProtocolGRPC = MicroServiceProtocol(constants.ProtocolGRPC)
)

// MicroServiceProtocol defines string as microservice protocol
type MicroServiceProtocol string

func (p *MicroServiceProtocol) Get() string {
	return string(*p)
}

func (p *MicroServiceProtocol) Set(protocol string) {
	*p = MicroServiceProtocol(protocol)
}

func (p *MicroServiceProtocol) SetVar(protocol MicroServiceProtocol) {
	*p = protocol
}

func (p *MicroServiceProtocol) Empty() bool {
	return len(*p) == 0
}

// MicroServicePort defines int as microservice port
type MicroServicePort int32

func (p *MicroServicePort) Get() int32 {
	return int32(*p)
}

func (p *MicroServicePort) Set(port int32) {
	*p = MicroServicePort(port)
}

// MicroServiceAddress defines string as microservice address
type MicroServiceAddress string

func (a *MicroServiceAddress) Get() string {
	return string(*a)
}

func (a *MicroServiceAddress) Set(addr string) {
	*a = MicroServiceAddress(addr)
}

// To4 converts the IPv4 address ip to a 4-byte representation.
// If ip is not an IPv4 address, To4 returns nil.
func (a *MicroServiceAddress) To4() net.IP {
	return net.ParseIP(string(*a)).To4()
}

// To16 converts the IP address ip to a 16-byte representation.
// If ip is not an IP address (it is the wrong length), To16 returns nil.
func (a *MicroServiceAddress) To16() net.IP {
	return net.ParseIP(string(*a)).To16()
}

type MicroServiceVia struct {
	viaAddress MicroServiceAddress
	viaPort    MicroServicePort
}

func (v *MicroServiceVia) Get() (MicroServiceAddress, MicroServicePort) {
	return v.viaAddress, v.viaPort
}

func (v *MicroServiceVia) Set(viaAddress MicroServiceAddress, viaPort MicroServicePort) {
	v.viaAddress = viaAddress
	v.viaPort = viaPort
}

type MicroServiceEndpoint struct {
	address MicroServiceAddress
	port    MicroServicePort
}

func (ept *MicroServiceEndpoint) Get() (MicroServiceAddress, MicroServicePort) {
	return ept.address, ept.port
}

func (ept *MicroServiceEndpoint) Set(address MicroServiceAddress, port MicroServicePort) {
	ept.address = address
	ept.port = port
}

type MicroService struct {
	ctv1.NamespacedService

	protocol MicroServiceProtocol
	endpoint MicroServiceEndpoint
	via      MicroServiceVia
}

func (s *MicroService) SetHTTPPort(port int32) {
	s.endpoint.port = MicroServicePort(port)
	s.protocol = ProtocolHTTP
}

func (s *MicroService) SetGRPCPort(port int32) {
	s.endpoint.port = MicroServicePort(port)
	s.protocol = ProtocolGRPC
}

func (s *MicroService) Protocol() *MicroServiceProtocol {
	return &s.protocol
}

func (s *MicroService) Endpoint() *MicroServiceEndpoint {
	return &s.endpoint
}

func (s *MicroService) EndpointAddress() *MicroServiceAddress {
	return &s.endpoint.address
}
func (s *MicroService) EndpointPort() *MicroServicePort {
	return &s.endpoint.port
}

func (s *MicroService) Via() *MicroServiceVia {
	return &s.via
}
