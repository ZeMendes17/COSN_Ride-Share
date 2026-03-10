package matching

import (
	"context"
	"matching-service/pkg/model"

	"github.com/go-kit/kit/endpoint"
)

// --- GetMatchesForRequest ---

type GetMatchesRequest struct {
	RequestID string
}

type GetMatchesResponse struct {
	Matches []model.Match `json:"matches"`
	Error   string        `json:"error,omitempty"`
}

func MakeGetMatchesEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetMatchesRequest)
		matches, err := s.GetMatchesForRequest(ctx, req.RequestID)
		if err != nil {
			return GetMatchesResponse{Error: err.Error()}, err
		}
		return GetMatchesResponse{Matches: matches}, nil
	}
}

// --- SelectMatch ---

type SelectMatchRequest struct {
	RequestID string
	OfferID   string
}

type SelectMatchResponse struct {
	Error string `json:"error,omitempty"`
}

func MakeSelectMatchEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SelectMatchRequest)
		err := s.SelectMatch(ctx, req.RequestID, req.OfferID)
		if err != nil {
			return SelectMatchResponse{Error: err.Error()}, err
		}
		return SelectMatchResponse{}, nil
	}
}

// --- Testing/Manual Trigger (Optional) ---
// Useful for simulating a Kafka message via HTTP during dev

type TriggerOfferRequest struct {
	Offer            model.Offer `json:"offer"`
	TriggerRequestID []string    `json:"triggerRequestId"`
}

type TriggerOfferResponse struct {
	Message string `json:"message"`
}

func MakeTriggerOfferEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TriggerOfferRequest)
		err := s.ProcessOffer(ctx, req.Offer, req.TriggerRequestID)
		if err != nil {
			return nil, err
		}
		return TriggerOfferResponse{Message: "Offer processed"}, nil
	}
}

type CancelMatchRequest struct {
	RequestID string
	MatchID   string
}
type CancelMatchResponse struct {
	Error string `json:"error,omitempty"`
}

func MakeCancelMatchEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CancelMatchRequest)
		err := s.CancelMatch(ctx, req.RequestID, req.MatchID)
		if err != nil {
			return CancelMatchResponse{Error: err.Error()}, err
		}
		return CancelMatchResponse{}, nil
	}
}
