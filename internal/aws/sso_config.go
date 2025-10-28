package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AuthMethod represents the authentication method
type AuthMethod string

const (
	AuthMethodEnv     AuthMethod = "env"
	AuthMethodProfile AuthMethod = "profile"
	AuthMethodSSO     AuthMethod = "sso"
)

// AuthConfig holds the authentication configuration
type AuthConfig struct {
	Method      AuthMethod `json:"method"`
	ProfileName string     `json:"profile_name,omitempty"`
	SSOStartURL string     `json:"sso_start_url,omitempty"`
	SSORegion   string     `json:"sso_region,omitempty"`
}

// SSOConfig holds the SSO configuration (for backward compatibility)
type SSOConfig struct {
	StartURL string `json:"start_url"`
	Region   string `json:"region"`
}

// GetAuthConfigPath returns the path to the auth config file
func GetAuthConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".lazyaws")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// GetSSOConfigPath returns the path to the legacy SSO config file
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

// LoadAuthConfig loads the authentication configuration from disk
func LoadAuthConfig() (*AuthConfig, error) {
	configPath, err := GetAuthConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to migrate from old SSO config
			return migrateSSOConfig()
		}
		return nil, err
	}

	var config AuthConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// migrateSSOConfig attempts to load the old sso-config.json and migrate it
func migrateSSOConfig() (*AuthConfig, error) {
	ssoConfig, err := LoadSSOConfig()
	if err != nil || ssoConfig == nil {
		return nil, nil // No old config to migrate
	}

	// Migrate to new format
	authConfig := &AuthConfig{
		Method:      AuthMethodSSO,
		SSOStartURL: ssoConfig.StartURL,
		SSORegion:   ssoConfig.Region,
	}

	// Save in new format
	if err := SaveAuthConfig(authConfig); err == nil {
		// Delete old config file
		oldPath, _ := GetSSOConfigPath()
		os.Remove(oldPath)
	}

	return authConfig, nil
}

// LoadSSOConfig loads the SSO configuration from disk (legacy)
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

// SaveAuthConfig saves the authentication configuration to disk
func SaveAuthConfig(config *AuthConfig) error {
	configPath, err := GetAuthConfigPath()
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

// SaveSSOConfig saves the SSO configuration to disk (legacy - now wraps SaveAuthConfig)
func SaveSSOConfig(config *SSOConfig) error {
	authConfig := &AuthConfig{
		Method:      AuthMethodSSO,
		SSOStartURL: config.StartURL,
		SSORegion:   config.Region,
	}
	return SaveAuthConfig(authConfig)
}

// ValidateSSOStartURL performs basic validation on the SSO start URL
func ValidateSSOStartURL(url string) error {
	if url == "" {
		return fmt.Errorf("SSO start URL cannot be empty")
	}

	// Basic validation - should start with https://
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("SSO start URL must start with https://")
	}

	// Should be reasonable length
	if len(url) < 10 {
		return fmt.Errorf("SSO start URL is too short")
	}

	return nil
}

// ValidateProfileName performs basic validation on profile name
func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	// AWS profile names can't contain certain characters
	for _, c := range name {
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' || c == '"' || c == '<' || c == '>' || c == '|' {
			return fmt.Errorf("profile name contains invalid character: %c", c)
		}
	}
	return nil
}

// CheckEnvVarsAvailable checks if AWS environment variables are set
func CheckEnvVarsAvailable() bool {
	// Check for standard AWS environment variables
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	return accessKey != "" && secretKey != ""
}
