package main

import "time"

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
