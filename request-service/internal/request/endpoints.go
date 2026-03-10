package request

import (
	"context"
	"request-service/pkg/model"
	"time"

	"github.com/go-kit/kit/endpoint"
)

// --- CreateRequest (POST /requests) ---
type CreateRequestRequest struct {
	PassengerID string            `json:"passengerID"`
	Origin      model.GeoLocation `json:"origin"`
	Destination model.GeoLocation `json:"destination"`
	DesiredTime time.Time         `json:"desiredTime"`
	Passengers  int               `json:"passengers"`
	Preferences model.Preferences `json:"preferences"`
}

type CreateRequestResponse struct {
	model.CarRequest
	Error string `json:"error,omitempty"`
}

func MakeCreateRequestEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateRequestRequest)
		userID := ctx.Value(ContextKeyUserID).(string)

		carReq := model.CarRequest{
			PassengerID: req.PassengerID,
			Origin:      req.Origin,
			Destination: req.Destination,
			DesiredTime: req.DesiredTime,
			Passengers:  req.Passengers,
			Preferences: req.Preferences,
		}
		result, err := s.CreateRequest(ctx, userID, carReq)
		if err != nil {
			return CreateRequestResponse{Error: err.Error()}, err
		}
		return CreateRequestResponse{CarRequest: result}, nil
	}
}

// --- GetRequests (GET /requests) ---
type GetRequestsRequest struct {
	Status string
}

type GetRequestsResponse struct {
	Requests []model.CarRequest `json:"requests"`
	Error    string             `json:"error,omitempty"`
}

func MakeGetRequestsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetRequestsRequest)
		userID := ctx.Value(ContextKeyUserID).(string)

		var results []model.CarRequest
		var err error

		if userID == "system" {
			results, err = s.GetRequests(ctx, model.RequestStatus(req.Status), "")
		} else {
			results, err = s.GetRequests(ctx, model.RequestStatus(req.Status), userID)
		}

		if err != nil {
			return GetRequestsResponse{Error: err.Error()}, err
		}

		return GetRequestsResponse{Requests: results}, nil
	}
}

// --- GetAllPendingRequests (GET /requests/pending) ---
type GetAllPendingRequestsRequest struct{}

type GetAllPendingRequestsResponse struct {
	Requests []model.CarRequest `json:"requests"`
	Error    string             `json:"error,omitempty"`
}

func MakeGetAllPendingRequestsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Retrieve all pending requests from the service
		userID := ctx.Value(ContextKeyUserID).(string)
		var results []model.CarRequest
		var err error

		if userID == "system" {
			results, err = s.GetAllPendingRequests(ctx, "")
		} else {
			results, err = s.GetAllPendingRequests(ctx, userID)
		}
		if err != nil {
			return GetAllPendingRequestsResponse{Error: err.Error()}, err
		}

		return GetAllPendingRequestsResponse{Requests: results}, nil
	}
}

// --- CancelRequest (DELETE /requests/{id}) ---
type CancelRequestRequest struct {
	ID string
}

type CancelRequestResponse struct {
	Error string `json:"error,omitempty"`
}

func MakeCancelRequestEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CancelRequestRequest)
		userID := ctx.Value(ContextKeyUserID).(string)

		err := s.CancelRequest(ctx, userID, req.ID)
		if err != nil {
			return CancelRequestResponse{Error: err.Error()}, err
		}
		return CancelRequestResponse{}, nil
	}
}

// --- PatchRequest (PATCH /requests/{id}) ---
type PatchRequestRequest struct {
	ID          string            `json:"-"`
	Origin      model.GeoLocation `json:"origin"`
	Destination model.GeoLocation `json:"destination"`
	DesiredTime time.Time         `json:"desiredTime"`
	Passengers  int               `json:"passengers"`
	Preferences model.Preferences `json:"preferences"`
	Status      string            `json:"status"` // --- ADDED THIS FIELD ---
}

type PatchRequestResponse struct {
	model.CarRequest
	Error string `json:"error,omitempty"`
}

func MakePatchRequestEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PatchRequestRequest)
		userID := ctx.Value(ContextKeyUserID).(string)

		update := model.CarRequest{
			Origin:      req.Origin,
			Destination: req.Destination,
			DesiredTime: req.DesiredTime,
			Passengers:  req.Passengers,
			Preferences: req.Preferences,
			Status:      model.RequestStatus(req.Status), // --- MAP STATUS HERE ---
		}
		result, err := s.PatchRequest(ctx, userID, req.ID, update)
		if err != nil {
			return PatchRequestResponse{Error: err.Error()}, err
		}
		return PatchRequestResponse{CarRequest: result}, nil
	}
}
