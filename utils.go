package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func decodeDisbursementRequest(r *http.Request) (disbursementRequest, error) {
	var req disbursementRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		if err == io.EOF {
			return defaultDisbursementRequest(), nil
		}
		return disbursementRequest{}, fmt.Errorf("invalid json")
	}
	if req.ExternalID == "" {
		req.ExternalID = defaultDisbursementRequest().ExternalID
	}
	if req.Amount == 0 {
		req.Amount = 10000
	}
	if req.BankCode == "" {
		req.BankCode = "BCA"
	}
	if req.AccountHolderName == "" {
		req.AccountHolderName = "xamock user"
	}
	if req.AccountNumber == "" {
		req.AccountNumber = "xamock-1234567890"
	}
	if req.Description == "" {
		req.Description = "xamock disbursement"
	}
	return req, nil
}

func defaultDisbursementRequest() disbursementRequest {
	return disbursementRequest{
		ExternalID:        fmt.Sprintf("xamock_ext_%s", shortHash(time.Now().Format(time.RFC3339Nano))),
		Amount:            10000,
		BankCode:          "BCA",
		AccountHolderName: "xamock user",
		AccountNumber:     "xamock-1234567890",
		Description:       "xamock disbursement",
	}
}

func buildDisbursementResponse(req disbursementRequest, status string) disbursementResponse {
	now := time.Now().Format(time.RFC3339)
	userID := getenv("XENDIT_USER_ID", "user_mock")
	return disbursementResponse{
		ID:                      "disb_" + shortHash(req.ExternalID),
		UserID:                  userID,
		ExternalID:              req.ExternalID,
		Amount:                  req.Amount,
		BankCode:                req.BankCode,
		AccountHolderName:       req.AccountHolderName,
		DisbursementDescription: req.Description,
		Status:                  status,
		Created:                 now,
		Updated:                 now,
	}
}
