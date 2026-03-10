package sns

import (
	"context"
	"encoding/json"
	"log"

	"request-service/pkg/contracts"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

const (
	TopicRequestCreated   = "request.created.event"
	TopicRequestCancelled = "request.cancelled.event"
	TopicLogging          = "logging.log"
)

// Producer defines the interface for publishing request events to SNS
type Producer interface {
	SendRequestCreated(ctx context.Context, event contracts.RequestCreatedEvent) error
	SendRequestCancelled(ctx context.Context, event contracts.RequestCancelledEvent) error
	SendLog(ctx context.Context, event contracts.LogEvent) error
}

// SNSProducer implements the Producer interface using AWS SNS
type SNSProducer struct {
	client              *sns.SNS
	requestCreatedARN   string
	requestCancelledARN string
	loggingARN          string
}

// NewSNSProducer creates a new SNS producer
func NewSNSProducer(region, accessKey, secretKey, requestCreatedARN, requestCancelledARN string, loggingARN string) (*SNSProducer, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	return &SNSProducer{
		client:              sns.New(sess),
		requestCreatedARN:   requestCreatedARN,
		requestCancelledARN: requestCancelledARN,
		loggingARN:          loggingARN,
	}, nil
}

func (p *SNSProducer) SendRequestCreated(ctx context.Context, event contracts.RequestCreatedEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.PublishWithContext(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.requestCreatedARN),
		Message:  aws.String(string(bytes)),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"eventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(TopicRequestCreated),
			},
			"requestId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.RequestID),
			},
		},
	})

	if err != nil {
		log.Printf("SNS ERROR [%s]: Failed to send Request Created %s: %v", TopicRequestCreated, event.RequestID, err)
		return err
	}

	log.Printf("SNS [%s]: Sent Request Created %s for Passenger %s", TopicRequestCreated, event.RequestID, event.PassengerID)
	return nil
}

func (p *SNSProducer) SendRequestCancelled(ctx context.Context, event contracts.RequestCancelledEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.PublishWithContext(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.requestCancelledARN),
		Message:  aws.String(string(bytes)),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"eventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(TopicRequestCancelled),
			},
			"requestId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.RequestID),
			},
		},
	})

	if err != nil {
		log.Printf("SNS ERROR [%s]: Failed to send Request Cancelled %s: %v", TopicRequestCancelled, event.RequestID, err)
		return err
	}

	log.Printf("SNS [%s]: Sent Request Cancelled %s (Reason: %s)", TopicRequestCancelled, event.RequestID, event.Reason)
	return nil
}

func (p *SNSProducer) SendLog(ctx context.Context, event contracts.LogEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.PublishWithContext(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.loggingARN),
		Message:  aws.String(string(bytes)),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"eventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(TopicLogging),
			},
			"serviceID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.ServiceID),
			},
		},
	})

	if err != nil {
		log.Printf("SNS ERROR [%s]: Failed to send log event: %v", TopicLogging, err)
		return err
	}

	log.Printf("SNS [%s] [%s]: %s", TopicLogging, event.Topic, event.Message)
	return nil
}

// NoOpProducer is a producer that does nothing (for testing)
type NoOpProducer struct{}

func NewNoOpProducer() *NoOpProducer {
	return &NoOpProducer{}
}

func (p *NoOpProducer) SendRequestCreated(ctx context.Context, event contracts.RequestCreatedEvent) error {
	log.Printf("SNS MOCK [%s]: Created event for RequestID: %s", TopicRequestCreated, event.RequestID)
	return nil
}

func (p *NoOpProducer) SendRequestCancelled(ctx context.Context, event contracts.RequestCancelledEvent) error {
	log.Printf("SNS MOCK [%s]: Cancelled event for RequestID: %s (Reason: %s)", TopicRequestCancelled, event.RequestID, event.Reason)
	return nil
}

func (p *NoOpProducer) SendLog(ctx context.Context, event contracts.LogEvent) error {
	log.Printf("KAFKA MOCK [%s] [%s]: %s", TopicLogging, event.Topic, event.Message)
	return nil
}
