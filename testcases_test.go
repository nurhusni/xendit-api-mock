package main

import (
	"testing"
	"time"
)

func TestPickStatusDefaultSequence(t *testing.T) {
	s := &server{
		seen:       make(map[string]bool),
		attempts:   make(map[string]int),
		firstSeen:  make(map[string]time.Time),
		accountIdx: make(map[string]int),
	}
	if got := s.pickStatusDefault("ext-1"); got != "FAILED" {
		t.Fatalf("expected FAILED, got %s", got)
	}
	if got := s.pickStatusDefault("ext-1"); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}
	if got := s.pickStatusDefault("ext-2"); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED for new external after first fail, got %s", got)
	}
}

func TestPickStatusScenarioExactMatch(t *testing.T) {
	s := &server{
		attempts:   make(map[string]int),
		firstSeen:  make(map[string]time.Time),
		accountIdx: make(map[string]int),
		scenario: &scenarioConfig{
			Accounts: []accountScenario{
				{AccountNumber: "x1", Disbursements: []disbursementRule{{ExternalID: "ext-1", Outcome: "success"}}},
			},
		},
	}
	status := s.pickStatusScenario(disbursementRequest{AccountNumber: "x1", ExternalID: "ext-1"})
	if status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", status)
	}
}

func TestPickStatusScenarioOrderBased(t *testing.T) {
	s := &server{
		attempts:   make(map[string]int),
		firstSeen:  make(map[string]time.Time),
		accountIdx: make(map[string]int),
		scenario: &scenarioConfig{
			Accounts: []accountScenario{
				{AccountNumber: "x1", Disbursements: []disbursementRule{{Outcome: "fail_then_succeed", RetrySuccessAt: 1}}},
			},
		},
	}
	status1 := s.pickStatusScenario(disbursementRequest{AccountNumber: "x1", ExternalID: "ext-1"})
	if status1 != "FAILED" {
		t.Fatalf("expected FAILED, got %s", status1)
	}
}

func TestApplyRuleOutcomes(t *testing.T) {
	s := &server{attempts: make(map[string]int), firstSeen: make(map[string]time.Time)}
	if got := s.applyRule("ext", disbursementRule{Outcome: "success"}); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}

	s = &server{attempts: make(map[string]int), firstSeen: make(map[string]time.Time)}
	if got := s.applyRule("ext", disbursementRule{Outcome: "fail_then_succeed", RetrySuccessAt: 1}); got != "FAILED" {
		t.Fatalf("expected FAILED first, got %s", got)
	}
	if got := s.applyRule("ext", disbursementRule{Outcome: "fail_then_succeed", RetrySuccessAt: 1}); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED second, got %s", got)
	}

	s = &server{attempts: make(map[string]int), firstSeen: make(map[string]time.Time)}
	if got := s.applyRule("ext", disbursementRule{Outcome: "fail_until_timeout"}); got != "FAILED" {
		t.Fatalf("expected FAILED, got %s", got)
	}
}

func TestNormalizeStatus(t *testing.T) {
	if got := normalizeStatus("COMPLETED"); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}
	if got := normalizeStatus("FAILED"); got != "FAILED" {
		t.Fatalf("expected FAILED, got %s", got)
	}
	if got := normalizeStatus("UNKNOWN"); got != "FAILED" {
		t.Fatalf("expected FAILED for unknown, got %s", got)
	}
}
