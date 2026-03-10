package contracts

import (
	"time"
)

// LogEvent matches the schema for "logging.log" topic
type LogEvent struct {
	ServiceID string `json:"serviceID"`
	Topic     string `json:"topic"`
	Message   string `json:"message"`
}

// OfferTripAvailableEvent represents the 'offer.tripAvailable.event' topic
type OfferTripAvailableEvent struct {
	DriverName       string               `json:"driverName"`
	DriverID         string               `json:"driverId"`
	OfferID          string               `json:"offerId"`
	OriginLat        float64              `json:"originLat"`
	OriginLon        float64              `json:"originLon"`
	DestinyLat       float64              `json:"destinyLat"`
	DestinyLon       float64              `json:"destinyLon"`
	DepartureTimeMin time.Time            `json:"departureTimeMin"`
	DepartureTimeMax time.Time            `json:"departureTimeMax"`
	Waypoints        map[string]string    `json:"waypoints"`
	AvailableSeats   int                  `json:"availableSeats"`
	Preferences      *OfferPreferences    `json:"preferences,omitempty"`
	ExpectedDuration string               `json:"expectedDuration"`
	TriggerRequest   []TriggerRequestData `json:"triggerRequest"`
}

type TriggerRequestData struct {
	RequesterID           string    `json:"requesterId"`
	RequesterName         string    `json:"reqiesterName"`
	OriginLat             float64   `json:"originLat"`
	OriginLon             float64   `json:"originLon"`
	DestinyLat            float64   `json:"destinyLat"`
	DestinyLon            float64   `json:"destinyLon"`
	TimeframeDepartureMin time.Time `json:"timeframeDepartureMin"`
	TimeframeDepartureMax time.Time `json:"timeframeDepartureMax"`
	PendingRequestIds     []string  `json:"pendingRequestIds"`
}

// OfferUpdateEvent represents the 'offer.updateOffer.event' topic
type OfferUpdateEvent struct {
	DriverName       string            `json:"driverName"`
	DriverID         string            `json:"driverId"`
	OfferID          string            `json:"offerId"`
	OriginLat        float64           `json:"originLat"`
	OriginLon        float64           `json:"originLon"`
	DestinyLat       float64           `json:"destinyLat"`
	DestinyLon       float64           `json:"destinyLon"`
	DepartureTimeMin time.Time         `json:"departureTimeMin"`
	DepartureTimeMax time.Time         `json:"departureTimeMax"`
	Waypoints        map[string]string `json:"waypoints"`
	AvailableSeats   int               `json:"availableSeats"`
	Preferences      *OfferPreferences `json:"preferences,omitempty"`
	ExpectedDuration string            `json:"expectedDuration"`
}

// OfferPreferences describes simple boolean preferences an offer supports
type OfferPreferences struct {
	Smoking bool `json:"smoking"`
	Pets    bool `json:"pets"`
	Music   bool `json:"music"`
}

// MatchCreatedEvent (Outgoing)
type MatchCreatedEvent struct {
	MatchID             string    `json:"matchId"`
	RequestID           string    `json:"requestId"`
	OfferID             string    `json:"offerId"`
	DriverID            string    `json:"driverId"`
	PassengerID         string    `json:"passengerId"`
	EstimatedPickupTime time.Time `json:"estimatedPickupTime"`
	Timestamp           time.Time `json:"timestamp"`
}

// MatchCancelledEvent (Outgoing)
type MatchCancelledEvent struct {
	MatchID   string    `json:"matchId"`
	RequestID string    `json:"requestId"`
	OfferID   string    `json:"offerId"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}
