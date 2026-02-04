package httptransport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"xendit-api-mock/internal/domain"
)

func decodeDisbursementRequest(r *http.Request) (domain.DisbursementRequest, error) {
	var req domain.DisbursementRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		if err == io.EOF {
			return domain.DefaultDisbursementRequest(), nil
		}
		return domain.DisbursementRequest{}, fmt.Errorf("invalid json")
	}
	defaultReq := domain.DefaultDisbursementRequest()
	if req.ExternalID == "" {
		req.ExternalID = defaultReq.ExternalID
	}
	if req.Amount == 0 {
		req.Amount = defaultReq.Amount
	}
	if req.BankCode == "" {
		req.BankCode = defaultReq.BankCode
	}
	if req.AccountHolderName == "" {
		req.AccountHolderName = defaultReq.AccountHolderName
	}
	if req.AccountNumber == "" {
		req.AccountNumber = defaultReq.AccountNumber
	}
	if req.Description == "" {
		req.Description = defaultReq.Description
	}
	return req, nil
}
