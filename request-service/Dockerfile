# Stage 1: Build the application
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy modules manifests first for better caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application (no CGO needed for DynamoDB)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o request-service cmd/request-service/main.go

# Stage 2: Create minimal runtime image
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/request-service .


# Expose the port defined in config
EXPOSE 8080

# Command to run the executable
CMD ["./request-service"]