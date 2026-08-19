package repo

import (
	"bytes"
	"testing"
)

func TestRemoteLoggingLevelOneGuardsMissingTraceID(t *testing.T) {
	if !bytes.Contains(codebaseLoggingJs, []byte("sampled = (!traceId || !tracingLimitedID || toInt63(traceId) < tracingLimitedID)")) {
		t.Fatal("remote logging level 1 does not guard a missing trace ID")
	}
}

func TestOutboundLoggingDoesNotContainPerRequestDebugLog(t *testing.T) {
	if bytes.Contains(codebaseModulesOutboundLoggingHTTPJs, []byte("[DEBUG outbound]")) {
		t.Fatal("outbound logging contains an unconditional per-request debug log")
	}
}

func TestHTTPRoutingStreamsResponsesAtMessageStart(t *testing.T) {
	tests := []struct {
		name     string
		codebase []byte
	}{
		{name: "outbound routing", codebase: codebaseModulesOutboundHTTPRoutingJs},
		{name: "outbound load balancing", codebase: codebaseModulesOutboundHTTPLoadBalancingJs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if bytes.Contains(tt.codebase, []byte(".replaceMessage(")) {
				t.Fatal("response failover waits for the complete HTTP message")
			}
			if !bytes.Contains(tt.codebase, []byte(".replaceMessageStart(")) {
				t.Fatal("response failover is not evaluated when response headers arrive")
			}
		})
	}
}

func TestHTTPResponseBufferSizeIsConfigurable(t *testing.T) {
	want := []byte(".demuxHTTP({ bufferSize: httpResponseBufferSize })")
	for name, codebase := range map[string][]byte{
		"inbound routing":  codebaseModulesInboundHTTPRoutingJs,
		"outbound routing": codebaseModulesOutboundHTTPRoutingJs,
	} {
		t.Run(name, func(t *testing.T) {
			if !bytes.Contains(codebase, want) {
				t.Fatal("HTTP response buffer size is not passed to demuxHTTP")
			}
		})
	}
}
