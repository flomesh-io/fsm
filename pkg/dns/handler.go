package dns

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"

	"github.com/flomesh-io/fsm/pkg/service"
	"github.com/flomesh-io/fsm/pkg/utils"
)

const (
	notIPQuery = 0
	_IP4Query  = 4
	_IP6Query  = 6
)

// Question type
type Question struct {
	Qname  string `json:"name"`
	Qtype  string `json:"type"`
	Qclass string `json:"class"`
}

// QuestionCacheEntry represents a full query from a client with metadata
type QuestionCacheEntry struct {
	Date    int64    `json:"date"`
	Remote  string   `json:"client"`
	Blocked bool     `json:"blocked"`
	Query   Question `json:"query"`
}

// String formats a question
func (q *Question) String() string {
	return q.Qname + " " + q.Qclass + " " + q.Qtype
}

// DNSHandler type
type DNSHandler struct {
	requestChannel chan DNSOperationData
	resolver       *Resolver
	active         bool
	muActive       sync.RWMutex
}

// DNSOperationData type
type DNSOperationData struct {
	Net string
	w   dns.ResponseWriter
	req *dns.Msg
}

// NewHandler returns a new DNSHandler
func NewHandler(config *Config) *DNSHandler {
	var (
		clientConfig *dns.ClientConfig
		resolver     *Resolver
	)

	resolver = &Resolver{clientConfig}

	handler := &DNSHandler{
		requestChannel: make(chan DNSOperationData),
		resolver:       resolver,
		active:         true,
	}

	go handler.do(config)

	return handler
}

func (h *DNSHandler) do(cfg *Config) {
	trustDomain := service.GetTrustDomain()
	suffixDomain := fmt.Sprintf(".svc.%s.", trustDomain)
	for {
		data, ok := <-h.requestChannel
		if !ok {
			break
		}
		func(Net string, w dns.ResponseWriter, req *dns.Msg) {
			defer func(w dns.ResponseWriter) {
				_ = w.Close()
			}(w)

			for index, q := range req.Question {
				if strings.HasSuffix(q.Name, suffixDomain) {
					if segs := strings.Split(q.Name, "."); len(segs) == 7 {
						req.Question[index].Name = fmt.Sprintf("%s.%s.svc.%s.", segs[0], segs[1], trustDomain)
					}
				}
			}

			q := req.Question[0]
			Q := Question{UnFqdn(q.Name), dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass]}

			var remote net.IP
			if Net == "tcp" {
				remote = w.RemoteAddr().(*net.TCPAddr).IP
			} else {
				remote = w.RemoteAddr().(*net.UDPAddr).IP
			}

			log.Info().Msgf("%s lookup　%s\n", remote, Q.String())

			ipQuery := h.isIPQuery(q)
			if ipQuery == 0 {
				m := new(dns.Msg)
				m.SetReply(req)
				m.SetRcode(req, dns.RcodeNameError)
				h.WriteReplyMsg(w, m)
				return
			}

			resp, err := h.resolver.Lookup(Net, req, cfg.GetTimeout(), cfg.GetInterval(), cfg.GetNameservers())
			if err != nil {
				log.Error().Msgf("resolve query error %s\n", err)
				h.HandleFailed(w, req)
				return
			}

			if resp.Truncated && Net == "udp" {
				resp, err = h.resolver.Lookup("tcp", req, cfg.GetTimeout(), cfg.GetInterval(), cfg.GetNameservers())
				if err != nil {
					log.Error().Msgf("resolve tcp query error %s\n", err)
					h.HandleFailed(w, req)
					return
				}
			}

			if resp.Rcode == dns.RcodeNameError && cfg.IsWildcard() {
				h.HandleWildcard(req, cfg, ipQuery, &q, w)
				return
			}

			if cfg.IsWildcard() && len(cfg.GetLoopbackResolveDB()) > 0 {
				los := cfg.GetLoopbackResolveDB()
				dbs := cfg.GetWildcardResolveDB()
				for idx, rr := range resp.Answer {
					header := rr.Header()
					switch header.Rrtype {
					case dns.TypeA:
						a := rr.(*dns.A)
						ip := a.A
						for _, lo := range los {
							if strings.EqualFold(ip.String(), lo.IPv4) {
								for _, db := range dbs {
									if len(db.IPv4) > 0 {
										a.A = net.ParseIP(db.IPv4)
										resp.Answer[idx] = a
										break
									}
								}
								break
							}
						}
					}
				}
			}

			h.WriteReplyMsg(w, resp)
		}(data.Net, data.w, data.req)
	}
}

// DoTCP begins a tcp query
func (h *DNSHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	h.muActive.RLock()
	if h.active {
		h.requestChannel <- DNSOperationData{"tcp", w, req}
	}
	h.muActive.RUnlock()
}

// DoUDP begins a udp query
func (h *DNSHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	h.muActive.RLock()
	if h.active {
		h.requestChannel <- DNSOperationData{"udp", w, req}
	}
	h.muActive.RUnlock()
}

// HandleFailed handles dns failures
func (h *DNSHandler) HandleFailed(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.SetRcode(req, dns.RcodeServerFailure)
	h.WriteReplyMsg(w, m)
}

func (h *DNSHandler) HandleWildcard(req *dns.Msg, cfg *Config, ipQuery int, q *dns.Question, w dns.ResponseWriter) {
	m := new(dns.Msg)
	m.SetReply(req)

	if cfg.GetNXDomain() {
		m.SetRcode(req, dns.RcodeNameError)
	} else {
		dbs := cfg.GetWildcardResolveDB()
		switch ipQuery {
		case _IP4Query:
			rrHeader := dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    0,
			}
			for _, db := range dbs {
				if len(db.IPv4) == 0 {
					continue
				}
				a := &dns.A{Hdr: rrHeader, A: net.ParseIP(db.IPv4)}
				m.Answer = append(m.Answer, a)
			}
		case _IP6Query:
			rrHeader := dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    0,
			}
			for _, db := range dbs {
				if len(db.IPv6) > 0 {
					a := &dns.AAAA{Hdr: rrHeader, AAAA: net.ParseIP(db.IPv6)}
					m.Answer = append(m.Answer, a)
				} else if len(db.IPv4) > 0 && cfg.GenerateIPv6BasedOnIPv4() {
					a := &dns.AAAA{Hdr: rrHeader, AAAA: net.ParseIP(utils.IPv4Tov6(db.IPv4))}
					m.Answer = append(m.Answer, a)
				}
			}
		}
	}
	h.WriteReplyMsg(w, m)
}

// WriteReplyMsg writes the dns reply
func (h *DNSHandler) WriteReplyMsg(w dns.ResponseWriter, message *dns.Msg) {
	defer func() {
		if r := recover(); r != nil {
			log.Info().Msgf("Recovered in WriteReplyMsg: %s\n", r)
		}
	}()

	err := w.WriteMsg(message)
	if err != nil {
		log.Error().Err(err).Msg(err.Error())
	}
}

func (h *DNSHandler) isIPQuery(q dns.Question) int {
	if q.Qclass != dns.ClassINET {
		return notIPQuery
	}

	switch q.Qtype {
	case dns.TypeA:
		return _IP4Query
	case dns.TypeAAAA:
		return _IP6Query
	default:
		return notIPQuery
	}
}

// UnFqdn function
func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}
