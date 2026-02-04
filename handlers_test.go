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

	"xendit-api-mock/internal/callback"
	"xendit-api-mock/internal/domain"
	"xendit-api-mock/internal/scenario"
	"xendit-api-mock/internal/service/disbursement"
	httptransport "xendit-api-mock/internal/transport/http"
)

func newTestHandler() *httptransport.Handler {
	engine := scenario.NewEngine(nil)
	callbackURL := getenv("CALLBACK_URL", "")
	callbackToken := getenv("CALLBACK_TOKEN", "")
	userID := getenv("XENDIT_USER_ID", "user_mock")
	cbClient := callback.NewClient(callbackURL, callbackToken, nil)
	service := disbursement.NewService(engine, cbClient, userID)
	return httptransport.NewHandler(service, callbackURL)
}

func TestNewHandler(t *testing.T) {
	t.Setenv("SCENARIO_FILE", "")
	if handler := newTestHandler(); handler == nil {
		t.Fatal("expected handler to be initialized")
	}
}

func TestRegisterRoutesHealth(t *testing.T) {
	if handler := newTestHandler(); handler == nil {
		t.Fatal("expected handler to be initialized")
	}
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleReset(t *testing.T) {
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	t.Setenv("CALLBACK_URL", callback.URL)
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)

	reqBody := `{"external_id":"ext-1","amount":100}`
	createReq := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader(reqBody))
	createResp := httptest.NewRecorder()
	mux.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", createResp.Code)
	}
	var firstResp domain.DisbursementResponse
	if err := json.Unmarshal(createResp.Body.Bytes(), &firstResp); err != nil {
		t.Fatalf("expected json response, got %v", err)
	}
	if firstResp.Status != "FAILED" {
		t.Fatalf("expected FAILED before reset, got %s", firstResp.Status)
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/xendit/reset", nil)
	resetResp := httptest.NewRecorder()
	mux.ServeHTTP(resetResp, resetReq)
	if resetResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resetResp.Code)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader(reqBody))
	secondResp := httptest.NewRecorder()
	mux.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", secondResp.Code)
	}
	var secondBody domain.DisbursementResponse
	if err := json.Unmarshal(secondResp.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("expected json response, got %v", err)
	}
	if secondBody.Status != "FAILED" {
		t.Fatalf("expected FAILED after reset, got %s", secondBody.Status)
	}
}

func TestHandleCallbackHealthMissingEnv(t *testing.T) {
	t.Setenv("CALLBACK_URL", "")
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz-callback", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
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
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/xendit/healthz-callback", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHandleCreateDisbursementInvalidJSON(t *testing.T) {
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader("{bad"))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestHandleCreateDisbursementCallbackPayload(t *testing.T) {
	var payload domain.CallbackPayload
	var token string
	callbackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)
		token = r.Header.Get("X-Callback-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer callbackSrv.Close()

	t.Setenv("CALLBACK_URL", callbackSrv.URL)
	t.Setenv("CALLBACK_TOKEN", "token123")

	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	reqBody := `{"external_id":"ext-1","amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader(reqBody))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
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
	var payload domain.CallbackPayload
	callbackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer callbackSrv.Close()

	t.Setenv("CALLBACK_URL", callbackSrv.URL)
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/xendit/simulate/success", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if payload.Status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", payload.Status)
	}
}

func TestHandleCreateDisbursementResponseBody(t *testing.T) {
	callbackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer callbackSrv.Close()

	t.Setenv("CALLBACK_URL", callbackSrv.URL)
	t.Setenv("XENDIT_USER_ID", "xamock-user")

	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	reqBody := `{"external_id":"ext-123","amount":9000,"bank_code":"BNI","account_holder_name":"Mock User","account_number":"123","description":"topup-123"}`
	req := httptest.NewRequest(http.MethodPost, "/xendit/disbursements", strings.NewReader(reqBody))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content-type, got %s", got)
	}

	var payload domain.DisbursementResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected json body, got error %v", err)
	}
	if payload.ExternalID != "ext-123" {
		t.Fatalf("expected external_id ext-123, got %s", payload.ExternalID)
	}
	if payload.UserID != "xamock-user" {
		t.Fatalf("expected user_id xamock-user, got %s", payload.UserID)
	}
	if payload.Amount != 9000 {
		t.Fatalf("expected amount 9000, got %d", payload.Amount)
	}
	if payload.BankCode != "BNI" {
		t.Fatalf("expected bank_code BNI, got %s", payload.BankCode)
	}
	if payload.AccountHolderName != "Mock User" {
		t.Fatalf("expected account_holder_name Mock User, got %s", payload.AccountHolderName)
	}
	if payload.DisbursementDescription != "topup-123" {
		t.Fatalf("expected disbursement_description topup-123, got %s", payload.DisbursementDescription)
	}
	if payload.Status != "FAILED" {
		t.Fatalf("expected status FAILED on first attempt, got %s", payload.Status)
	}
	if payload.ID == "" || !strings.HasPrefix(payload.ID, "disb_") {
		t.Fatalf("expected id with disb_ prefix, got %s", payload.ID)
	}
	if _, err := time.Parse(time.RFC3339, payload.Created); err != nil {
		t.Fatalf("expected created to be RFC3339, got %s", payload.Created)
	}
	if _, err := time.Parse(time.RFC3339, payload.Updated); err != nil {
		t.Fatalf("expected updated to be RFC3339, got %s", payload.Updated)
	}
}

func TestHandleSimulateSuccessResponseBody(t *testing.T) {
	callbackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer callbackSrv.Close()

	t.Setenv("CALLBACK_URL", callbackSrv.URL)
	mux := http.NewServeMux()
	newTestHandler().RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/xendit/simulate/success", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var payload domain.DisbursementResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected json body, got error %v", err)
	}
	if payload.Status != "COMPLETED" {
		t.Fatalf("expected status COMPLETED, got %s", payload.Status)
	}
}

func TestShortHash(t *testing.T) {
	if got := domain.ShortHash("abc"); got != domain.ShortHash("abc") {
		t.Fatalf("expected stable hash, got %s", got)
	}
	if len(domain.ShortHash("abc")) != 8 {
		t.Fatalf("expected 8-char hash")
	}
}
