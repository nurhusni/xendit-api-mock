package disbursement

import (
	"time"
	"xendit-api-mock/internal/callback"
	"xendit-api-mock/internal/domain"
	"xendit-api-mock/internal/scenario"
)

type Service struct {
	engine                            *scenario.Engine
	cb                                *callback.Client
	userID                            string
	disableCallbacks                  bool
	allowFailedDisbursementCall       bool
	allowFailedGetDisbursementRequest bool
	randomGetDisbursementStatus       bool
}

func NewService(
	engine *scenario.Engine,
	cb *callback.Client,
	userID string,
	disableCallbacks, allowFailedDisbursementCall, allowFailedGetDisbursementRequest, randomGetDisbursementStatus bool,
) *Service {
	return &Service{
		engine:                            engine,
		cb:                                cb,
		userID:                            userID,
		disableCallbacks:                  disableCallbacks,
		allowFailedDisbursementCall:       allowFailedDisbursementCall,
		allowFailedGetDisbursementRequest: allowFailedGetDisbursementRequest,
		randomGetDisbursementStatus:       randomGetDisbursementStatus,
	}
}

func (s *Service) Create(req domain.DisbursementRequest) (domain.DisbursementResponse, error) {
	status := domain.NormalizeStatus(s.engine.PickStatus(req))
	resp := domain.BuildDisbursementResponse(req, status, s.userID)
	if s.disableCallbacks {
		return resp, nil
	}
	err := s.cb.Send(domain.BuildCallbackPayload(req, status, s.userID))
	return resp, err
}

func (s *Service) SimulateSuccess(req domain.DisbursementRequest) (domain.DisbursementResponse, error) {
	status := domain.NormalizeStatus(domain.StatusCompleted)
	resp := domain.BuildDisbursementResponse(req, status, s.userID)
	err := s.cb.Send(domain.BuildCallbackPayload(req, status, s.userID))
	return resp, err
}

func (s *Service) Reset() {
	s.engine.Reset()
}

func (s *Service) AllowFailedDisbursementCall() bool {
	return s.allowFailedDisbursementCall
}

func (s *Service) AllowFailedGetDisbursementRequest() bool {
	return s.allowFailedGetDisbursementRequest
}

func (s *Service) GetByExternalID(externalID string) []domain.CallbackPayload {
	req := domain.DefaultDisbursementRequest()
	req.ExternalID = externalID

	status := domain.StatusCompleted
	if s.randomGetDisbursementStatus {
		status = domain.RandomStatus()
	}

	now := time.Now().Format(time.RFC3339)
	disbursementID := domain.DisbursementID(externalID)

	var response []domain.CallbackPayload
	response = append(response, domain.CallbackPayload{
		ID:                      disbursementID,
		Created:                 now,
		Updated:                 now,
		ExternalID:              req.ExternalID,
		UserID:                  s.userID,
		Amount:                  req.Amount,
		BankCode:                req.BankCode,
		AccountHolderName:       req.AccountHolderName,
		AccountNumber:           req.AccountNumber,
		DisbursementDescription: req.Description,
		Status:                  status,
		FailureCode:             "",
		IsInstant:               false,
		WebhookID:               domain.WebhookID(disbursementID, status),
	})

	return response
}
