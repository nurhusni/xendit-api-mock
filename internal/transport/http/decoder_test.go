package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeDisbursementRequestEmptyBodyDefaults(t *testing.T) {
	req := httptest.NewRequest("POST", "/xendit/disbursements", bytes.NewReader(nil))
	decoded, err := decodeDisbursementRequest(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !strings.HasPrefix(decoded.ExternalID, "xamock_ext_") {
		t.Fatalf("expected xamock_ext_ prefix, got %s", decoded.ExternalID)
	}
	if decoded.AccountHolderName != "xamock user" {
		t.Fatalf("expected account holder name xamock user, got %s", decoded.AccountHolderName)
	}
	if decoded.AccountNumber != "xamock-1234567890" {
		t.Fatalf("expected account number xamock-1234567890, got %s", decoded.AccountNumber)
	}
	if decoded.Description != "xamock disbursement" {
		t.Fatalf("expected description xamock disbursement, got %s", decoded.Description)
	}
}

func TestDecodeDisbursementRequestInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/xendit/disbursements", strings.NewReader("{bad"))
	if _, err := decodeDisbursementRequest(req); err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestDecodeDisbursementRequestPartialDefaults(t *testing.T) {
	payload := `{"external_id":"custom","amount":5000}`
	req := httptest.NewRequest("POST", "/xendit/disbursements", strings.NewReader(payload))
	decoded, err := decodeDisbursementRequest(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if decoded.ExternalID != "custom" {
		t.Fatalf("expected external_id custom, got %s", decoded.ExternalID)
	}
	if decoded.Amount != 5000 {
		t.Fatalf("expected amount 5000, got %d", decoded.Amount)
	}
	if decoded.BankCode != "BCA" {
		t.Fatalf("expected default bank code BCA, got %s", decoded.BankCode)
	}
}

func TestDefaultDisbursementRequestIsJSONSerializable(t *testing.T) {
	data, err := json.Marshal(decodeDefaultRequestForTest())
	if err != nil {
		t.Fatalf("expected json marshal to succeed, got %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected json output, got empty")
	}
}

func decodeDefaultRequestForTest() interface{} {
	req, _ := decodeDisbursementRequest(httptest.NewRequest("POST", "/xendit/disbursements", bytes.NewReader(nil)))
	return req
}
