package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SSOConfig holds the SSO configuration
type SSOConfig struct {
	StartURL string `json:"start_url"`
	Region   string `json:"region"`
}

// GetSSOConfigPath returns the path to the SSO config file
func GetSSOConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".lazyaws")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "sso-config.json"), nil
}

// LoadSSOConfig loads the SSO configuration from disk
func LoadSSOConfig() (*SSOConfig, error) {
	configPath, err := GetSSOConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file yet
		}
		return nil, err
	}

	var config SSOConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveSSOConfig saves the SSO configuration to disk
func SaveSSOConfig(config *SSOConfig) error {
	configPath, err := GetSSOConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return err
	}

	return nil
}

// ValidateSSOStartURL performs basic validation on the SSO start URL
func ValidateSSOStartURL(url string) error {
	if url == "" {
		return fmt.Errorf("SSO start URL cannot be empty")
	}

	// Basic validation - should start with https://
	if len(url) < 8 || url[:8] != "https://" {
		return fmt.Errorf("SSO start URL must start with https://")
	}

	// Should contain awsapps.com (typical AWS SSO domain)
	// But we'll be lenient and accept any https URL
	if len(url) < 10 {
		return fmt.Errorf("SSO start URL is too short")
	}

	return nil
}
