package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithAccessLogs_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})

	h := withAccessLogs(logger, base)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/foo?bar=baz", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	got := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(got, "GET /foo?bar=baz 201 2 ") {
		t.Fatalf("unexpected log line: %q", got)
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("ACCESS_LOGS", "true")
	if !envBool("ACCESS_LOGS") {
		t.Fatalf("expected envBool true")
	}
	if envBool("SOME_MISSING_VAR") {
		t.Fatalf("expected envBool false for missing var")
	}
}
