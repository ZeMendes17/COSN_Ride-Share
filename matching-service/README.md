# Matching Service

The **Matching Service** is responsible for consuming events produced by the Offer Service (via the Kafka message broker), processing potential matches for passenger ride requests, applying additional matching logic, and storing match data locally.  

---

## Responsibilities

- Consume incoming Kafka events from external microservices (e.g., `OfferMatched`)
- Apply matching and ranking logic:
    - Compare trip offers to passenger requests
    - Apply scoring or filtering rules
    - Handle re-matching
- Persist match data in the service's own database (`matches_db`)
- Publish events to Kafka, such as:
    - `match.created.event`
    - `match.cancelled.event`
- Expose HTTP endpoints for match history and actions

---

# Project Structure

## `cmd/matching-service/`

### **main.go**
- Entry point of the microservice.
- Sets up:
    - configuration loading
    - logger
    - database connection
    - Kafka consumer (and Kafka producer)
    - HTTP server
- Should remain minimal, delegating all logic to internal modules.

---

## `internal/matcher/`

This folder implements the **core business logic** for handling consumed events and performing matching functions.

### **service.go**
- Defines the `Service` interface.
- Contains domain-specific logic, such as:
    - Handling incoming offer match events
    - Scoring or ranking matches
    - Applying business rules (re-match, expiration, etc.)
- Contains **no networking, Kafka, or database code**.

### **endpoints.go**
- Go Kit endpoint definitions for any HTTP endpoints exposed by the matching service.

### **transport_http.go**
- HTTP handlers for incoming REST requests.
- Translates HTTP <--> endpoint <--> service.
- Uses Go Kit HTTP transport.

---

## `internal/kafka/`

### **consumer.go**
- Responsible for **consuming events** from Kafka.
- Listens to topics such as:
    - `offer.tripAvailable.event`
    - `offer.updateOffer.event`
- Decodes event messages and forwards them to the matcher service.
- Does not configure or manage Kafka infrastructure — only subscribes and consumes.

### **producer.go**
- Publishes domain events back to Kafka, such as:
    - `MatchFound`
    - `MatchExpired`
    - `TripCompleted`
- Mirrors the pattern used in the Request Service.

---

## `internal/db/`

### **repository.go**
- Defines the repository interface for interacting with the local database (`matches_db`).
- Includes operations such as:
    - Insert new match record
    - Update match status
    - Select Match
- Keeps persistence concerns isolated from core business logic.

---

## `internal/config/`

### **config.go**
- Handles configuration loading for:
    - Kafka broker address
    - Kafka consumer group ID
    - Database DSN
    - HTTP port (if HTTP is exposed)
- Loads environment variables and default values.

---

## `pkg/contracts/`

This folder contains **event schemas** used when consuming or publishing Kafka messages.

### **events.go**
- Defines the data structures for events such as:
    - `OfferMatched` (consumed)
    - `MatchFound`, `MatchExpired`, `TripCompleted` (produced)
- Ensures consistency across microservices.

---

## Summary

The Matching Service is designed as an event-driven microservice following Go Kit principles.  
It reacts to events, applies business logic, maintains its own data store, and publishes new events to Kafka.  
Each internal layer has a clear responsibility, ensuring modularity, maintainability, and clean separation of concerns.

# How to test the Matching Service

## Scenario A

1. Create a Request (Passenger)

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123' \
  -H 'Content-Type: application/json' \
  -d '{
    "passengerID": "passenger-123",
    "origin": {"lat": 41.1579, "lon": 8.6291},
    "destination": {"lat": 38.7223, "lon": 9.1393},
    "desiredTime": "2025-12-30T09:00:00Z",
    "passengers": 2,
    "preferences": {"smoking": false, "pets": false, "music": true}
  }'
```
2. Send an Offer (Simulated)

Change the REQ_ID using the result from step 1.

```bash
curl -X POST http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/test/offer \
  -H "Content-Type: application/json" \
  -d '{
    "offerId": "101",
    "driverId": "505",
    "driverName": "Tiago",
    "originLat": 41.1579,
    "originLon": 8.6291,
    "destinyLat": 38.7223,
    "destinyLon": 9.1393,
    "departureTimeMin": "2025-12-30T08:50:00Z",
    "departureTimeMax": "2025-12-30T09:10:00Z",
    "availableSeats": 4,
    "expectedDuration": "03:15",
    "triggerRequest": [
        { "requesterId": "{REQ_ID}" }
    ]
  }'

```

3. Verify Match Created

Change the REQ_ID using the result from step 1.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/requests/{REQ_ID}' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```

4. Accept the Match

