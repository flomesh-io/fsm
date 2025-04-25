package neigh

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"syscall"
)

const (
	ETH_P_ARP = 1544 //htons(syscall.ETH_P_ARP)
)

var (
	// Broadcast is a special hardware address which indicates a Frame should
	// be sent to every device on a given LAN segment.
	broadcast = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

// An Operation is an ARP operation, such as request or reply.
type arpOperation uint16

// Operation constants which indicate an ARP request or reply.
const (
	operationRequest arpOperation = 1
	operationReply   arpOperation = 2
)

// gratuitousARPOverIface sends an gratuitous arp over interface 'iface' from 'eip'.
func gratuitousARPOverIface(eip net.IP, ifName string, ifIndex int, macAddr net.HardwareAddr) error {
	request := newARPRequest(macAddr, eip, broadcast, eip)
	toSockaddr := &syscall.SockaddrLinklayer{Ifindex: ifIndex}
	sock, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, ETH_P_ARP)
	if err != nil {
		return err
	}
	defer syscall.Close(sock)

	if err := syscall.BindToDevice(sock, ifName); err != nil {
		return errors.New("bind-err")
	}

	return syscall.Sendto(sock, request, 0, toSockaddr)
}

func newARPRequest(sha, spa, tha, tpa []byte) []byte {
	frame := bytes.NewBuffer(nil)
	// Ethernet header.
	frame.Write(tha)                                                 // Destination MAC address.
	frame.Write(sha)                                                 // Source MAC address.
	binary.Write(frame, binary.BigEndian, uint16(syscall.ETH_P_ARP)) // Ethernet protocol type, 0x0806 for ARP.
	// ARP message.
	binary.Write(frame, binary.BigEndian, uint16(1))                // Hardware Type, Ethernet is 1.
	binary.Write(frame, binary.BigEndian, uint16(syscall.ETH_P_IP)) // Protocol type, IPv4 is 0x0800.
	binary.Write(frame, binary.BigEndian, uint8(6))                 // Hardware length, Ethernet address length is 6.
	binary.Write(frame, binary.BigEndian, uint8(4))                 // Protocol length, IPv4 address length is 4.
	binary.Write(frame, binary.BigEndian, operationReply)           // Operation, request is 2.
	frame.Write(sha)                                                // Sender hardware address.
	frame.Write(spa)                                                // Sender protocol address.
	frame.Write(tha)                                                // Target hardware address.
	frame.Write(tpa)                                                // Target protocol address.
	return frame.Bytes()
}
