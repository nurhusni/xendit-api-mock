package scenario

import (
	"math/rand"
	"sync"
	"time"

	"xendit-api-mock/internal/domain"
)

type Engine struct {
	mu         sync.Mutex
	firstFail  bool
	seen       map[string]bool
	attempts   map[string]int
	firstSeen  map[string]time.Time
	accountIdx map[string]int
	scenario   *Config
	useRandom  bool
	randomizer *rand.Rand
}

func NewEngine(cfg *Config) *Engine {
	return &Engine{
		seen:       make(map[string]bool),
		attempts:   make(map[string]int),
		firstSeen:  make(map[string]time.Time),
		accountIdx: make(map[string]int),
		scenario:   cfg,
		randomizer: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) WithRandomStatus(enabled bool) *Engine {
	e.useRandom = enabled
	return e
}

func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.firstFail = false
	e.seen = make(map[string]bool)
	e.attempts = make(map[string]int)
	e.firstSeen = make(map[string]time.Time)
	e.accountIdx = make(map[string]int)
}

func (e *Engine) PickStatus(req domain.DisbursementRequest) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.useRandom {
		return e.pickStatusRandom()
	}

	if e.scenario == nil {
		return e.pickStatusDefault(req.ExternalID)
	}

	return e.pickStatusScenario(req)
}

func (e *Engine) pickStatusRandom() string {
	if e.randomizer.Intn(2) == 0 {
		return domain.StatusCompleted
	}
	return domain.StatusFailed
}

func (e *Engine) pickStatusDefault(externalID string) string {
	if e.seen[externalID] {
		return domain.StatusCompleted
	}

	e.seen[externalID] = true
	if !e.firstFail {
		e.firstFail = true
		return domain.StatusFailed
	}

	return domain.StatusCompleted
}

func (e *Engine) pickStatusScenario(req domain.DisbursementRequest) string {
	for _, batch := range e.scenario.Batches {
		if batch.AccountNumber != req.AccountNumber {
			continue
		}
		if batch.TopupID != "" && batch.TopupID != req.Description {
			continue
		}
		if result, ok := e.applyRules(req, batch.Disbursements, "batch:"+batch.TopupID+":"+batch.AccountNumber); ok {
			return result
		}
	}

	for _, account := range e.scenario.Accounts {
		if account.AccountNumber != req.AccountNumber {
			continue
		}
		if result, ok := e.applyRules(req, account.Disbursements, "account:"+account.AccountNumber); ok {
			return result
		}
	}

	return domain.StatusCompleted
}

func (e *Engine) applyRules(req domain.DisbursementRequest, rules []Rule, key string) (string, bool) {
	for _, rule := range rules {
		if rule.ExternalID == "" {
			continue
		}
		if rule.ExternalID == req.ExternalID {
			return e.applyRule(req.ExternalID, rule), true
		}
	}

	idx := e.accountIdx[key]
	if idx < len(rules) {
		rule := rules[idx]
		e.accountIdx[key] = idx + 1
		return e.applyRule(req.ExternalID, rule), true
	}

	return "", false
}

func (e *Engine) applyRule(externalID string, rule Rule) string {
	if e.attempts[externalID] == 0 {
		e.firstSeen[externalID] = time.Now()
	}
	e.attempts[externalID]++

	switch rule.Outcome {
	case "success":
		return domain.StatusCompleted
	case "fail_then_succeed":
		if rule.RetrySuccessAt > 0 && e.attempts[externalID] > rule.RetrySuccessAt {
			return domain.StatusCompleted
		}
		return domain.StatusFailed
	case "fail_until_timeout":
		return domain.StatusFailed
	default:
		return domain.StatusCompleted
	}
}
