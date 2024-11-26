package maps

import "github.com/flomesh-io/fsm/pkg/logger"

var (
	log = logger.New("fsm-xnet-ebpf-maps")
)

const (
	IPPROTO_TCP L4Proto = 6
	IPPROTO_UDP L4Proto = 17
)

type L4Proto uint8

const (
	ACL_DENY    Acl = 0
	ACL_TRUSTED Acl = 2
)

type Acl uint8

type AclKey struct {
	Addr  [4]uint32
	Port  uint16
	Proto uint8
}

type AclVal struct {
	Acl  uint8
	Flag uint8
	Id   uint16
}

type IFaceKey struct {
	Len  uint8
	Name [16]uint8
}

type IFaceVal struct {
	Ifi  uint32
	Addr [4]uint32
	Mac  [6]uint8
}

const (
	TC_DIR_IGR TcDir = 0
	TC_DIR_EGR TcDir = 1
)

type TcDir uint8

type NatKey struct {
	Daddr [4]uint32
	Dport uint16
	Proto uint8
	V6    uint8
	TcDir uint8
}

type NatVal struct {
	Lock  struct{ Val uint32 }
	EpSel uint16
	EpCnt uint16
	Eps   [128]struct {
		Raddr    [4]uint32
		Rport    uint16
		Rmac     [6]uint8
		Inactive uint8
		_        [3]byte
	}
}

const (
	CfgFlagOffsetIPv6ProtoDenyAll uint8 = iota
	CfgFlagOffsetIPv4TCPProtoDenyAll
	CfgFlagOffsetIPv4TCPProtoAllowAll
	CfgFlagOffsetIPv4TCPProtoAllowNatEscape
	CfgFlagOffsetIPv4UDPProtoDenyAll
	CfgFlagOffsetIPv4UDPProtoAllowAll
	CfgFlagOffsetIPv4UDPProtoAllowNatEscape
	CfgFlagOffsetIPv4OTHProtoDenyAll
	CfgFlagOffsetIPv4TCPNatByIpPortOn
	CfgFlagOffsetIPv4TCPNatByIpOn
	CfgFlagOffsetIPv4TCPNatAllOff
	CfgFlagOffsetIPv4TCPNatOptOn
	CfgFlagOffsetIPv4TCPNatOptWithLocalAddrOn
	CfgFlagOffsetIPv4TCPNatOptWithLocalPortOn
	CfgFlagOffsetIPv4UDPNatByIpPortOn
	CfgFlagOffsetIPv4UDPNatByIpOn
	CfgFlagOffsetIPv4UDPNatByPortOn
	CfgFlagOffsetIPv4UDPNatAllOff
	CfgFlagOffsetIPv4UDPNatOptOn
	CfgFlagOffsetIPv4UDPNatOptWithLocalAddrOn
	CfgFlagOffsetIPv4UDPNatOptWithLocalPortOn
	CfgFlagOffsetIPv4AclCheckOn
	CfgFlagOffsetIPv4TraceHdrOn
	CfgFlagOffsetIPv4TraceNatOn
	CfgFlagOffsetIPv4TraceOptOn
	CfgFlagOffsetIPv4TraceAclOn
	CfgFlagOffsetIPv4TraceFlowOn
	CfgFlagOffsetIPv4TraceByIpOn
	CfgFlagOffsetIPv4TraceByPortOn
	CfgFlagMax
)

type CfgKey uint32
type CfgVal struct{ Flags uint64 }
