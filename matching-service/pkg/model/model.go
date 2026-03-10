package model

import (
	"errors"
	"time"
)

var (
	ErrMatchNotFound = errors.New("match not found")
	ErrOfferExpired  = errors.New("offer expired or unavailable")
)

type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Request struct {
	ID          string      `json:"id"`
	PassengerID string      `json:"passengerID"`
	Origin      GeoLocation `json:"origin"`
	Destination GeoLocation `json:"destination"`
	DesiredTime time.Time   `json:"desiredTime"`
	Passengers  int         `json:"passengers"`
	Preferences Preferences `json:"preferences"`
	CreatedAt   time.Time   `json:"createdAt"`
}

type Offer struct {
	OfferID          string            `json:"offerId"`
	DriverID         string            `json:"driverId"`
	DriverName       string            `json:"driverName"`
	Origin           GeoLocation       `json:"origin"`
	Destination      GeoLocation       `json:"destination"`
	AvailableSeats   int               `json:"availableSeats"`
	Preferences      Preferences       `json:"preferences"`
	Waypoints        map[string]string `json:"waypoints"`
	DepartureTimeMin time.Time         `json:"departureTimeMin"`
	DepartureTimeMax time.Time         `json:"departureTimeMax"`
}

type Preferences struct {
	Smoking bool `json:"smoking"`
	Pets    bool `json:"pets"`
	Music   bool `json:"music"`
}

type Match struct {
	MatchID             string      `json:"matchId"`
	RequestID           string      `json:"requestId"`
	OfferID             string      `json:"offerId"`
	DriverID            string      `json:"driverId"`
	PassengerID         string      `json:"passengerId"`
	PickupLocation      GeoLocation `json:"pickupLocation"`
	EstimatedPickupTime time.Time   `json:"estimatedPickupTime"`
	Status              string      `json:"status"`
	RankingScore        float64     `json:"rankingScore"`
}
