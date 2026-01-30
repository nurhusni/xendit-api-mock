package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type disbursementRequest struct {
	ExternalID        string `json:"external_id"`
	Amount            int    `json:"amount"`
	BankCode          string `json:"bank_code"`
	AccountHolderName string `json:"account_holder_name"`
	AccountNumber     string `json:"account_number"`
	Description       string `json:"description"`
}

type disbursementResponse struct {
	ID                      string `json:"id"`
	UserID                  string `json:"user_id"`
	ExternalID              string `json:"external_id"`
	Amount                  int    `json:"amount"`
	BankCode                string `json:"bank_code"`
	AccountHolderName       string `json:"account_holder_name"`
	DisbursementDescription string `json:"disbursement_description"`
	Status                  string `json:"status"`
	Created                 string `json:"created"`
	Updated                 string `json:"updated"`
}

type callbackPayload struct {
	ID                      string `json:"id"`
	Created                 string `json:"created"`
	Updated                 string `json:"updated"`
	ExternalID              string `json:"external_id"`
	UserID                  string `json:"user_id"`
	Amount                  int    `json:"amount"`
	BankCode                string `json:"bank_code"`
	AccountHolderName       string `json:"account_holder_name"`
	AccountNumber           string `json:"account_number"`
	DisbursementDescription string `json:"disbursement_description"`
	Status                  string `json:"status"`
	IsInstant               bool   `json:"is_instant"`
	WebhookID               string `json:"webhookId"`
}

type server struct {
	mu         sync.Mutex
	firstFail  bool
	seen       map[string]bool
	attempts   map[string]int
	firstSeen  map[string]time.Time
	accountIdx map[string]int
	scenario   *scenarioConfig
}

func newServer() *server {
	return &server{
		seen:       make(map[string]bool),
		attempts:   make(map[string]int),
		firstSeen:  make(map[string]time.Time),
		accountIdx: make(map[string]int),
		scenario:   loadScenario(getenv("SCENARIO_FILE", "")),
	}
}

func registerRoutes(s *server) {
	http.HandleFunc("/xendit/disbursements", s.handleCreateDisbursement)
	http.HandleFunc("/xendit/healthz", handleHealth)
	http.HandleFunc("/xendit/healthz-callback", handleCallbackHealth)
	http.HandleFunc("/xendit/simulate/success", s.handleSimulateSuccess)
	http.HandleFunc("/xendit/reset", s.handleReset)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleCallbackHealth(w http.ResponseWriter, r *http.Request) {
	callbackURL := "https://sandbox.api.of.ayoconnect.id/api/v1/it/xendit/disbursement/callback"
	req, err := http.NewRequest(http.MethodPost, callbackURL, nil)
	if err != nil {
		log.Printf("callback health request build failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("callback health request failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("callback health response read failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}

	if len(body) == 0 {
		log.Printf("callback health response status=%d body={empty}", resp.StatusCode)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if json.Valid(body) {
		var payload interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("callback health response status=%d body=%s", resp.StatusCode, string(body))
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		pretty, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			log.Printf("callback health response status=%d body=%s", resp.StatusCode, string(body))
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		log.Printf("callback health response status=%d json=%s", resp.StatusCode, string(pretty))
	} else {
		log.Printf("callback health response status=%d body=%s", resp.StatusCode, string(body))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleCreateDisbursement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	req, err := decodeDisbursementRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	status := s.pickStatus(req)

	resp := buildDisbursementResponse(req, status)
	if err := sendCallback(req, status); err != nil {
		log.Printf("callback failed: %v", err)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleSimulateSuccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	req, err := decodeDisbursementRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	status := "COMPLETED"
	resp := buildDisbursementResponse(req, status)
	if err := sendCallback(req, status); err != nil {
		log.Printf("callback failed: %v", err)
	}

	writeJSON(w, http.StatusOK, resp)
}

func decodeDisbursementRequest(r *http.Request) (disbursementRequest, error) {
	var req disbursementRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		if err == io.EOF {
			return defaultDisbursementRequest(), nil
		}
		return disbursementRequest{}, fmt.Errorf("invalid json")
	}
	if req.ExternalID == "" {
		req.ExternalID = defaultDisbursementRequest().ExternalID
	}
	if req.Amount == 0 {
		req.Amount = 10000
	}
	if req.BankCode == "" {
		req.BankCode = "BCA"
	}
	if req.AccountHolderName == "" {
		req.AccountHolderName = "Mock User"
	}
	if req.AccountNumber == "" {
		req.AccountNumber = "1234567890"
	}
	if req.Description == "" {
		req.Description = "mock disbursement"
	}
	return req, nil
}

func defaultDisbursementRequest() disbursementRequest {
	return disbursementRequest{
		ExternalID:        fmt.Sprintf("ext_success_%s", shortHash(time.Now().Format(time.RFC3339Nano))),
		Amount:            10000,
		BankCode:          "BCA",
		AccountHolderName: "Mock User",
		AccountNumber:     "1234567890",
		Description:       "mock disbursement",
	}
}

func buildDisbursementResponse(req disbursementRequest, status string) disbursementResponse {
	now := time.Now().Format(time.RFC3339)
	userID := getenv("XENDIT_USER_ID", "user_mock")
	return disbursementResponse{
		ID:                      "disb_" + shortHash(req.ExternalID),
		UserID:                  userID,
		ExternalID:              req.ExternalID,
		Amount:                  req.Amount,
		BankCode:                req.BankCode,
		AccountHolderName:       req.AccountHolderName,
		DisbursementDescription: req.Description,
		Status:                  status,
		Created:                 now,
		Updated:                 now,
	}
}

func (s *server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.firstFail = false
	s.seen = make(map[string]bool)
	s.attempts = make(map[string]int)
	s.firstSeen = make(map[string]time.Time)
	s.accountIdx = make(map[string]int)

	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func shortHash(value string) string {
	hash := md5.Sum([]byte(value))
	return hex.EncodeToString(hash[:])[:8]
}

func sendCallback(req disbursementRequest, status string) error {
	callbackURL := getenv("CALLBACK_URL", "")
	if callbackURL == "" {
		// #NOTE: set CALLBACK_URL env to the full callback URL
		return nil
	}
	userID := getenv("XENDIT_USER_ID", "user_mock")
	disbursementID := "disb_" + shortHash(req.ExternalID)
	webhookID := "wh_" + shortHash(disbursementID+":"+status)
	payload := callbackPayload{
		ID:                      disbursementID,
		Created:                 time.Now().Format(time.RFC3339),
		Updated:                 time.Now().Format(time.RFC3339),
		ExternalID:              req.ExternalID,
		UserID:                  userID,
		Amount:                  req.Amount,
		BankCode:                req.BankCode,
		AccountHolderName:       req.AccountHolderName,
		AccountNumber:           req.AccountNumber,
		DisbursementDescription: req.Description,
		Status:                  status,
		IsInstant:               false,
		WebhookID:               webhookID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, callbackURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
