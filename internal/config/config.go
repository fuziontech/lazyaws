package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Region string `json:"region"`
}

// LoadConfig loads the configuration from a file
func LoadConfig() (*Config, error) {
	// For now, we'll just use a default config
	return &Config{
		Region: GetDefaultRegion(),
	}, nil
}

// GetDefaultRegion returns the default AWS region
func GetDefaultRegion() string {
	if region, ok := os.LookupEnv("AWS_REGION"); ok {
		return region
	}
	if region, ok := os.LookupEnv("AWS_DEFAULT_REGION"); ok {
		return region
	}
	return "us-east-1"
}
