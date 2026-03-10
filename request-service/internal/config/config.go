package config

import (
	"fmt"
	"os"
)

// Config holds all necessary application configuration settings.
type Config struct {
	HTTPAddr               string
	AWSRegion              string
	DBSecretARN            string
	DBHost                 string
	DBPort                 string
	DBName                 string
	DBUser                 string
	DBPassword             string
	DBSSLMode              string
	SNSRequestCreatedARN   string
	SNSRequestCancelledARN string
	SNSLoggingARN          string
	ExternalAWSAccessKey   string
	ExternalAWSSecretKey   string
	LogLevel               string
}

// LoadConfig loads configuration from environment variables with defaults.
func LoadConfig() Config {
	return Config{
		HTTPAddr:               getEnv("HTTP_ADDR", ":8080"),
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		DBSecretARN:            getEnv("DB_SECRET_ARN", ""),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBName:                 getEnv("DB_NAME", "carpooling"),
		DBUser:                 getEnv("DB_USER", "dbadmin"),
		DBPassword:             getEnv("DB_PASSWORD", ""),
		DBSSLMode:              getEnv("DB_SSL_MODE", "require"),
		SNSRequestCreatedARN:   getEnv("SNS_REQUEST_CREATED_ARN", ""),
		SNSRequestCancelledARN: getEnv("SNS_REQUEST_CANCELLED_ARN", ""),
		SNSLoggingARN:          getEnv("SNS_LOGGING_ARN", ""),
		ExternalAWSAccessKey:   getEnv("EXTERNAL_AWS_ACCESS_KEY", ""),
		ExternalAWSSecretKey:   getEnv("EXTERNAL_AWS_SECRET_KEY", ""),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
	}
}

// GetDBConnString builds a PostgreSQL connection string from config
func (c *Config) GetDBConnString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
