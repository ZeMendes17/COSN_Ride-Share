package matching

import (
	"matching-service/pkg/model"
	"math"
	"time"
)

const (
	EarthRadiusKm       = 6371.0
	DefaultBaseDistance = 1.0 // 1 km base radius
	AnnealingExpansion  = 0.5 // 500m expansion per interval
	MeanVelocityKmH     = 50.0
)

type AlgorithmService struct{}

func (alg *AlgorithmService) CalculateMatch(req model.Request, offer model.Offer, maxDistanceKm float64) (model.Match, bool) {
	// 1. Filter for Available Seats
	if req.Passengers > offer.AvailableSeats {
		return model.Match{}, false
	}

	// 2. Filter for Preferences: if the request requires a preference (true),
	if req.Preferences.Smoking && !offer.Preferences.Smoking {
		return model.Match{}, false
	}
	if req.Preferences.Pets && !offer.Preferences.Pets {
		return model.Match{}, false
	}
	if req.Preferences.Music && !offer.Preferences.Music {
		return model.Match{}, false
	}

	// 3. Filter for Distance
	distToPath := pointToSegmentDistance(
		req.Origin.Lat, req.Origin.Lon,
		offer.Origin.Lat, offer.Origin.Lon,
		offer.Destination.Lat, offer.Destination.Lon,
	)

	if distToPath > maxDistanceKm {
		return model.Match{}, false
	}

	// 4. Filter for Time
	distFromOfferStart := haversine(offer.Origin.Lat, offer.Origin.Lon, req.Origin.Lat, req.Origin.Lon)
	hoursToPickup := distFromOfferStart / MeanVelocityKmH
	estimatedPickupTime := offer.DepartureTimeMin.Add(time.Duration(hoursToPickup * float64(time.Hour)))
	if estimatedPickupTime.After(req.DesiredTime.Add(30 * time.Minute)) {
		return model.Match{}, false
	}

	return model.Match{
		OfferID:             offer.OfferID,
		RequestID:           req.ID,
		DriverID:            offer.DriverID,
		PassengerID:         req.PassengerID,
		PickupLocation:      req.Origin,
		EstimatedPickupTime: estimatedPickupTime,
		RankingScore:        100.0 - (distToPath * 10.0), // Simple ranking based on distance
		Status:              "Created",
	}, true
}

// --- Helper Math Functions ---

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusKm * c
}

func pointToSegmentDistance(pLat, pLon, aLat, aLon, bLat, bLon float64) float64 {
	return haversine(pLat, pLon, aLat, aLon)
}
