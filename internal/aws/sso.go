package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/skratchdot/open-golang/open"
)

const (
	// Default SSO configuration
	DefaultSSORegion = "us-east-1"
	ClientName       = "lazyaws"
	ClientType       = "public"
)

// SSOSession represents an active SSO session with cached tokens
type SSOSession struct {
	StartURL              string    `json:"start_url"`
	Region                string    `json:"region"`
	AccessToken           string    `json:"access_token"`
	ExpiresAt             time.Time `json:"expires_at"`
	ClientID              string    `json:"client_id"`
	ClientSecret          string    `json:"client_secret"`
	RegistrationExpiresAt time.Time `json:"registration_expires_at"`
}

// SSOAccount represents an AWS account available through SSO
type SSOAccount struct {
	AccountID    string
	AccountName  string
	EmailAddress string
	RoleName     string
}

// SSOAuthenticator handles AWS SSO authentication
type SSOAuthenticator struct {
	startURL string
	region   string
	session  *SSOSession
}

// NewSSOAuthenticator creates a new SSO authenticator
func NewSSOAuthenticator(startURL, region string) *SSOAuthenticator {
	if region == "" {
		region = DefaultSSORegion
	}

	return &SSOAuthenticator{
		startURL: startURL,
		region:   region,
	}
}

// Authenticate performs the SSO device authorization flow
func (a *SSOAuthenticator) Authenticate(ctx context.Context) error {
	// Try to load cached session first
	if err := a.loadCachedSession(); err == nil && a.session.AccessToken != "" && time.Now().Before(a.session.ExpiresAt) {
		return nil // Valid cached session
	}

	// Create OIDC client for device authorization
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(a.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	oidcClient := ssooidc.NewFromConfig(cfg)

	// Register client if needed or if registration expired
	if a.session == nil || time.Now().After(a.session.RegistrationExpiresAt) {
		if err := a.registerClient(ctx, oidcClient); err != nil {
			return fmt.Errorf("failed to register SSO client: %w", err)
		}
	}

	// Start device authorization
	deviceAuthResp, err := oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     &a.session.ClientID,
		ClientSecret: &a.session.ClientSecret,
		StartUrl:     &a.startURL,
	})
	if err != nil {
		return fmt.Errorf("failed to start device authorization: %w", err)
	}

	// Open browser for user authentication
	verificationURL := *deviceAuthResp.VerificationUriComplete
	if err := openBrowser(verificationURL); err != nil {
		// If browser fails to open, provide the URL for manual navigation
		fmt.Fprintf(os.Stderr, "\nUnable to open browser automatically.\n")
		fmt.Fprintf(os.Stderr, "Please navigate to the following URL to complete authentication:\n\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", verificationURL)
		fmt.Fprintf(os.Stderr, "Waiting for authentication to complete...\n\n")
	}

	// Poll for token
	deviceCode := *deviceAuthResp.DeviceCode
	interval := time.Duration(deviceAuthResp.Interval) * time.Second
	expiresAt := time.Now().Add(time.Duration(deviceAuthResp.ExpiresIn) * time.Second)

	for time.Now().Before(expiresAt) {
		time.Sleep(interval)

	
tokenResp, err := oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     &a.session.ClientID,
			ClientSecret: &a.session.ClientSecret,
			GrantType:    strPtr("urn:ietf:params:oauth:grant-type:device_code"),
			DeviceCode:   &deviceCode,
		})

		if err != nil {
			// Check if authorization is pending
			if isAuthPending(err) {
				continue
			}
			return fmt.Errorf("failed to create token: %w", err)
		}

		// Success! Store the token
		a.session.AccessToken = *tokenResp.AccessToken
		a.session.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

		// Cache the session
		if err := a.saveCachedSession(); err != nil {
			// Non-fatal, just log
			fmt.Fprintf(os.Stderr, "Warning: failed to cache SSO session: %v\n", err)
		}

		return nil
	}

	return fmt.Errorf("SSO authorization timed out")
}

