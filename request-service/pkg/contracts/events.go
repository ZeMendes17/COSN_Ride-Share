package contracts

import (
	"request-service/pkg/model"
	"time"
)

// RequestCreatedEvent matches the schema for "request.created.event"
type RequestCreatedEvent struct {
	RequestID   string            `json:"requestId"`
	PassengerID string            `json:"passengerId"`
	Origin      model.GeoLocation `json:"origin"`
	Destination model.GeoLocation `json:"destination"`
	DesiredTime time.Time         `json:"desiredTime"`
	Passengers  int               `json:"passengers"`
	Preferences model.Preferences `json:"preferences"`
	Status      string            `json:"status"`
	Timestamp   time.Time         `json:"timestamp"`
}

// RequestCancelledEvent matches the schema for "request.cancelled.event"
type RequestCancelledEvent struct {
	RequestID   string    `json:"requestId"`
	PassengerID string    `json:"passengerId"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// LogEvent matches the schema for "logging.log" topic
type LogEvent struct {
	ServiceID string `json:"serviceID"`
	Topic     string `json:"topic"`
	Message   string `json:"message"`
}
