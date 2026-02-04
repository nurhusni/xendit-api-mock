package domain

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	StatusCompleted = "COMPLETED"
	StatusFailed    = "FAILED"
)

type DisbursementRequest struct {
	ExternalID        string   `json:"external_id"`
	Amount            int      `json:"amount"`
	BankCode          string   `json:"bank_code"`
	AccountHolderName string   `json:"account_holder_name"`
	AccountNumber     string   `json:"account_number"`
	Description       string   `json:"description"`
	EmailTo           []string `json:"email_to,omitempty"`
	EmailCC           []string `json:"email_cc,omitempty"`
	EmailBCC          []string `json:"email_bcc,omitempty"`
}

type DisbursementResponse struct {
	ID                      string   `json:"id"`
	UserID                  string   `json:"user_id"`
	ExternalID              string   `json:"external_id"`
	Amount                  int      `json:"amount"`
	BankCode                string   `json:"bank_code"`
	AccountHolderName       string   `json:"account_holder_name"`
	DisbursementDescription string   `json:"disbursement_description"`
	Status                  string   `json:"status"`
	Created                 string   `json:"created"`
	Updated                 string   `json:"updated"`
	FailureCode             string   `json:"failure_code,omitempty"`
	EmailTo                 []string `json:"email_to,omitempty"`
	EmailCC                 []string `json:"email_cc,omitempty"`
	EmailBCC                []string `json:"email_bcc,omitempty"`
}

type CallbackPayload struct {
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
	FailureCode             string `json:"failure_code,omitempty"`
	IsInstant               bool   `json:"is_instant"`
	WebhookID               string `json:"webhookId"`
}

func NormalizeStatus(status string) string {
	if status == StatusCompleted {
		return StatusCompleted
	}
	if status == StatusFailed {
		return StatusFailed
	}
	return StatusFailed
}

func ShortHash(value string) string {
	hash := md5.Sum([]byte(value))
	return hex.EncodeToString(hash[:])[:8]
}

func DisbursementID(externalID string) string {
	return "disb_" + ShortHash(externalID)
}

func WebhookID(disbursementID, status string) string {
	return "wh_" + ShortHash(disbursementID+":"+status)
}

func DefaultDisbursementRequest() DisbursementRequest {
	return DisbursementRequest{
		ExternalID:        fmt.Sprintf("xamock_ext_%s", ShortHash(time.Now().Format(time.RFC3339Nano))),
		Amount:            10000,
		BankCode:          "BCA",
		AccountHolderName: "xamock user",
		AccountNumber:     "xamock-1234567890",
		Description:       "xamock disbursement",
	}
}

func BuildDisbursementResponse(req DisbursementRequest, status, userID string) DisbursementResponse {
	now := time.Now().Format(time.RFC3339)
	return DisbursementResponse{
		ID:                      DisbursementID(req.ExternalID),
		UserID:                  userID,
		ExternalID:              req.ExternalID,
		Amount:                  req.Amount,
		BankCode:                req.BankCode,
		AccountHolderName:       req.AccountHolderName,
		DisbursementDescription: req.Description,
		Status:                  status,
		Created:                 now,
		Updated:                 now,
		EmailTo:                 req.EmailTo,
		EmailCC:                 req.EmailCC,
		EmailBCC:                req.EmailBCC,
	}
}

func BuildCallbackPayload(req DisbursementRequest, status, userID string) CallbackPayload {
	status = NormalizeStatus(status)
	now := time.Now().Format(time.RFC3339)
	disbursementID := DisbursementID(req.ExternalID)
	return CallbackPayload{
		ID:                      disbursementID,
		Created:                 now,
		Updated:                 now,
		ExternalID:              req.ExternalID,
		UserID:                  userID,
		Amount:                  req.Amount,
		BankCode:                req.BankCode,
		AccountHolderName:       req.AccountHolderName,
		AccountNumber:           req.AccountNumber,
		DisbursementDescription: req.Description,
		Status:                  status,
		IsInstant:               false,
		WebhookID:               WebhookID(disbursementID, status),
	}
}
