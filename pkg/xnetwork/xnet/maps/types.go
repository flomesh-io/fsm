package maps

import "github.com/flomesh-io/fsm/pkg/logger"

var (
	log = logger.New("fsm-xnet-ebpf-maps")
)

const (
	SysMesh = SysID(1)
	SysE4lb = SysID(2)
)

const (
	BPF_F_EGRESS  = 0
	BPF_F_INGRESS = 1
)

type SysID uint32

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
	Sys   uint32
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
	Sys   uint32
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
		Raddr   [4]uint32
		Rport   uint16
		Rmac    [6]uint8
		Ofi     uint32
		Oflags  uint32
		Omac    [6]uint8
		OmacSet uint8
		Active  uint8
	}
}

const (
	CfgFlagOffsetDenyAll uint8 = iota
	CfgFlagOffsetAllowAll
	CfgFlagOffsetTCPProtoDenyAll
	CfgFlagOffsetTCPProtoAllowAll
	CfgFlagOffsetTCPProtoAllowNatEscape
	CfgFlagOffsetUDPProtoDenyAll
	CfgFlagOffsetUDPProtoAllowAll
	CfgFlagOffsetUDPProtoAllowNatEscape
	CfgFlagOffsetOTHProtoDenyAll
	CfgFlagOffsetTCPNatByIpPortOn
	CfgFlagOffsetTCPNatByIpOn
	CfgFlagOffsetTCPNatAllOff
	CfgFlagOffsetTCPNatOptOn
	CfgFlagOffsetTCPNatOptWithLocalAddrOn
	CfgFlagOffsetTCPNatOptWithLocalPortOn
	CfgFlagOffsetUDPNatByIpPortOn
	CfgFlagOffsetUDPNatByIpOn
	CfgFlagOffsetUDPNatByPortOn
	CfgFlagOffsetUDPNatAllOff
	CfgFlagOffsetUDPNatOptOn
	CfgFlagOffsetUDPNatOptWithLocalAddrOn
	CfgFlagOffsetUDPNatOptWithLocalPortOn
	CfgFlagOffsetAclCheckOn
	CfgFlagOffsetTraceHdrOn
	CfgFlagOffsetTraceNatOn
	CfgFlagOffsetTraceOptOn
	CfgFlagOffsetTraceAclOn
	CfgFlagOffsetTraceFlowOn
	CfgFlagOffsetTraceByIpOn
	CfgFlagOffsetTraceByPortOn
	CfgFlagMax
)

type FlagT struct {
	Flags uint64
}

type CfgKey uint32
type CfgVal struct {
	Ipv4 FlagT
	Ipv6 FlagT
}
