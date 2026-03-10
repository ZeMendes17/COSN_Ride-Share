package request

import (
	"context"
	"fmt"
	"request-service/internal/db"
	"request-service/internal/sns"
	"request-service/pkg/contracts"
	"request-service/pkg/model"
	"time"
)

type Service interface {
	CreateRequest(ctx context.Context, authenticatedUserID string, req model.CarRequest) (model.CarRequest, error)
	GetRequests(ctx context.Context, status model.RequestStatus, userID string) ([]model.CarRequest, error)
	GetAllPendingRequests(ctx context.Context, userID string) ([]model.CarRequest, error)
	CancelRequest(ctx context.Context, authenticatedUserID, requestID string) error
	PatchRequest(ctx context.Context, authenticatedUserID, requestID string, update model.CarRequest) (model.CarRequest, error)
	HealthCheck() error
}

type BasicService struct {
	repo db.Repository
	prod sns.Producer
}

func NewBasicService(repo db.Repository, prod sns.Producer) Service {
	return &BasicService{repo: repo, prod: prod}
}

func (s *BasicService) CreateRequest(ctx context.Context, authenticatedUserID string, req model.CarRequest) (model.CarRequest, error) {
	if req.PassengerID != authenticatedUserID {
		return model.CarRequest{}, ErrForbidden
	}

	if err := req.Validate(); err != nil {
		return model.CarRequest{}, err
	}

	req.Status = model.StatusPending
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.ID = "req-" + time.Now().Format("20060102150405")

	savedReq, err := s.repo.Save(ctx, req)
	if err != nil {
		return model.CarRequest{}, err
	}

	// Send "request.created.event" to SNS
	event := contracts.RequestCreatedEvent{
		RequestID:   savedReq.ID,
		PassengerID: savedReq.PassengerID,
		Origin:      savedReq.Origin,
		Destination: savedReq.Destination,
		DesiredTime: savedReq.DesiredTime,
		Passengers:  savedReq.Passengers,
		Preferences: savedReq.Preferences,
		Status:      string(savedReq.Status),
		Timestamp:   time.Now(),
	}
	_ = s.prod.SendRequestCreated(ctx, event)
	_ = s.prod.SendLog(ctx, contracts.LogEvent{
		ServiceID: "request-service",
		Topic:     "request.created",
		Message:   fmt.Sprintf("Request created: ID=%s, PassengerID=%s", savedReq.ID, savedReq.PassengerID),
	})

	return savedReq, nil
}

func (s *BasicService) GetRequests(ctx context.Context, status model.RequestStatus, userID string) ([]model.CarRequest, error) {
	return s.repo.GetAll(ctx, status, userID)
}

func (s *BasicService) GetAllPendingRequests(ctx context.Context, userID string) ([]model.CarRequest, error) {
	return s.repo.GetAllPending(ctx, userID)
}

func (s *BasicService) CancelRequest(ctx context.Context, authenticatedUserID string, requestID string) error {
	req, err := s.repo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}

	if req.PassengerID != authenticatedUserID {
		return ErrForbidden
	}

	if req.Status == model.StatusCompleted || req.Status == model.StatusCancelled {
		return model.ErrRequestFinalized
	}

	req.Status = model.StatusCancelled
	req.UpdatedAt = time.Now()
	if _, err := s.repo.Update(ctx, req); err != nil {
		return err
	}

	event := contracts.RequestCancelledEvent{
		RequestID:   req.ID,
		PassengerID: req.PassengerID,
		Reason:      "passenger_cancelled",
		Timestamp:   time.Now(),
	}
	_ = s.prod.SendRequestCancelled(ctx, event)
	_ = s.prod.SendLog(ctx, contracts.LogEvent{
		ServiceID: "request-service",
		Topic:     "request.cancelled",
		Message:   fmt.Sprintf("Request cancelled: ID=%s, PassengerID=%s, Reason=passenger_cancelled", req.ID, req.PassengerID),
	})

	return nil
}

func (s *BasicService) PatchRequest(ctx context.Context, authenticatedUserID, requestID string, update model.CarRequest) (model.CarRequest, error) {
	existing, err := s.repo.GetByID(ctx, requestID)
	if err != nil {
		return model.CarRequest{}, err
	}

	isSystemUser := authenticatedUserID == "system"

	if !isSystemUser && existing.PassengerID != authenticatedUserID {
		return model.CarRequest{}, ErrForbidden
	}

	// FIX: Allow "system" user to modify request even if it is not Pending (e.g., reverting Completed -> Pending)
	if !isSystemUser && existing.Status != model.StatusPending {
		return model.CarRequest{}, model.ErrRequestFinalized
	}

	// Only normal users update trip details
	if !isSystemUser {
		if update.Passengers > 0 {
			existing.Passengers = update.Passengers
		}
		if !update.DesiredTime.IsZero() {
			existing.DesiredTime = update.DesiredTime
		}
		if update.Origin.Lat != 0 || update.Origin.Lon != 0 {
			existing.Origin = update.Origin
		}
		if update.Destination.Lat != 0 || update.Destination.Lon != 0 {
			existing.Destination = update.Destination
		}
		existing.Preferences = update.Preferences
	}

	// Allow updating Status (typically by system)
	if update.Status != "" {
		existing.Status = update.Status
	}

	existing.UpdatedAt = time.Now()

	updatedReq, err := s.repo.Update(ctx, existing)
	if err != nil {
		return model.CarRequest{}, err
	}

	return updatedReq, nil
}

func (s *BasicService) HealthCheck() error {
	return s.repo.Ping()
}
