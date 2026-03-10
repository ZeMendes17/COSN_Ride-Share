# Request Service

The **Request Service** is responsible for receiving passenger ride requests, validating them, storing them locally, and publishing events to the system via Kafka  (the message broker).  
This microservice does **not** perform any trip matching.

## Responsibilities

- Expose HTTP endpoints to:
  - Create a ride request
  - Cancel a ride request
- Validate request input
- Persist request data in the service's own database (`requests_db`)
- Publish `request.created.event` and `request.cancelled.event` events to Kafka  

# Project Structure

## `cmd/request-service/`

### **main.go**
- Entry point of the microservice.
- Should remain very small, as all logic is delegated to internal layers.

---

## `internal/request/`

This folder implements the **business logic** for handling ride requests.

### **service.go**
- Defines the `Service` interface.
- Contains core business rules:
  - Create request
  - Cancel request
  - Validate request  
- No networking or database code in this file.

### **endpoints.go**
- Go Kit endpoint definitions.
- Each endpoint adapts a service method to an RPC/public interface.
- This is where you define request/response structs for API calls.

### **transport_http.go**
- HTTP handlers for incoming REST requests.
- Translates HTTP <--> endpoint <--> service.
- Uses Go Kit HTTP transport.

---

## `internal/kafka/`

### **producer.go**
- Responsible for sending messages to Kafka (events only).
- Publish:
  - `request.created.event`
  - `request.cancelled.event`
- Does not configure Kafka—just sends messages.

---

## `internal/db/`

### **repository.go**
- Defines the repository interface for interacting with the local database.
- Includes:
  - Insert request
  - Update status
  - Fetch request (optional)

---

## `internal/config/`

### **config.go**
- Loads configuration values:
  - Kafka broker URL
  - Database DSN
  - HTTP port
- Includes environment variable parsing.

---

## `pkg/contracts/`

This folder contains **message schemas** that other services rely on.

### **events.go**
- Defines the structure of events published to Kafka.
- Ensures all microservices agree on the same event format.

---

# How to test the Request Service

1. Create a Request

```bash
curl -X 'POST' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123' \
  -H 'Content-Type: application/json' \
  -d '{
    "passengerID": "passenger-123",
    "origin": {"lat": 40.7128, "lon": -74.0060},
    "destination": {"lat": 40.7306, "lon": -73.9352},
    "desiredTime": "2025-12-31T20:00:00Z",
    "passengers": 2,
    "preferences": {"smoking": false, "pets": false, "music": true}
  }'
```

2. Get All Requests

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```

3. Get All Pending Requests

```bash
curl -X 'GET' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/pending' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123'
```

4. Patch a Request

```bash
curl -X 'PATCH' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/{requestID}' \
  -H 'accept: application/json' \
  -H 'Authorization: passenger-123' \
  -H 'Content-Type: application/json' \
  -d '{
    "passengers": 4
  }'
```
5. Cancel a Request

```bash
curl -X 'DELETE' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/{requestID}' \
  -H 'accept: */*' \
  -H 'Authorization: passenger-123'
```

6. Test Security
Pass in requestID one that doesn't belong to the user in the Authorization Header.
```bash
curl -X 'DELETE' \
  'http://k8s-carpooli-carpooli-b929a24afd-1853003854.us-east-1.elb.amazonaws.com/requests/{requestID}' \
  -H 'accept: */*' \
  -H 'Authorization: hacker-456'
```