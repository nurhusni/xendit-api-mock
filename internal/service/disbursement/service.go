package disbursement

import (
	"xendit-api-mock/internal/callback"
	"xendit-api-mock/internal/domain"
	"xendit-api-mock/internal/scenario"
)

type Service struct {
	engine *scenario.Engine
	cb     *callback.Client
	userID string
}

func NewService(engine *scenario.Engine, cb *callback.Client, userID string) *Service {
	return &Service{engine: engine, cb: cb, userID: userID}
}

func (s *Service) Create(req domain.DisbursementRequest) (domain.DisbursementResponse, error) {
	status := domain.NormalizeStatus(s.engine.PickStatus(req))
	resp := domain.BuildDisbursementResponse(req, status, s.userID)
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
