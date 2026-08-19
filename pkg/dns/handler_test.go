package dns

import (
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/miekg/dns"
	corev1 "k8s.io/api/core/v1"

	configv1alpha3 "github.com/flomesh-io/fsm/pkg/apis/config/v1alpha3"
	"github.com/flomesh-io/fsm/pkg/configurator"
	"github.com/flomesh-io/fsm/pkg/k8s"
)

const (
	searchExpandedQuestion   = "box-e.be.iflorens-test.svc.cluster.local."
	absoluteExternalQuestion = "box-e.be."
	shortServiceQuestion     = "service.unit1."
	forwardedServiceQuestion = "service.unit1.svc.cluster.local."
)

type fakeResolver struct {
	t         *testing.T
	want      string
	responses map[string]func(*dns.Msg) *dns.Msg
	calls     []string
}

func (r *fakeResolver) Lookup(network string, req *dns.Msg, _ int, _ int, _ []string) (*dns.Msg, error) {
	r.calls = append(r.calls, network)
	if got := req.Question[0].Name; got != r.want {
		r.t.Errorf("forwarded question = %q, want %q", got, r.want)
	}
	return r.responses[network](req), nil
}

type captureResponseWriter struct {
	responses chan *dns.Msg
}

func (w *captureResponseWriter) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 15053}
}

func (w *captureResponseWriter) RemoteAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.2"), Port: 53000}
}

func (w *captureResponseWriter) WriteMsg(msg *dns.Msg) error {
	wire, err := msg.Pack()
	if err != nil {
		return err
	}
	resp := new(dns.Msg)
	if err := resp.Unpack(wire); err != nil {
		return err
	}
	w.responses <- resp
	return nil
}

func (w *captureResponseWriter) Write(data []byte) (int, error) { return len(data), nil }
func (w *captureResponseWriter) Close() error                   { return nil }
func (w *captureResponseWriter) TsigStatus() error              { return nil }
func (w *captureResponseWriter) TsigTimersOnly(bool)            {}
func (w *captureResponseWriter) Hijack()                        {}

