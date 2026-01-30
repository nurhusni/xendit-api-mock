package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	t.Setenv("SCENARIO_FILE", "")
	s := newServer()
	if s.seen == nil || s.attempts == nil || s.firstSeen == nil || s.accountIdx == nil {
		t.Fatal("expected server maps to be initialized")
	}
}

func TestRegisterRoutesHealth(t *testing.T) {
	s := newServer()
	mux := http.NewServeMux()
	registerRoutes(mux, s)
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz", nil)
	resp := httptest.NewRecorder()
	handleHealth(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleReset(t *testing.T) {
	s := &server{seen: map[string]bool{"a": true}, attempts: map[string]int{"a": 1}, firstSeen: make(map[string]time.Time), accountIdx: map[string]int{"a": 1}}
	req := httptest.NewRequest(http.MethodPost, "/xendit/reset", nil)
	resp := httptest.NewRecorder()
	s.handleReset(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if len(s.seen) != 0 || len(s.attempts) != 0 || len(s.accountIdx) != 0 {
		t.Fatalf("expected state reset")
	}
}

func TestHandleCallbackHealthMissingEnv(t *testing.T) {
	t.Setenv("CALLBACK_URL", "")
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz-callback", nil)
	resp := httptest.NewRecorder()
	handleCallbackHealth(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestHandleCallbackHealthSuccess(t *testing.T) {
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer callback.Close()

	t.Setenv("CALLBACK_URL", callback.URL)
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz-callback", nil)
	resp := httptest.NewRecorder()
	handleCallbackHealth(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleCreateDisbursementInvalidJSON(t *testing.T) {
	s := newServer()
	req := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader("{bad"))
	resp := httptest.NewRecorder()
	s.handleCreateDisbursement(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestHandleCreateDisbursementCallbackPayload(t *testing.T) {
	var payload callbackPayload
	var token string
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)
		token = r.Header.Get("X-Callback-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	t.Setenv("CALLBACK_URL", callback.URL)
	t.Setenv("CALLBACK_TOKEN", "token123")

	s := newServer()
	reqBody := `{"external_id":"ext-1","amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader(reqBody))
	resp := httptest.NewRecorder()
	s.handleCreateDisbursement(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if payload.Status != "FAILED" {
		t.Fatalf("expected FAILED on first attempt, got %s", payload.Status)
	}
	if token != "token123" {
		t.Fatalf("expected callback token, got %s", token)
	}
}

func TestHandleSimulateSuccessCallbackPayload(t *testing.T) {
	var payload callbackPayload
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	t.Setenv("CALLBACK_URL", callback.URL)

	s := newServer()
	req := httptest.NewRequest(http.MethodPost, "/xendit/simulate/success", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	s.handleSimulateSuccess(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if payload.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", payload.Status)
	}
}

func TestSendCallbackMissingURL(t *testing.T) {
	t.Setenv("CALLBACK_URL", "")
	if err := sendCallback(disbursementRequest{ExternalID: "x"}, "COMPLETED"); err == nil {
		t.Fatal("expected error when CALLBACK_URL is missing")
	}
}

func TestShortHash(t *testing.T) {
	if got := shortHash("abc"); got != shortHash("abc") {
		t.Fatalf("expected stable hash, got %s", got)
	}
	if len(shortHash("abc")) != 8 {
		t.Fatalf("expected 8-char hash")
	}
}
