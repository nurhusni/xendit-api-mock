package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"xendit-api-mock/internal/domain"
	"xendit-api-mock/internal/scenario"
)

func TestPickStatusDefaultSequence(t *testing.T) {
	engine := scenario.NewEngine(nil)
	if got := engine.PickStatus(domain.DisbursementRequest{ExternalID: "ext-1"}); got != "FAILED" {
		t.Fatalf("expected FAILED, got %s", got)
	}
	if got := engine.PickStatus(domain.DisbursementRequest{ExternalID: "ext-1"}); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}
	if got := engine.PickStatus(domain.DisbursementRequest{ExternalID: "ext-2"}); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED for new external after first fail, got %s", got)
	}
}

func TestPickStatusScenarioExactMatch(t *testing.T) {
	engine := scenario.NewEngine(&scenario.Config{
		Accounts: []scenario.AccountScenario{
			{AccountNumber: "x1", Disbursements: []scenario.Rule{{ExternalID: "ext-1", Outcome: "success"}}},
		},
	})
	status := engine.PickStatus(domain.DisbursementRequest{AccountNumber: "x1", ExternalID: "ext-1"})
	if status != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", status)
	}
}

func TestPickStatusScenarioOrderBased(t *testing.T) {
	engine := scenario.NewEngine(&scenario.Config{
		Accounts: []scenario.AccountScenario{
			{AccountNumber: "x1", Disbursements: []scenario.Rule{{Outcome: "fail_then_succeed", RetrySuccessAt: 1}}},
		},
	})
	status1 := engine.PickStatus(domain.DisbursementRequest{AccountNumber: "x1", ExternalID: "ext-1"})
	if status1 != "FAILED" {
		t.Fatalf("expected FAILED, got %s", status1)
	}
}

func TestApplyRuleOutcomes(t *testing.T) {
	engine := scenario.NewEngine(&scenario.Config{
		Accounts: []scenario.AccountScenario{
			{AccountNumber: "x1", Disbursements: []scenario.Rule{{Outcome: "success"}}},
		},
	})
	if got := engine.PickStatus(domain.DisbursementRequest{AccountNumber: "x1", ExternalID: "ext"}); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}

	engine = scenario.NewEngine(&scenario.Config{
		Accounts: []scenario.AccountScenario{
			{AccountNumber: "x1", Disbursements: []scenario.Rule{{Outcome: "fail_then_succeed", RetrySuccessAt: 1}}},
		},
	})
	if got := engine.PickStatus(domain.DisbursementRequest{AccountNumber: "x1", ExternalID: "ext"}); got != "FAILED" {
		t.Fatalf("expected FAILED first, got %s", got)
	}
	if got := engine.PickStatus(domain.DisbursementRequest{AccountNumber: "x1", ExternalID: "ext"}); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED second, got %s", got)
	}

	engine = scenario.NewEngine(&scenario.Config{
		Accounts: []scenario.AccountScenario{
			{AccountNumber: "x1", Disbursements: []scenario.Rule{{Outcome: "fail_until_timeout"}}},
		},
	})
	if got := engine.PickStatus(domain.DisbursementRequest{AccountNumber: "x1", ExternalID: "ext"}); got != "FAILED" {
		t.Fatalf("expected FAILED, got %s", got)
	}
}

func TestNormalizeStatus(t *testing.T) {
	if got := domain.NormalizeStatus("COMPLETED"); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED, got %s", got)
	}
	if got := domain.NormalizeStatus("FAILED"); got != "FAILED" {
		t.Fatalf("expected FAILED, got %s", got)
	}
	if got := domain.NormalizeStatus("UNKNOWN"); got != "FAILED" {
		t.Fatalf("expected FAILED for unknown, got %s", got)
	}
}

func TestScenarioSampleTopup001Order(t *testing.T) {
	cfg := loadScenario(filepath.Join(".", "scenario.sample.json"))
	if cfg == nil {
		t.Fatal("expected scenario sample to load")
	}

	engine := scenario.NewEngine(cfg)

	accountNumber := "1234567890"
	description := "TOPUP-001"

	req := domain.DisbursementRequest{AccountNumber: accountNumber, ExternalID: "ext-1", Description: description}
	if got := engine.PickStatus(req); got != "COMPLETED" {
		t.Fatalf("expected COMPLETED for first rule, got %s", got)
	}

	req = domain.DisbursementRequest{AccountNumber: accountNumber, ExternalID: "ext-2", Description: description}
	if got := engine.PickStatus(req); got != "FAILED" {
		t.Fatalf("expected FAILED for fail_then_succeed rule, got %s", got)
	}

	req = domain.DisbursementRequest{AccountNumber: accountNumber, ExternalID: "ext-3", Description: description}
	if got := engine.PickStatus(req); got != "FAILED" {
		t.Fatalf("expected FAILED for fail_until_timeout rule, got %s", got)
	}
}

func TestScenarioSampleTopup002Order(t *testing.T) {
	cfg := loadScenario(filepath.Join(".", "scenario.sample.json"))
	if cfg == nil {
		t.Fatal("expected scenario sample to load")
	}

	engine := scenario.NewEngine(cfg)

	accountNumber := "1234567890"
	description := "TOPUP-002"

	for i := 1; i <= 2; i++ {
		req := domain.DisbursementRequest{AccountNumber: accountNumber, ExternalID: fmt.Sprintf("ext-%d", i), Description: description}
		if got := engine.PickStatus(req); got != "COMPLETED" {
			t.Fatalf("expected COMPLETED for success rule %d, got %s", i, got)
		}
	}
}

func TestScenarioBatchPrecedence(t *testing.T) {
	engine := scenario.NewEngine(&scenario.Config{
		Batches: []scenario.BatchScenario{
			{
				TopupID:       "TOPUP-999",
				AccountNumber: "acct-1",
				Disbursements: []scenario.Rule{{Outcome: "fail_until_timeout"}},
			},
		},
		Accounts: []scenario.AccountScenario{
			{AccountNumber: "acct-1", Disbursements: []scenario.Rule{{Outcome: "success"}}},
		},
	})

	req := domain.DisbursementRequest{AccountNumber: "acct-1", ExternalID: "ext-1", Description: "TOPUP-999"}
	if got := engine.PickStatus(req); got != "FAILED" {
		t.Fatalf("expected batch rule to take precedence, got %s", got)
	}
}
