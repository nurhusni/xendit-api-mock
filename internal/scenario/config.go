package scenario

import (
	"encoding/json"
	"os"
)

type Config struct {
	RetryTimeoutMinutes int               `json:"retry_timeout_minutes"`
	Accounts            []AccountScenario `json:"accounts"`
	Batches             []BatchScenario   `json:"batches"`
}

type AccountScenario struct {
	AccountNumber string `json:"account_number"`
	Disbursements []Rule `json:"disbursements"`
}

type Rule struct {
	ExternalID     string `json:"external_id"`
	Outcome        string `json:"outcome"`
	RetrySuccessAt int    `json:"retry_success_at"`
}

type BatchScenario struct {
	TopupID       string `json:"topup_id"`
	AccountNumber string `json:"account_number"`
	Disbursements []Rule `json:"disbursements"`
}

func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.RetryTimeoutMinutes == 0 {
		cfg.RetryTimeoutMinutes = 60
	}
	return &cfg, nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}
