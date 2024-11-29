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

func (h *DNSHandler) getSuffixDomains(trustDomain string) []string {
	suffixes := []string{fmt.Sprintf(`.svc.%s.`, trustDomain)}
	sections := strings.Split(trustDomain, `.`)
	for index := range sections {
		suffixes = append(suffixes, fmt.Sprintf(`.%s.`, strings.Join(sections[index:], `.`)))
	}
	return suffixes
}

func (h *DNSHandler) getTrustDomainSearches(trustDomain, namespace string) []string {
	searches := []string{fmt.Sprintf(`.svc.%s`, namespace)}
	sections := strings.Split(trustDomain, `.`)
	for index := range sections {
		searches = append(searches, fmt.Sprintf(`.svc.%s.%s`, strings.Join(sections[0:index+1], `.`), namespace))
	}
	searches = append(searches, fmt.Sprintf(`.%s`, namespace))
	return searches
}

func (h *DNSHandler) getRawQName(qname, trustDomain string) (string, string) {
	fromNamespace := `default`
	suffixDomains := h.getSuffixDomains(trustDomain)
	for _, suffixDomain := range suffixDomains {
		if strings.HasSuffix(qname, suffixDomain) {
			qname = strings.TrimSuffix(qname, suffixDomain)
			sections := strings.Split(qname, `.`)
			ndots := len(sections)
			if ndots > 1 {
				fromNamespace = sections[len(sections)-1]
				searches := h.getTrustDomainSearches(trustDomain, fromNamespace)
				for _, search := range searches {
					if strings.HasSuffix(qname, search) {
						qname = strings.TrimSuffix(qname, search)
						break
					}
				}
			}
			break
		}
	}
	return strings.TrimSuffix(qname, `.`), fromNamespace
}

func (h *DNSHandler) do(cfg *Config) {
	trustDomain := service.GetTrustDomain()
	for {
		data, ok := <-h.requestChannel
		if !ok {
			break
		}
		func(Net string, w dns.ResponseWriter, req *dns.Msg) {
			defer func(w dns.ResponseWriter) {
				_ = w.Close()
			}(w)

			var remote net.IP
			if Net == "tcp" {
				remote = w.RemoteAddr().(*net.TCPAddr).IP
			} else {
				remote = w.RemoteAddr().(*net.UDPAddr).IP
			}

			var origQuestions []dns.Question

			for index, q := range req.Question {
				origQuestions = append(origQuestions, q)
				qname, fromNamespace := h.getRawQName(q.Name, trustDomain)
				log.Debug().Msgf("%s lookup q.Name:%s qname:%s namespace:%s　trustDomain:%s", remote, q.Name, qname, fromNamespace, trustDomain)

				segs := strings.Split(qname, `.`)
				sections := len(segs)
				if sections == 1 { //internal domain name
					req.Question[index].Name = fmt.Sprintf(`%s.%s.svc.%s.`, segs[0], fromNamespace, trustDomain)
				} else if sections > 2 { //external domain name
					req.Question[index].Name = fmt.Sprintf(`%s.`, qname)
				} else {
					if k8sClient.GetK8sNamespace(segs[1]) != nil {
						req.Question[index].Name = fmt.Sprintf(`%s.%s.svc.%s.`, segs[0], segs[1], trustDomain)
					} else { //external domain name
						req.Question[index].Name = fmt.Sprintf(`%s.`, qname)
					}
				}
			}

			q := req.Question[0]
			Q := Question{UnFqdn(q.Name), dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass]}

			log.Debug().Msgf("%s lookup %s", remote, Q.String())

			ipQuery := h.isIPQuery(q)
			if ipQuery == 0 {
				m := new(dns.Msg)
				m.SetReply(req)
				m.SetRcode(req, dns.RcodeNameError)
				m.Question = origQuestions
				h.WriteReplyMsg(w, m)
				return
			}

			resp, err := h.resolver.Lookup(Net, req, cfg.GetTimeout(), cfg.GetInterval(), cfg.GetNameservers())
			if err != nil {
				log.Error().Msgf("resolve query error %s\n", err)
				req.Question = origQuestions
				h.HandleFailed(w, req)
				return
			}
			resp.Question = origQuestions

			if resp.Truncated && Net == "udp" {
				resp, err = h.resolver.Lookup("tcp", req, cfg.GetTimeout(), cfg.GetInterval(), cfg.GetNameservers())
				if err != nil {
					log.Error().Msgf("resolve tcp query error %s\n", err)
					h.HandleFailed(w, req)
					return
				}
			}

			log.Debug().Msgf("%s lookup　%s rcode:%d", remote, Q.String(), resp.Rcode)

			if resp.Rcode == dns.RcodeNameError && cfg.IsWildcard() {
				req.Question = origQuestions
				h.HandleWildcard(req, cfg, ipQuery, &q, w)
				return
			}

			if dbs := cfg.GetWildcardResolveDB(); cfg.IsWildcard() && len(dbs) > 0 {
				los := cfg.GetLoopbackResolveDB()
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
					case dns.TypeAAAA:
						resp.Answer = nil
						//aaaa := rr.(*dns.AAAA)
						//ip := aaaa.AAAA
						//if ip.IsUnspecified() || ip.IsLoopback() {
						//	for _, db := range dbs {
						//		if len(db.IPv6) > 0 {
						//			aaaa.AAAA = net.ParseIP(db.IPv6)
						//			resp.Answer[idx] = aaaa
						//			break
						//		}
						//	}
						//	break
						//}
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
