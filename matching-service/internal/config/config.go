package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr                  string
	RequestServiceURL         string
	AWSRegion                 string
	DBSecretARN               string
	DBHost                    string
	DBPort                    string
	DBName                    string
	DBUser                    string
	DBPassword                string
	DBSSLMode                 string
	SQSTripAvailableURL       string
	SQSTripAvailableAccessKey string
	SQSTripAvailableSecretKey string
	SQSUpdateOfferURL         string
	SQSUpdateOfferAccessKey   string
	SQSUpdateOfferSecretKey   string
	SNSAccessKey              string
	SNSSecretKey              string
	SNSMatchCreatedARN        string
	SNSMatchCancelledARN      string
	SNSLoggingARN             string
}

func LoadConfig() Config {
	return Config{
		HTTPAddr:                  getEnv("HTTP_ADDR", ":8081"),
		RequestServiceURL:         getEnv("REQUEST_SERVICE_URL", "http://request-service:8080"),
		AWSRegion:                 getEnv("AWS_REGION", "us-east-1"),
		DBSecretARN:               getEnv("DB_SECRET_ARN", ""),
		DBHost:                    getEnv("DB_HOST", "localhost"),
		DBPort:                    getEnv("DB_PORT", "5432"),
		DBName:                    getEnv("DB_NAME", "carpooling"),
		DBUser:                    getEnv("DB_USER", "dbadmin"),
		DBPassword:                getEnv("DB_PASSWORD", ""),
		DBSSLMode:                 getEnv("DB_SSL_MODE", "require"),
		SQSTripAvailableURL:       getEnv("SQS_TRIP_AVAILABLE_URL", ""),
		SQSTripAvailableAccessKey: getEnv("SQS_TRIP_AVAILABLE_ACCESS_KEY", ""),
		SQSTripAvailableSecretKey: getEnv("SQS_TRIP_AVAILABLE_SECRET_KEY", ""),
		SQSUpdateOfferURL:         getEnv("SQS_UPDATE_OFFER_URL", ""),
		SQSUpdateOfferAccessKey:   getEnv("SQS_UPDATE_OFFER_ACCESS_KEY", ""),
		SQSUpdateOfferSecretKey:   getEnv("SQS_UPDATE_OFFER_SECRET_KEY", ""),
		SNSAccessKey:              getEnv("SNS_ACCESS_KEY", ""),
		SNSSecretKey:              getEnv("SNS_SECRET_KEY", ""),
		SNSMatchCreatedARN:        getEnv("SNS_MATCH_CREATED_ARN", ""),
		SNSMatchCancelledARN:      getEnv("SNS_MATCH_CANCELLED_ARN", ""),
		SNSLoggingARN:             getEnv("SNS_LOGGING_ARN", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}
