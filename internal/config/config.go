package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Region  string   `json:"region"`
	Regions []string `json:"regions"`
}

// LoadConfig loads the configuration from a file
func LoadConfig() (*Config, error) {
	// For now, we'll just use a default config
	return &Config{
		Region:  GetDefaultRegion(),
		Regions: []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2", "eu-central-1", "eu-west-1", "eu-west-2", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1"},
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
