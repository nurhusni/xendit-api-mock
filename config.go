package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

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
