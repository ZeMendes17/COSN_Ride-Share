package sns

import (
	"context"
	"encoding/json"
	"log"

	"matching-service/pkg/contracts"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

const (
	TopicMatchCreated   = "match.created.event"
	TopicMatchCancelled = "match.cancelled.event"
	TopicLogging        = "logging.log"
)

// Producer defines the interface for publishing messages to SNS
type Producer interface {
	SendMatchCreated(ctx context.Context, event contracts.MatchCreatedEvent) error
	SendMatchCancelled(ctx context.Context, event contracts.MatchCancelledEvent) error
	SendLog(ctx context.Context, event contracts.LogEvent) error
}

// SNSProducer implements the Producer interface using AWS SNS
type SNSProducer struct {
	client            *sns.SNS
	matchCreatedARN   string
	matchCancelledARN string
	loggingARN        string
}

// NewSNSProducer creates a new SNS producer
func NewSNSProducer(region, accessKey, secretKey, matchCreatedARN, matchCancelledARN, loggingARN string) (*SNSProducer, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	return &SNSProducer{
		client:            sns.New(sess),
		matchCreatedARN:   matchCreatedARN,
		matchCancelledARN: matchCancelledARN,
		loggingARN:        loggingARN,
	}, nil
}

func (p *SNSProducer) SendMatchCreated(ctx context.Context, event contracts.MatchCreatedEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.PublishWithContext(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.matchCreatedARN),
		Message:  aws.String(string(bytes)),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"eventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(TopicMatchCreated),
			},
			"matchId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.MatchID),
			},
		},
	})

	if err != nil {
		log.Printf("SNS ERROR [%s]: Failed to send Match Created %s: %v", TopicMatchCreated, event.MatchID, err)
		return err
	}

	log.Printf("SNS [%s]: Sent Match Created %s for Request %s", TopicMatchCreated, event.MatchID, event.RequestID)
	return nil
}

func (p *SNSProducer) SendMatchCancelled(ctx context.Context, event contracts.MatchCancelledEvent) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = p.client.PublishWithContext(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.matchCancelledARN),
		Message:  aws.String(string(bytes)),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"eventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(TopicMatchCancelled),
			},
			"matchId": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.MatchID),
			},
		},
	})

	if err != nil {
		log.Printf("SNS ERROR [%s]: Failed to send Match Cancelled %s: %v", TopicMatchCancelled, event.MatchID, err)
		return err
	}

	log.Printf("SNS [%s]: Sent Match Cancelled %s", TopicMatchCancelled, event.MatchID)
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

// NoOpProducer implements the Producer interface but does nothing (for testing/mock)
type NoOpProducer struct{}

func NewNoOpProducer() *NoOpProducer {
	return &NoOpProducer{}
}

func (p *NoOpProducer) SendMatchCreated(ctx context.Context, event contracts.MatchCreatedEvent) error {
	log.Printf("SNS MOCK [%s]: Sent Match Created %s for Request %s", TopicMatchCreated, event.MatchID, event.RequestID)
	return nil
}

func (p *NoOpProducer) SendMatchCancelled(ctx context.Context, event contracts.MatchCancelledEvent) error {
	log.Printf("SNS MOCK [%s]: Sent Match Cancelled %s", TopicMatchCancelled, event.MatchID)
	return nil
}

func (p *NoOpProducer) SendLog(ctx context.Context, event contracts.LogEvent) error {
	log.Printf("KAFKA MOCK [%s] [%s]: %s", TopicLogging, event.Topic, event.Message)
	return nil
}
