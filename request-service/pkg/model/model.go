package model

import (
	"errors"
	"time"
)

var (
	ErrInvalidRequest   = errors.New("invalid request parameters")
	ErrNotFound         = errors.New("request not found")
	ErrNotModified      = errors.New("request status cannot be modified")
	ErrRequestFinalized = errors.New("request already finalized")
)

type RequestStatus string

const (
	StatusPending   RequestStatus = "Pending"
	StatusCompleted RequestStatus = "Completed"
	StatusCancelled RequestStatus = "Cancelled"
)

// GeoLocation represents a lat/lon pair
type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Preferences represents passenger preferences (more could be added)
type Preferences struct {
	Smoking bool `json:"smoking"`
	Pets    bool `json:"pets"`
	Music   bool `json:"music"`
}

// CarRequest represents the stored ride request
type CarRequest struct {
	ID          string        `json:"id"`
	PassengerID string        `json:"passengerID"`
	Origin      GeoLocation   `json:"origin"`
	Destination GeoLocation   `json:"destination"`
	DesiredTime time.Time     `json:"desiredTime"`
	Passengers  int           `json:"passengers"`
	Preferences Preferences   `json:"preferences"`
	Status      RequestStatus `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

// Validate ensures the request has essential data.
func (r *CarRequest) Validate() error {
	if r.PassengerID == "" {
		return ErrInvalidRequest
	}
	if r.Passengers < 1 {
		return ErrInvalidRequest
	}
	return nil
}
