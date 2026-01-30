package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetenv(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	if got := getenv("TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("expected value, got %s", got)
	}
	if got := getenv("MISSING_KEY", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %s", got)
	}
}

func TestLoadDotEnvSetsValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "FOO=bar\n# comment\nBAZ=qux\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	loadDotEnv(path)
	if got := os.Getenv("FOO"); got != "bar" {
		t.Fatalf("expected FOO=bar, got %s", got)
	}
	if got := os.Getenv("BAZ"); got != "qux" {
		t.Fatalf("expected BAZ=qux, got %s", got)
	}
}

func TestLoadDotEnvDoesNotOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "FOO=bar\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	os.Setenv("FOO", "existing")
	loadDotEnv(path)
	if got := os.Getenv("FOO"); got != "existing" {
		t.Fatalf("expected FOO=existing, got %s", got)
	}
}

func TestLoadScenarioEmptyPath(t *testing.T) {
	if got := loadScenario(""); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestLoadScenarioInvalidPath(t *testing.T) {
	if got := loadScenario("/nope/path.json"); got != nil {
		t.Fatalf("expected nil for invalid path, got %#v", got)
	}
}

func TestLoadScenarioInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0644); err != nil {
		t.Fatalf("write scenario file: %v", err)
	}

	if got := loadScenario(path); got != nil {
		t.Fatalf("expected nil for invalid json, got %#v", got)
	}
}

func TestLoadScenarioDefaultsTimeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.json")
	content := `{"accounts":[{"account_number":"123","disbursements":[{"outcome":"success"}]}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write scenario file: %v", err)
	}

	got := loadScenario(path)
	if got == nil {
		t.Fatal("expected scenario, got nil")
	}
	if got.RetryTimeoutMinutes != 60 {
		t.Fatalf("expected retry timeout 60, got %d", got.RetryTimeoutMinutes)
	}
}