func TestDNSHandlerRestoresEveryAnswerOwnerForSearchQuery(t *testing.T) {
	tests := []struct {
		name  string
		qtype uint16
		make  func(string) dns.RR
	}{
		{
			name:  "multiple A answers",
			qtype: dns.TypeA,
			make: func(ip string) dns.RR {
				return &dns.A{
					Hdr: dns.RR_Header{Name: forwardedServiceQuestion, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
					A:   net.ParseIP(ip),
				}
			},
		},
		{
			name:  "multiple AAAA answers",
			qtype: dns.TypeAAAA,
			make: func(ip string) dns.RR {
				return &dns.AAAA{
					Hdr:  dns.RR_Header{Name: forwardedServiceQuestion, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
					AAAA: net.ParseIP(ip),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installK8sController(t, "unit1", true)
			resolver := &fakeResolver{
				t:    t,
				want: forwardedServiceQuestion,
				responses: map[string]func(*dns.Msg) *dns.Msg{
					"udp": func(req *dns.Msg) *dns.Msg {
						resp := new(dns.Msg)
						resp.SetReply(req)
						resp.Answer = []dns.RR{
							withOwner(tt.make("192.0.2.10"), forwardedServiceQuestion),
							withOwner(tt.make("192.0.2.11"), forwardedServiceQuestion),
						}
						return resp
					},
				},
			}

			resp := exchange(t, newTestConfig(t, false, nil), resolver, shortServiceQuestion, tt.qtype)
			assertQuestionName(t, resp, shortServiceQuestion)
			for i, answer := range resp.Answer {
				if got := answer.Header().Name; got != shortServiceQuestion {
					t.Errorf("answer[%d] owner = %q, want %q", i, got, shortServiceQuestion)
				}
			}
		})
	}
}

func TestDNSHandlerRestoresCNAMEOwnerWithoutChangingTargetAnswer(t *testing.T) {
	installK8sController(t, "unit1", true)
	const alias = "alias.example."
	resolver := &fakeResolver{
		t:    t,
		want: forwardedServiceQuestion,
		responses: map[string]func(*dns.Msg) *dns.Msg{
			"udp": func(req *dns.Msg) *dns.Msg {
				resp := new(dns.Msg)
				resp.SetReply(req)
				resp.Answer = []dns.RR{
					&dns.CNAME{
						Hdr:    dns.RR_Header{Name: forwardedServiceQuestion, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 60},
						Target: alias,
					},
					&dns.A{
						Hdr: dns.RR_Header{Name: alias, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
						A:   net.ParseIP("192.0.2.10"),
					},
				}
				return resp
			},
		},
	}

	resp := exchange(t, newTestConfig(t, false, nil), resolver, shortServiceQuestion, dns.TypeA)
	assertQuestionName(t, resp, shortServiceQuestion)
	if got := resp.Answer[0].Header().Name; got != shortServiceQuestion {
		t.Errorf("CNAME owner = %q, want %q", got, shortServiceQuestion)
	}
	if got := resp.Answer[1].Header().Name; got != alias {
		t.Errorf("CNAME target answer owner = %q, want %q", got, alias)
	}
}

func TestDNSHandlerReturnsNXDomainForSearchExpandedExternalName(t *testing.T) {
	tests := []string{
		searchExpandedQuestion,
		"box-e.be.svc.cluster.local.",
		"box-e.be.cluster.local.",
	}

	for _, question := range tests {
		t.Run(question, func(t *testing.T) {
			installK8sController(t, "be", false)
			resolver := &fakeResolver{
				t:    t,
				want: absoluteExternalQuestion,
				responses: map[string]func(*dns.Msg) *dns.Msg{
					"udp": func(req *dns.Msg) *dns.Msg {
						resp := new(dns.Msg)
						resp.SetReply(req)
						resp.Answer = []dns.RR{
							&dns.A{Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("192.0.2.10")},
						}
						return resp
					},
				},
			}
			wildcardRecords := []*configv1alpha3.ResolveAddr{{IPv4: "192.168.33.1"}}

			resp := exchange(t, newTestConfig(t, true, wildcardRecords), resolver, question, dns.TypeA)
			assertQuestionName(t, resp, question)
			if resp.Rcode != dns.RcodeNameError {
				t.Errorf("response rcode = %s, want NXDOMAIN", dns.RcodeToString[resp.Rcode])
			}
			if len(resp.Answer) != 0 {
				t.Errorf("answer count = %d, want 0", len(resp.Answer))
			}
			if len(resolver.calls) != 0 {
				t.Errorf("resolver calls = %v, want no upstream lookup", resolver.calls)
			}
		})
	}
}

func TestDNSHandlerResolvesFinalAbsoluteExternalName(t *testing.T) {
	installK8sController(t, "be", false)
	resolver := &fakeResolver{
		t:    t,
		want: absoluteExternalQuestion,
		responses: map[string]func(*dns.Msg) *dns.Msg{
			"udp": func(req *dns.Msg) *dns.Msg {
				resp := new(dns.Msg)
				resp.SetReply(req)
				resp.Answer = []dns.RR{
					&dns.A{Hdr: dns.RR_Header{Name: absoluteExternalQuestion, Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("192.0.2.10")},
					&dns.A{Hdr: dns.RR_Header{Name: absoluteExternalQuestion, Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("192.0.2.11")},
				}
				return resp
			},
		},
	}

	resp := exchange(t, newTestConfig(t, true, nil), resolver, absoluteExternalQuestion, dns.TypeA)
	assertQuestionName(t, resp, absoluteExternalQuestion)
	for i, answer := range resp.Answer {
		if got := answer.Header().Name; got != absoluteExternalQuestion {
			t.Errorf("answer[%d] owner = %q, want %q", i, got, absoluteExternalQuestion)
		}
	}
}

func TestDNSHandlerKeepsFullyQualifiedClusterServiceQuery(t *testing.T) {
	const serviceQuestion = "service.unit1.svc.cluster.local."
	installK8sController(t, "unit1", true)
	resolver := &fakeResolver{
		t:    t,
		want: serviceQuestion,
		responses: map[string]func(*dns.Msg) *dns.Msg{
			"udp": func(req *dns.Msg) *dns.Msg {
				resp := new(dns.Msg)
				resp.SetReply(req)
				resp.Answer = []dns.RR{
					&dns.A{Hdr: dns.RR_Header{Name: serviceQuestion, Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("10.0.0.10")},
				}
				return resp
			},
		},
	}

	resp := exchange(t, newTestConfig(t, false, nil), resolver, serviceQuestion, dns.TypeA)
	assertQuestionName(t, resp, serviceQuestion)
	if resp.Rcode != dns.RcodeSuccess {
		t.Errorf("response rcode = %s, want NOERROR", dns.RcodeToString[resp.Rcode])
	}
	if len(resp.Answer) != 1 || resp.Answer[0].Header().Name != serviceQuestion {
		t.Errorf("answers = %v, want one answer for %q", resp.Answer, serviceQuestion)
	}
}

func TestDNSHandlerRestoresNamesAfterTCPFallback(t *testing.T) {
	installK8sController(t, "unit1", true)
	resolver := &fakeResolver{
		t:    t,
		want: forwardedServiceQuestion,
		responses: map[string]func(*dns.Msg) *dns.Msg{
			"udp": func(req *dns.Msg) *dns.Msg {
				resp := new(dns.Msg)
				resp.SetReply(req)
				resp.Truncated = true
				return resp
			},
			"tcp": func(req *dns.Msg) *dns.Msg {
				resp := new(dns.Msg)
				resp.SetReply(req)
				resp.Answer = []dns.RR{
					&dns.A{Hdr: dns.RR_Header{Name: forwardedServiceQuestion, Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("192.0.2.10")},
					&dns.A{Hdr: dns.RR_Header{Name: forwardedServiceQuestion, Rrtype: dns.TypeA, Class: dns.ClassINET}, A: net.ParseIP("192.0.2.11")},
				}
				return resp
			},
		},
	}

	resp := exchange(t, newTestConfig(t, false, nil), resolver, shortServiceQuestion, dns.TypeA)
	if len(resolver.calls) != 2 || resolver.calls[0] != "udp" || resolver.calls[1] != "tcp" {
		t.Fatalf("resolver calls = %v, want [udp tcp]", resolver.calls)
	}
	assertQuestionName(t, resp, shortServiceQuestion)
	for i, answer := range resp.Answer {
		if got := answer.Header().Name; got != shortServiceQuestion {
			t.Errorf("answer[%d] owner = %q, want %q", i, got, shortServiceQuestion)
		}
	}
}

func exchange(t *testing.T, config *Config, resolver dnsResolver, question string, qtype uint16) *dns.Msg {
	t.Helper()
	handler := newHandler(config, resolver)
	t.Cleanup(func() {
		handler.muActive.Lock()
		if handler.active {
			handler.active = false
			close(handler.requestChannel)
		}
		handler.muActive.Unlock()
	})

	writer := &captureResponseWriter{responses: make(chan *dns.Msg, 1)}
	req := new(dns.Msg)
	req.SetQuestion(question, qtype)
	handler.DoUDP(writer, req)

	select {
	case resp := <-writer.responses:
		return resp
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for DNS response")
		return nil
	}
}

func newTestConfig(t *testing.T, wildcard bool, records []*configv1alpha3.ResolveAddr) *Config {
	t.Helper()
	mockConfig := configurator.NewMockConfigurator(gomock.NewController(t))
	mockConfig.EXPECT().GetLocalDNSProxyPrimaryUpstream().Return("").AnyTimes()
	mockConfig.EXPECT().GetLocalDNSProxySecondaryUpstream().Return("").AnyTimes()
	mockConfig.EXPECT().IsWildcardDNSProxyEnabled().Return(wildcard).AnyTimes()
	meshConfig := configv1alpha3.MeshConfig{}
	meshConfig.Spec.Sidecar.LocalDNSProxy.Wildcard.IPs = records
	mockConfig.EXPECT().GetMeshConfig().Return(meshConfig).AnyTimes()
	return &Config{cfg: mockConfig}
}

func installK8sController(t *testing.T, namespace string, exists bool) {
	t.Helper()
	originalController := k8sClient
	mockController := k8s.NewMockController(gomock.NewController(t))
	var result *corev1.Namespace
	if exists {
		result = &corev1.Namespace{}
	}
	mockController.EXPECT().GetK8sNamespace(namespace).Return(result).AnyTimes()
	k8sClient = mockController
	t.Cleanup(func() {
		k8sClient = originalController
	})
}

func withOwner(rr dns.RR, owner string) dns.RR {
	rr.Header().Name = owner
	return rr
}

func assertQuestionName(t *testing.T, resp *dns.Msg, want string) {
	t.Helper()
	if len(resp.Question) != 1 {
		t.Fatalf("question count = %d, want 1", len(resp.Question))
	}
	if got := resp.Question[0].Name; got != want {
		t.Errorf("response question = %q, want %q", got, want)
	}
}
