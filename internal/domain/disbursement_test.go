package domain

import "testing"

func setFeatureFlagsForTest(t *testing.T, flags FeatureFlags) {
	t.Helper()
	SetFeatureFlags(flags)
	t.Cleanup(func() {
		SetFeatureFlags(DefaultFeatureFlags())
	})
}

func TestNormalizeStatusDisallowFailed(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowFailed = false
	setFeatureFlagsForTest(t, flags)

	if got := NormalizeStatus(StatusFailed); got != StatusCompleted {
		t.Fatalf("expected COMPLETED when failed is disabled, got %s", got)
	}
	if got := NormalizeStatus("UNKNOWN"); got != StatusCompleted {
		t.Fatalf("expected COMPLETED for unknown status when failed is disabled, got %s", got)
	}
}

func TestBuildDisbursementResponseNormalizesStatus(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowFailed = false
	setFeatureFlagsForTest(t, flags)

	resp := BuildDisbursementResponse(DefaultDisbursementRequest(), StatusFailed, "user")
	if resp.Status != StatusCompleted {
		t.Fatalf("expected COMPLETED status in response, got %s", resp.Status)
	}
}

func TestBuildCallbackPayloadDisallowNonRetryableCodes(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowNonRetryable = false
	setFeatureFlagsForTest(t, flags)

	blocked := map[string]bool{
		"INSUFFICIENT_BALANCE": true,
		"INVALID_DESTINATION":  true,
		"TRANSFER_ERROR":       true,
	}

	for i := 0; i < 200; i++ {
		payload := BuildCallbackPayload(DefaultDisbursementRequest(), StatusFailed, "user")
		if payload.FailureCode == "" {
			t.Fatal("expected failure_code on failed status")
		}
		if blocked[payload.FailureCode] {
			t.Fatalf("unexpected non-retryable failure code: %s", payload.FailureCode)
		}
	}
}

func TestBuildCallbackPayloadDisallowFailureWith5MinsCodes(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowFailureWith5Mins = false
	setFeatureFlagsForTest(t, flags)

	blocked := map[string]bool{
		"UNKNOWN_BANK_NETWORK_ERROR": true,
		"REJECTED_BY_BANK":           true,
	}

	for i := 0; i < 200; i++ {
		payload := BuildCallbackPayload(DefaultDisbursementRequest(), StatusFailed, "user")
		if payload.FailureCode == "" {
			t.Fatal("expected failure_code on failed status")
		}
		if blocked[payload.FailureCode] {
			t.Fatalf("unexpected 5-minute failure code: %s", payload.FailureCode)
		}
	}
}

func TestBuildCallbackPayloadDisallowFailureWith70MinsCodes(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowFailureWith70Mins = false
	setFeatureFlagsForTest(t, flags)

	blocked := map[string]bool{
		"TEMPORARY_BANK_NETWORK_ERROR": true,
		"SWITCHING_NETWORK_ERROR":      true,
		"TEMPORARY_TRANSFER_ERROR":     true,
	}

	for i := 0; i < 200; i++ {
		payload := BuildCallbackPayload(DefaultDisbursementRequest(), StatusFailed, "user")
		if payload.FailureCode == "" {
			t.Fatal("expected failure_code on failed status")
		}
		if blocked[payload.FailureCode] {
			t.Fatalf("unexpected 70-minute failure code: %s", payload.FailureCode)
		}
	}
}

func TestBuildCallbackPayloadAllFailureCodesDisabled(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowNonRetryable = false
	flags.AllowFailureWith5Mins = false
	flags.AllowFailureWith70Mins = false
	setFeatureFlagsForTest(t, flags)

	payload := BuildCallbackPayload(DefaultDisbursementRequest(), StatusFailed, "user")
	if payload.Status != StatusFailed {
		t.Fatalf("expected FAILED status, got %s", payload.Status)
	}
	if payload.FailureCode != "" {
		t.Fatalf("expected empty failure_code when all failure code groups are disabled, got %s", payload.FailureCode)
	}
}

func TestBuildCallbackPayloadDisallowFailed(t *testing.T) {
	flags := DefaultFeatureFlags()
	flags.AllowFailed = false
	setFeatureFlagsForTest(t, flags)

	payload := BuildCallbackPayload(DefaultDisbursementRequest(), StatusFailed, "user")
	if payload.Status != StatusCompleted {
		t.Fatalf("expected COMPLETED when failed is disabled, got %s", payload.Status)
	}
	if payload.FailureCode != "" {
		t.Fatalf("expected empty failure_code for COMPLETED status, got %s", payload.FailureCode)
	}
}
