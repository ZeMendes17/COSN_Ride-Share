package sqs

import (
	"context"
	"encoding/json"

	"matching-service/internal/matching"
	"matching-service/pkg/contracts"
	"matching-service/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-kit/kit/log"
)

// Consumer defines the interface for consuming messages from SQS
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// SQSConsumer implements the Consumer interface using AWS SQS
type SQSConsumer struct {
	tripAvailableClient *sqs.SQS
	updateOfferClient   *sqs.SQS
	tripAvailableURL    string
	updateOfferURL      string
	service             matching.Service
	logger              log.Logger
	stopChan            chan struct{}
}

// NewSQSConsumer creates a new SQS consumer with separate credentials for each queue
func NewSQSConsumer(region, tripAvailableAccessKey, tripAvailableSecretKey, tripAvailableURL, updateOfferAccessKey, updateOfferSecretKey, updateOfferURL string, service matching.Service, logger log.Logger) (*SQSConsumer, error) {
	// Create client for trip available queue
	tripAvailableSess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(tripAvailableAccessKey, tripAvailableSecretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	// Create client for update offer queue
	updateOfferSess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(updateOfferAccessKey, updateOfferSecretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	return &SQSConsumer{
		tripAvailableClient: sqs.New(tripAvailableSess),
		updateOfferClient:   sqs.New(updateOfferSess),
		tripAvailableURL:    tripAvailableURL,
		updateOfferURL:      updateOfferURL,
		service:             service,
		logger:              logger,
		stopChan:            make(chan struct{}),
	}, nil
}

func (c *SQSConsumer) Start(ctx context.Context) error {
	c.logger.Log("msg", "Starting SQS consumer",
		"tripAvailableURL", c.tripAvailableURL,
		"updateOfferURL", c.updateOfferURL)

	// Start goroutines for each queue with its own client
	go c.consumeQueue(ctx, c.tripAvailableClient, c.tripAvailableURL, "tripAvailable")
	go c.consumeQueue(ctx, c.updateOfferClient, c.updateOfferURL, "updateOffer")

	return nil
}

func (c *SQSConsumer) Stop() error {
	close(c.stopChan)
	return nil
}

func (c *SQSConsumer) consumeQueue(ctx context.Context, client *sqs.SQS, queueURL, queueName string) {
	_ = c.logger.Log("msg", "Started consuming queue", "queue", queueName, "url", queueURL)

	for {
		select {
		case <-c.stopChan:
			_ = c.logger.Log("msg", "Stopping queue consumer", "queue", queueName)
			return
		case <-ctx.Done():
			_ = c.logger.Log("msg", "Context cancelled, stopping queue consumer", "queue", queueName)
			return
		default:
			// Receive messages from SQS
			result, err := client.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueURL),
				MaxNumberOfMessages: aws.Int64(10),
				WaitTimeSeconds:     aws.Int64(20), // Long polling
				VisibilityTimeout:   aws.Int64(30),
			})

			if err != nil {
				_ = c.logger.Log("error", "Failed to receive messages", "queue", queueName, "err", err)
				continue
			}

			for _, message := range result.Messages {
				if err := c.processMessage(ctx, message, queueName); err != nil {
					_ = c.logger.Log("error", "Failed to process message", "queue", queueName, "err", err)
					continue
				}

				// Delete message after successful processing
				_, err := client.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueURL),
					ReceiptHandle: message.ReceiptHandle,
				})

				if err != nil {
					_ = c.logger.Log("error", "Failed to delete message", "queue", queueName, "err", err)
				}
			}
		}
	}
}

func (c *SQSConsumer) processMessage(ctx context.Context, message *sqs.Message, queueName string) error {
	_ = c.logger.Log("msg", "Received message", "queue", queueName, "messageId", *message.MessageId)

	// Handle SNS wrapped messages
	var snsMessage struct {
		Message string `json:"Message"`
	}

	messageBody := *message.Body

	// Try to unwrap SNS message first
	if err := json.Unmarshal([]byte(messageBody), &snsMessage); err == nil && snsMessage.Message != "" {
		messageBody = snsMessage.Message
	}

	// Process based on queue name
	switch queueName {
	case "tripAvailable":
		return c.processTripAvailable(ctx, messageBody)
	case "updateOffer":
		return c.processOfferUpdate(ctx, messageBody)
	default:
		_ = c.logger.Log("error", "Unknown queue name", "queue", queueName)
		return nil
	}
}

// ProcessTripAvailable handles mapping from Contract -> Model
func (c *SQSConsumer) processTripAvailable(ctx context.Context, messageBody string) error {
	var event contracts.OfferTripAvailableEvent
	if err := json.Unmarshal([]byte(messageBody), &event); err != nil {
		_ = c.logger.Log("error", "Failed to unmarshal trip available event", "err", err)
		return err
	}

	_ = c.logger.Log("msg", "Processing Offer Available", "offerId", event.OfferID, "driverName", event.DriverName)

	domainOffer := model.Offer{
		OfferID:          event.OfferID,
		DriverID:         event.DriverID,
		DriverName:       event.DriverName,
		Origin:           model.GeoLocation{Lat: event.OriginLat, Lon: event.OriginLon},
		Destination:      model.GeoLocation{Lat: event.DestinyLat, Lon: event.DestinyLon},
		AvailableSeats:   event.AvailableSeats,
		DepartureTimeMin: event.DepartureTimeMin,
		DepartureTimeMax: event.DepartureTimeMax,
		Waypoints:        event.Waypoints,
	}

	// Aggregate all Trigger IDs from the array
	var triggerReqIDs []string
	if event.TriggerRequest != nil {
		for _, tr := range event.TriggerRequest {
			if tr.RequesterID != "" {
				triggerReqIDs = append(triggerReqIDs, tr.RequesterID)
			}
			if tr.PendingRequestIds != nil {
				triggerReqIDs = append(triggerReqIDs, tr.PendingRequestIds...)
			}
		}
	}

	return c.service.ProcessOffer(ctx, domainOffer, triggerReqIDs)
}

// ProcessOfferUpdate handles mapping from Contract -> Model
func (c *SQSConsumer) processOfferUpdate(ctx context.Context, messageBody string) error {
	var event contracts.OfferUpdateEvent
	if err := json.Unmarshal([]byte(messageBody), &event); err != nil {
		_ = c.logger.Log("error", "Failed to unmarshal offer update event", "err", err)
		return err
	}

	_ = c.logger.Log("msg", "Processing Offer Update", "offerId", event.OfferID)

	domainOffer := model.Offer{
		OfferID:          event.OfferID,
		DriverID:         event.DriverID,
		DriverName:       event.DriverName,
		Origin:           model.GeoLocation{Lat: event.OriginLat, Lon: event.OriginLon},
		Destination:      model.GeoLocation{Lat: event.DestinyLat, Lon: event.DestinyLon},
		AvailableSeats:   event.AvailableSeats,
		DepartureTimeMin: event.DepartureTimeMin,
		DepartureTimeMax: event.DepartureTimeMax,
		Waypoints:        event.Waypoints,
	}

	// Updates are broadcast (nil triggers)
	return c.service.ProcessOffer(ctx, domainOffer, nil)
}

// NoOpConsumer is a consumer that does nothing (for testing)
type NoOpConsumer struct{}

func NewNoOpConsumer() *NoOpConsumer {
	return &NoOpConsumer{}
}

func (c *NoOpConsumer) Start(ctx context.Context) error {
	return nil
}

func (c *NoOpConsumer) Stop() error {
	return nil
}
