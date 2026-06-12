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