// ListAccounts returns all AWS accounts available through SSO
func (a *SSOAuthenticator) ListAccounts(ctx context.Context) ([]SSOAccount, error) {
	if a.session == nil || a.session.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated - call Authenticate first")
	}

	// Check if token is expired
	if time.Now().After(a.session.ExpiresAt) {
		return nil, fmt.Errorf("SSO session expired - please re-authenticate")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(a.region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)

	// List accounts
	accountsResp, err := ssoClient.ListAccounts(ctx, &sso.ListAccountsInput{
		AccessToken: &a.session.AccessToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var accounts []SSOAccount
	for _, acc := range accountsResp.AccountList {
		// List roles for each account
		rolesResp, err := ssoClient.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
			AccessToken: &a.session.AccessToken,
			AccountId:   acc.AccountId,
		})
		if err != nil {
			continue // Skip accounts we can't access
		}

		// Add each role as a separate account entry (user may have multiple roles per account)
		for _, role := range rolesResp.RoleList {
			accounts = append(accounts, SSOAccount{
				AccountID:    *acc.AccountId,
				AccountName:  *acc.AccountName,
				EmailAddress: *acc.EmailAddress,
				RoleName:     *role.RoleName,
			})
		}
	}

	return accounts, nil
}

// SSOCredentials represents credentials from SSO
type SSOCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// GetCredentials retrieves temporary credentials for a specific account/role
func (a *SSOAuthenticator) GetCredentials(ctx context.Context, accountID, roleName string) (*SSOCredentials, error) {
	if a.session == nil || a.session.AccessToken == "" {
		return nil, fmt.Errorf("not authenticated - call Authenticate first")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(a.region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ssoClient := sso.NewFromConfig(cfg)

	// Get role credentials
	credResp, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
		AccessToken: &a.session.AccessToken,
		AccountId:   &accountID,
		RoleName:    &roleName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role credentials: %w", err)
	}

	creds := credResp.RoleCredentials
	expiresAt := time.UnixMilli(creds.Expiration)

	return &SSOCredentials{
		AccessKeyID:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expiration:      expiresAt,
	},
}

// registerClient registers the application as an SSO OIDC client
func (a *SSOAuthenticator) registerClient(ctx context.Context, oidcClient *ssooidc.Client) error {
	registerResp, err := oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: strPtr(ClientName),
		ClientType: strPtr(ClientType),
	})
	if err != nil {
		return err
	}

	// Initialize session if needed
	if a.session == nil {
		a.session = &SSOSession{
			StartURL: a.startURL,
			Region:   a.region,
		}
	}

	a.session.ClientID = *registerResp.ClientId
	a.session.ClientSecret = *registerResp.ClientSecret
	a.session.RegistrationExpiresAt = time.Unix(registerResp.ClientSecretExpiresAt, 0)

	return nil
}

// getCacheFilePath returns the path to the SSO session cache file
func (a *SSOAuthenticator) getCacheFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(home, ".lazyaws", "sso-cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", err
	}

	// Use a hash of the start URL as filename
	filename := fmt.Sprintf("session-%s.json", hashString(a.startURL))
	return filepath.Join(cacheDir, filename), nil
}

// loadCachedSession loads a cached SSO session if available
func (a *SSOAuthenticator) loadCachedSession() error {
	cachePath, err := a.getCacheFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return err
	}

	var session SSOSession
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	a.session = &session
	return nil
}

// saveCachedSession saves the current SSO session to cache
func (a *SSOAuthenticator) saveCachedSession() error {
	if a.session == nil {
		return fmt.Errorf("no session to cache")
	}

	cachePath, err := a.getCacheFilePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(a.session)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) error {
	return open.Run(url)
}

// isAuthPending checks if the error indicates authorization is still pending
func isAuthPending(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "AuthorizationPendingException") ||
		strings.Contains(errMsg, "authorization_pending")
}

// hashString creates a simple hash of a string for cache filenames
func hashString(s string) string {
	// Simple hash for filename (not cryptographic)
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("%08x", hash)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}