package route

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

const (
	requestOper  = 1
	responseOper = 2

	timeout = time.Duration(500 * time.Millisecond)
)

type arpDatagram struct {
	htype uint16 // Hardware Type
	ptype uint16 // Protocol Type
	hlen  uint8  // Hardware address Length
	plen  uint8  // Protocol address length
	oper  uint16 // Operation 1->request, 2->response
	sha   []byte // Sender hardware address, length from Hlen
	spa   []byte // Sender protocol address, length from Plen
	tha   []byte // Target hardware address, length from Hlen
	tpa   []byte // Target protocol address, length from Plen
}

func newArpRequest(
	srcMac net.HardwareAddr,
	srcIP net.IP,
	dstMac net.HardwareAddr,
	dstIP net.IP) arpDatagram {
	return arpDatagram{
		htype: uint16(1),
		ptype: uint16(0x0800),
		hlen:  uint8(6),
		plen:  uint8(4),
		oper:  uint16(requestOper),
		sha:   srcMac,
		spa:   srcIP.To4(),
		tha:   dstMac,
		tpa:   dstIP.To4()}
}

func (datagram arpDatagram) Marshal() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, datagram.htype)
	binary.Write(buf, binary.BigEndian, datagram.ptype)
	binary.Write(buf, binary.BigEndian, datagram.hlen)
	binary.Write(buf, binary.BigEndian, datagram.plen)
	binary.Write(buf, binary.BigEndian, datagram.oper)
	buf.Write(datagram.sha)
	buf.Write(datagram.spa)
	buf.Write(datagram.tha)
	buf.Write(datagram.tpa)

	return buf.Bytes()
}

func (datagram arpDatagram) MarshalWithEthernetHeader() []byte {
	// ethernet frame header
	var ethernetHeader []byte
	ethernetHeader = append(ethernetHeader, datagram.tha...)
	ethernetHeader = append(ethernetHeader, datagram.sha...)
	ethernetHeader = append(ethernetHeader, []byte{0x08, 0x06}...) // arp

	return append(ethernetHeader, datagram.Marshal()...)
}

func (datagram arpDatagram) SenderIP() net.IP {
	return net.IP(datagram.spa)
}
func (datagram arpDatagram) SenderMac() net.HardwareAddr {
	return net.HardwareAddr(datagram.sha)
}

func (datagram arpDatagram) IsResponseOf(request arpDatagram) bool {
	return datagram.oper == responseOper && bytes.Equal(request.spa, datagram.tpa) &&
		bytes.Equal(request.tpa, datagram.spa)
}

func parseArpDatagram(buffer []byte) arpDatagram {
	var datagram arpDatagram

	b := bytes.NewBuffer(buffer)
	binary.Read(b, binary.BigEndian, &datagram.htype)
	binary.Read(b, binary.BigEndian, &datagram.ptype)
	binary.Read(b, binary.BigEndian, &datagram.hlen)
	binary.Read(b, binary.BigEndian, &datagram.plen)
	binary.Read(b, binary.BigEndian, &datagram.oper)

	haLen := int(datagram.hlen)
	paLen := int(datagram.plen)
	datagram.sha = b.Next(haLen)
	datagram.spa = b.Next(paLen)
	datagram.tha = b.Next(haLen)
	datagram.tpa = b.Next(paLen)

	return datagram
}

type LinuxSocket struct {
	sock       int
	toSockaddr syscall.SockaddrLinklayer
}

func initialize(iface *net.Interface) (s *LinuxSocket, err error) {
	s = &LinuxSocket{}
	s.toSockaddr = syscall.SockaddrLinklayer{Ifindex: iface.Index}

	// 1544 = htons(ETH_P_ARP)
	const proto = 1544
	s.sock, err = syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, proto)
	return s, err
}

func (s *LinuxSocket) send(request arpDatagram) (time.Time, error) {
	return time.Now(), syscall.Sendto(s.sock, request.MarshalWithEthernetHeader(), 0, &s.toSockaddr)
}

func (s *LinuxSocket) receive() (arpDatagram, time.Time, error) {
	buffer := make([]byte, 128)
	socketTimeout := timeout.Nanoseconds() * 2
	t := syscall.NsecToTimeval(socketTimeout)
	syscall.SetsockoptTimeval(s.sock, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &t)
	n, _, err := syscall.Recvfrom(s.sock, buffer, 0)
	if err != nil {
		return arpDatagram{}, time.Now(), err
	}
	if n <= 14 {
		// amount of bytes read by socket is less than an ethernet header. clearly not what we look for
		return arpDatagram{}, time.Now(), fmt.Errorf("buffer with invalid length")
	}
	// skip 14 bytes ethernet header
	return parseArpDatagram(buffer[14:n]), time.Now(), nil
}

func (s *LinuxSocket) deinitialize() error {
	return syscall.Close(s.sock)
}

// ARPing sends an arp ping over interface 'iface' to 'dstIP'
func ARPing(srcIP, dstIP net.IP, iface *net.Interface) (net.HardwareAddr, error) {
	srcMac := iface.HardwareAddr

	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	request := newArpRequest(srcMac, srcIP, broadcastMac, dstIP)

	sock, err := initialize(iface)
	if err != nil {
		return nil, err
	}
	defer sock.deinitialize()

	type PingResult struct {
		mac net.HardwareAddr
		err error
	}
	pingResultChan := make(chan PingResult, 1)

	go func() {
		// send arp request
		if _, err := sock.send(request); err != nil {
			pingResultChan <- PingResult{nil, err}
		} else {
			for {
				// receive arp response
				response, _, err := sock.receive()

				if err != nil {
					pingResultChan <- PingResult{nil, err}
					return
				}

				if response.IsResponseOf(request) {
					pingResultChan <- PingResult{response.SenderMac(), err}
					return
				}
			}
		}
	}()

	select {
	case pingResult := <-pingResultChan:
		return pingResult.mac, pingResult.err
	case <-time.After(timeout):
		sock.deinitialize()
		return nil, errors.New("arping timeout")
	}
}
