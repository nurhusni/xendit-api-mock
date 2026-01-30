package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatBody(t *testing.T) {
	if got := formatBody(nil); got != "{empty}" {
		t.Fatalf("expected {empty}, got %s", got)
	}
	jsonBody := []byte(`{"a":1}`)
	formatted := formatBody(jsonBody)
	if !strings.Contains(formatted, "\"a\": 1") {
		t.Fatalf("expected formatted json, got %s", formatted)
	}
	nonJSON := []byte("plain text")
	if got := formatBody(nonJSON); got != "plain text" {
		t.Fatalf("expected raw body, got %s", got)
	}
}

func TestLoggingHandlerLogsRequestAndResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	orig := log.Writer()
	log.SetOutput(buf)
	defer log.SetOutput(orig)

	handler := loggingHandler("testHandler", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("nope"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/xendit/test", strings.NewReader(`{"x":1}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	logs := buf.String()
	if !strings.Contains(logs, "[testHandler.logRequest]") {
		t.Fatalf("expected logRequest tag, got %s", logs)
	}
	if !strings.Contains(logs, "[testHandler.logResponse]") {
		t.Fatalf("expected logResponse tag, got %s", logs)
	}
	if !strings.Contains(logs, "response error") {
		t.Fatalf("expected response error log, got %s", logs)
	}
}