Change the REQ_ID using the result from step 1.

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/{MATCH_ID}/requests/{REQ_ID}' \
  -H 'accept: */*' \
  -H 'Authorization: passenger-123'
```

5. Verify Request Status

Should appear as `Completed`

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```

## Scenario B

1. Create a Request

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-789' \
  -H 'Content-Type: application/json' \
  -d '{
    "passengerID": "passenger-789",
    "origin": {"lat": 40.4168, "lon": -3.7038}, 
    "destination": {"lat": 38.7223, "lon": 9.1393},
    "desiredTime": "2025-12-30T09:00:00Z",
    "passengers": 1,
    "preferences": {"smoking": false, "pets": false, "music": false}
  }'
```

2. Send an Offer (Far Away from the Request)

Change the REQ_ID using the result from step 1.

```bash
curl -X POST http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/test/offer \
  -H "Content-Type: application/json" \
  -d '{
    "offerId": "101",
    "driverId": "505",
    "driverName": "Tiago",
    "originLat": 41.1579,
    "originLon": 8.6291,
    "destinyLat": 38.7223,
    "destinyLon": 9.1393,
    "departureTimeMin": "2025-12-30T09:00:00Z",
    "departureTimeMax": "2025-12-30T09:10:00Z",
    "availableSeats": 4,
    "expectedDuration": "03:15",
    "triggerRequest": [
        { "requesterId": "{REQ_ID}" }
    ]
  }'
```

3. Verify no Match

Change the REQ_ID using the result from step 1.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/requests/{REQ_ID}' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-789'
```

## Scenario C

1. Create a Request

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-456' \
  -H 'Content-Type: application/json' \
  -d '{
    "passengerID": "passenger-456",
    "origin": {"lat": 41.1500, "lon": 8.6100},
    "destination": {"lat": 38.7200, "lon": 9.1400},
    "desiredTime": "2025-12-30T10:00:00Z",
    "passengers": 1,
    "preferences": {"smoking": true, "pets": false, "music": true}
  }'
```

2. Send Initial Offer

```bash
curl -X POST http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/test/offer \
  -H "Content-Type: application/json" \
  -d '{
    "offerId": "202",
    "driverId": "606",
    "driverName": "Helder",
    "originLat": 41.1500,
    "originLon": 8.6100,
    "destinyLat": 38.7200,
    "destinyLon": 9.1400,
    "departureTimeMin": "2025-12-30T09:30:00Z",
    "departureTimeMax": "2025-12-30T10:30:00Z",
    "availableSeats": 3,
    "expectedDuration": "03:15"
  }'
```

3. Verify First Match

Change the REQ_ID using the result from step 1.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/requests/{REQ_ID}' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-456'
```

4. Send Offer Update

```bash
curl -X POST http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/test/offer/update \
  -H "Content-Type: application/json" \
  -d '{
    "offerId": "202",
    "driverId": "606",
    "driverName": "Helder",
    "originLat": 41.1500,
    "originLon": 8.6100,
    "destinyLat": 38.7200,
    "destinyLon": 9.1400,
    "departureTimeMin": "2025-12-30T09:30:00Z",
    "departureTimeMax": "2025-12-30T10:30:00Z",
    "availableSeats": 4, 
    "expectedDuration": "03:15"
  }'
```

5. Verify Updated Match

Change the REQ_ID using the result from step 1.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/requests/{REQ_ID}' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-456'
```

The match ID is now different.

## Scenario D

1. Create a Request

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123' \
  -H 'Content-Type: application/json' \
  -d '{
    "passengerID": "passenger-123",
    "origin": {"lat": 41.1579, "lon": 8.6291},
    "destination": {"lat": 38.7223, "lon": 9.1393},
    "desiredTime": "2025-12-30T09:00:00Z",
    "passengers": 2,
    "preferences": {"smoking": false, "pets": false, "music": true}
  }'
```

2. Create a Match

Change the REQ_ID using the result from step 1.

```bash
curl -X POST http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/test/offer \
  -H "Content-Type: application/json" \
  -d '{
    "offerId": "101",
    "driverId": "505",
    "driverName": "Tiago",
    "originLat": 41.1579,
    "originLon": 8.6291,
    "destinyLat": 38.7223,
    "destinyLon": 9.1393,
    "departureTimeMin": "2025-12-30T08:50:00Z",
    "departureTimeMax": "2025-12-30T09:10:00Z",
    "availableSeats": 4,
    "preferences": {"smoking": false, "pets": false, "music": true},
    "expectedDuration": "03:15",
    "triggerRequest": [
        { "requesterId": "{REQ_ID}" }
    ]
  }'

```

3. Retrieve and Verify Matches

Change the REQ_ID using the result from step 1.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/requests/{REQ_ID}' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```

4. Select the Match

Change the REQ_ID using the result from step 1.

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/{MATCH_ID}/requests/{REQ_ID}' \
  -H 'accept: */*' \
  -H 'Authorization: passenger-123'
```

Check the request status, it should be `Completed`.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```

5. Cancel the Match

Change the REQ_ID using the result from step 1.
Change the MATCH_ID using the result from step 3.

```bash
curl -X 'DELETE' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/matches/{MATCH_ID}/requests/{REQ_ID}' \
  -H 'accept: */*' \
  -H 'Authorization: passenger-123'
```

Check the request status, it should be back to `Pending`.

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```