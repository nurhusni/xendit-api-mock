package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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

type scenarioConfig struct {
	RetryTimeoutMinutes int               `json:"retry_timeout_minutes"`
	Accounts            []accountScenario `json:"accounts"`
	Batches             []batchScenario   `json:"batches"`
}

type accountScenario struct {
	AccountNumber string             `json:"account_number"`
	Disbursements []disbursementRule `json:"disbursements"`
}

type disbursementRule struct {
	ExternalID     string `json:"external_id"`
	Outcome        string `json:"outcome"`
	RetrySuccessAt int    `json:"retry_success_at"`
}

type batchScenario struct {
	TopupID       string             `json:"topup_id"`
	AccountNumber string             `json:"account_number"`
	Disbursements []disbursementRule `json:"disbursements"`
}

func main() {
	loadDotEnv(".env")
	addr := getenv("PORT", "8080")
	log.Printf("xendit-api-mock listening on :%s", addr)

	s := &server{
		seen:       make(map[string]bool),
		attempts:   make(map[string]int),
		firstSeen:  make(map[string]time.Time),
		accountIdx: make(map[string]int),
		scenario:   loadScenario(getenv("SCENARIO_FILE", "")),
	}

	http.HandleFunc("/xendit/disbursements", s.handleCreateDisbursement)
	http.HandleFunc("/xendit/healthz", handleHealth)
	http.HandleFunc("/xendit/reset", s.handleReset)

	if err := http.ListenAndServe(":"+addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleCreateDisbursement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req disbursementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.ExternalID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "external_id required"})
		return
	}

	status := s.pickStatus(req)

	now := time.Now().Format(time.RFC3339)
	userID := getenv("XENDIT_USER_ID", "user_mock")
	resp := disbursementResponse{
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

	if err := sendCallback(req, status); err != nil {
		log.Printf("callback failed: %v", err)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *server) pickStatus(req disbursementRequest) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scenario == nil {
		return s.pickStatusDefault(req.ExternalID)
	}

	return s.pickStatusScenario(req)
}

func (s *server) pickStatusDefault(externalID string) string {
	if s.seen[externalID] {
		return "COMPLETED"
	}

	s.seen[externalID] = true
	if !s.firstFail {
		s.firstFail = true
		return "FAILED"
	}

	return "COMPLETED"
}

func (s *server) pickStatusScenario(req disbursementRequest) string {
	for _, batch := range s.scenario.Batches {
		if batch.AccountNumber != req.AccountNumber {
			continue
		}
		if batch.TopupID != "" && batch.TopupID != req.Description {
			continue
		}
		if result, ok := s.applyRules(req, batch.Disbursements, "batch:"+batch.TopupID+":"+batch.AccountNumber); ok {
			return result
		}
	}

	for _, account := range s.scenario.Accounts {
		if account.AccountNumber != req.AccountNumber {
			continue
		}
		if result, ok := s.applyRules(req, account.Disbursements, "account:"+account.AccountNumber); ok {
			return result
		}
	}

	return "COMPLETED"
}

func (s *server) applyRules(req disbursementRequest, rules []disbursementRule, key string) (string, bool) {
	for _, rule := range rules {
		if rule.ExternalID == "" {
			continue
		}
		if rule.ExternalID == req.ExternalID {
			return s.applyRule(req.ExternalID, rule), true
		}
	}

	idx := s.accountIdx[key]
	if idx < len(rules) {
		rule := rules[idx]
		s.accountIdx[key] = idx + 1
		return s.applyRule(req.ExternalID, rule), true
	}

	return "", false
}

func (s *server) applyRule(externalID string, rule disbursementRule) string {
	if s.attempts[externalID] == 0 {
		s.firstSeen[externalID] = time.Now()
	}
	s.attempts[externalID]++

	switch rule.Outcome {
	case "success":
		return "COMPLETED"
	case "fail_then_succeed":
		if rule.RetrySuccessAt > 0 && s.attempts[externalID] > rule.RetrySuccessAt {
			return "COMPLETED"
		}
		return "FAILED"
	case "fail_until_timeout":
		return "FAILED"
	default:
		return "COMPLETED"
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func shortHash(value string) string {
	hash := md5.Sum([]byte(value))
	return hex.EncodeToString(hash[:])[:8]
}

func sendCallback(req disbursementRequest, status string) error {
	baseURL := getenv("CALLBACK_BASE_URL", "")
	if baseURL == "" {
		// #NOTE: set CALLBACK_BASE_URL env to the service base URL (e.g. https://sandbox.example.com)
		return nil
	}

	callbackURL := fmt.Sprintf("%s/api/v1/it/xendit/disbursement/callback", baseURL)
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

func loadScenario(path string) *scenarioConfig {
	if path == "" {
		return nil
	}

	file, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read scenario file: %v", err)
		return nil
	}

	var cfg scenarioConfig
	if err := json.Unmarshal(file, &cfg); err != nil {
		log.Printf("failed to parse scenario file: %v", err)
		return nil
	}

	if cfg.RetryTimeoutMinutes == 0 {
		cfg.RetryTimeoutMinutes = 60
	}

	return &cfg
}
