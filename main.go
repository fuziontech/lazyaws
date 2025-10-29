package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fuziontech/lazyaws/internal/aws"
	"github.com/fuziontech/lazyaws/internal/config"
	"github.com/fuziontech/lazyaws/internal/vim"
)

type screen int

const (
	authMethodScreen screen = iota
	authProfileScreen
	ssoConfigScreen
	accountScreen
	ec2Screen
	ec2DetailsScreen
	s3Screen
	s3BrowseScreen
	s3ObjectDetailsScreen
	eksScreen
	eksDetailsScreen
)

// Auth method indices
const (
	authMethodEnvVars = 0
	authMethodProfile = 1
	authMethodSSO     = 2
	maxAuthMethod     = authMethodSSO
)

type model struct {
	currentScreen           screen
	width                   int
	height                  int
	awsClient               *aws.Client
	ec2Instances            []aws.Instance
	ec2FilteredInstances    []aws.Instance // VIM-filtered view
	ec2SelectedIndex        int
	ec2SelectedInstances    map[string]bool // Multi-select support
	ec2InstanceDetails      *aws.InstanceDetails
	ec2InstanceStatus       *aws.InstanceStatus
	ec2InstanceMetrics      *aws.InstanceMetrics
	ec2SSMStatus            *aws.SSMConnectionStatus
	s3Buckets               []aws.Bucket
	s3FilteredBuckets       []aws.Bucket // VIM-filtered view
	s3SelectedIndex         int
	s3CurrentBucket         string
	s3CurrentPrefix         string
	s3Objects               []aws.S3Object
	s3FilteredObjects       []aws.S3Object // VIM-filtered view
	s3ObjectSelectedIndex   int
	s3NextContinuationToken *string
	s3IsTruncated           bool
	s3ObjectDetails         *aws.S3ObjectDetails
	s3Filter                string
	s3FilterActive          bool
	s3PresignedURL          string
	s3BucketPolicy          string
	s3BucketVersioning      string
	s3ShowingInfo           bool   // For showing bucket policy/versioning
	s3InfoType              string // "policy" or "versioning"
	s3ConfirmDelete         bool
	s3DeleteTarget          string // "object" or "bucket"
	s3DeleteKey             string
	deleteConfirmInput      textinput.Model // For typing confirmation
	eksClusters             []aws.EKSCluster
	eksFilteredClusters     []aws.EKSCluster // VIM-filtered view
	eksSelectedIndex        int
	eksClusterDetails       *aws.EKSClusterDetails
	eksNodeGroups           []aws.EKSNodeGroup
	eksAddons               []aws.EKSAddon
	loading                 bool
	err                     error
	config                  *config.Config
	filterInput             textinput.Model
	ssoURLInput             textinput.Model
	profileInput            textinput.Model
	filtering               bool
	configuringSSO          bool
	configuringProfile      bool
	selectedAuthMethod      int // For auth method selection screen
	filter                  string
	confirmAction           string
	confirmInstanceID       string
	showingConfirm          bool
	statusMessage           string
	autoRefresh             bool
	autoRefreshInterval     int // in seconds
	copyToClipboard         string
	vimState                *vim.State
	pageSize                int      // For VIM page navigation
	viewportOffset          int      // Scroll offset for current view
	commandSuggestions      []string // Command suggestions for tab completion
	ssmInstanceID           string   // Store instance ID for SSM session launch
	ssmRegion               string   // Store region for SSM session launch
	ssoAuthenticator        *aws.SSOAuthenticator
	ssoAccounts             []aws.SSOAccount
	ssoFilteredAccounts     []aws.SSOAccount
	ssoSelectedIndex        int
	authConfig              *aws.AuthConfig
	currentAccountID        string
	currentAccountName      string
}

type instancesLoadedMsg struct {
	instances []aws.Instance
	err       error
}

type instanceDetailsLoadedMsg struct {
	details *aws.InstanceDetails
	err     error
}

type instanceStatusLoadedMsg struct {
	status *aws.InstanceStatus
	err    error
}

type instanceMetricsLoadedMsg struct {
	metrics *aws.InstanceMetrics
	err     error
}

type ssmStatusLoadedMsg struct {
	status *aws.SSMConnectionStatus
	err    error
}

type instanceActionCompletedMsg struct {
	action string
	err    error
}

type bucketsLoadedMsg struct {
	buckets []aws.Bucket
	err     error
}

type objectsLoadedMsg struct {
	result *aws.S3ListResult
	err    error
}

type objectDetailsLoadedMsg struct {
	details *aws.S3ObjectDetails
	err     error
}

type fileOperationCompletedMsg struct {
	operation string
	err       error
}

type tickMsg struct{}

type bulkActionCompletedMsg struct {
	action       string
	successCount int
	failureCount int
}

type s3ActionCompletedMsg struct {
	action string
	err    error
}

type presignedURLGeneratedMsg struct {
	url string
	err error
}

type bucketPolicyLoadedMsg struct {
	policy string
	err    error
}

type bucketVersioningLoadedMsg struct {
	versioning string
	err        error
}

type launchSSMSessionMsg struct {
	instanceID string
	region     string
}

type eksClustersLoadedMsg struct {
	clusters []aws.EKSCluster
	err      error
}

type eksClusterDetailsLoadedMsg struct {
	details    *aws.EKSClusterDetails
	nodeGroups []aws.EKSNodeGroup
	addons     []aws.EKSAddon
	err        error
}

type ssoAuthCompletedMsg struct {
	authenticator *aws.SSOAuthenticator
	err           error
}

type ssoAccountsLoadedMsg struct {
	accounts []aws.SSOAccount
	err      error
}

type accountSwitchedMsg struct {
	client      *aws.Client
	accountID   string
	accountName string
	err         error
}

type kubeconfigUpdatedMsg struct {
	clusterName string
	err         error
}

type ssoConfigSavedMsg struct {
	config *aws.SSOConfig
	err    error
}

func initialModel(cfg *config.Config) model {
	// Filter input
	ti := textinput.New()
	ti.Placeholder = "<name>, <id>, state=<state> or tag:key=value"
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20

	// SSO URL input
	ssoInput := textinput.New()
	ssoInput.Placeholder = "https://d-xxxxxxxxxx.awsapps.com/start"
	ssoInput.Focus()
	ssoInput.CharLimit = 256
	ssoInput.Width = 80

	// Profile name input
	profileInput := textinput.New()
	profileInput.Placeholder = "default"
	profileInput.Focus()
	profileInput.CharLimit = 64
	profileInput.Width = 40

	// Delete confirmation input
	deleteInput := textinput.New()
	deleteInput.Placeholder = "Type name to confirm"
	deleteInput.CharLimit = 256
	deleteInput.Width = 80

	// Load auth config if available
	authConfig, _ := aws.LoadAuthConfig()

	// Determine starting screen
	startScreen := ec2Screen
	if authConfig == nil {
		startScreen = authMethodScreen
	}

	return model{
		currentScreen:        startScreen,
		loading:              authConfig != nil, // Only load if we have config
		config:               cfg,
		filterInput:          ti,
		ssoURLInput:          ssoInput,
		profileInput:         profileInput,
		deleteConfirmInput:   deleteInput,
		authConfig:           authConfig,
		configuringSSO:       false,
		configuringProfile:   false,
		selectedAuthMethod:   0,
		filtering:            false,
		ec2SelectedInstances: make(map[string]bool),
		autoRefresh:          false,
		autoRefreshInterval:  30, // Default 30 seconds
		vimState:             vim.NewState(),
		pageSize:             20, // Default page size for ctrl+d/ctrl+u
	}
}

func (m model) Init() tea.Cmd {
	// If auth config doesn't exist, stay on auth method selection screen
	if m.authConfig == nil {
		return nil
	}
	return m.initAWSClient
}

func (m model) initAWSClient() tea.Msg {
	ctx := context.Background()
	var client *aws.Client
	var err error

	if m.authConfig == nil {
		// No auth config, use default (env vars or ~/.aws/config)
		client, err = aws.NewClient(ctx, m.config)
	} else {
		switch m.authConfig.Method {
		case aws.AuthMethodEnv:
			// Use environment variables (AWS SDK handles this automatically)
			client, err = aws.NewClient(ctx, m.config)
		case aws.AuthMethodProfile:
			// Use specific AWS profile
			client, err = aws.NewClientWithProfile(ctx, m.authConfig.ProfileName)
			if client != nil {
				// Override region from profile if configured
				client.Region = m.config.Region
			}
		case aws.AuthMethodSSO:
			// SSO will be handled via :account command
			client, err = aws.NewClient(ctx, m.config)
		default:
			client, err = aws.NewClient(ctx, m.config)
		}
	}

	if err != nil {
		return instancesLoadedMsg{err: err}
	}
	return client
}

// SSO authentication and account switching functions
func (m model) authenticateSSO(startURL, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		authenticator := aws.NewSSOAuthenticator(startURL, region)
		err := authenticator.Authenticate(ctx)
		return ssoAuthCompletedMsg{authenticator: authenticator, err: err}
	}
}

func (m model) loadSSOAccounts() tea.Cmd {
	return func() tea.Msg {
		if m.ssoAuthenticator == nil {
			return ssoAccountsLoadedMsg{err: fmt.Errorf("SSO not authenticated")}
		}
		ctx := context.Background()
		accounts, err := m.ssoAuthenticator.ListAccounts(ctx)
		return ssoAccountsLoadedMsg{accounts: accounts, err: err}
	}
}

func (m model) switchToSSOAccount(account aws.SSOAccount, region string) tea.Cmd {
	return func() tea.Msg {
		if m.ssoAuthenticator == nil {
			return accountSwitchedMsg{err: fmt.Errorf("SSO not authenticated")}
		}
		ctx := context.Background()

		// Get credentials for the account/role
		creds, err := m.ssoAuthenticator.GetCredentials(ctx, account.AccountID, account.RoleName)
		if err != nil {
			return accountSwitchedMsg{err: fmt.Errorf("failed to get credentials: %w", err)}
		}

		// Create new AWS client with SSO credentials
		client, err := aws.NewClientWithSSOCredentials(ctx, creds, region, account.AccountName)
		if err != nil {
			return accountSwitchedMsg{err: fmt.Errorf("failed to create client: %w", err)}
		}

		return accountSwitchedMsg{
			client:      client,
			accountID:   account.AccountID,
			accountName: account.AccountName,
		}
	}
}

func (m model) loadEC2Instances() tea.Msg {
	ctx := context.Background()
	instances, err := m.awsClient.ListInstances(ctx)
	return instancesLoadedMsg{instances: instances, err: err}
}

func (m model) loadS3Buckets() tea.Msg {
	ctx := context.Background()
	buckets, err := m.awsClient.ListBuckets(ctx)
	return bucketsLoadedMsg{buckets: buckets, err: err}
}

func (m model) loadEKSClusters() tea.Msg {
	ctx := context.Background()
	clusters, err := m.awsClient.ListEKSClusters(ctx)
	return eksClustersLoadedMsg{clusters: clusters, err: err}
}

func (m model) loadEKSClusterDetails(clusterName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Load cluster details, node groups, and addons in parallel
		detailsChan := make(chan *aws.EKSClusterDetails)
		nodeGroupsChan := make(chan []aws.EKSNodeGroup)
		addonsChan := make(chan []aws.EKSAddon)
		errChan := make(chan error, 3)

		// Load cluster details
		go func() {
			details, err := m.awsClient.GetEKSClusterDetails(ctx, clusterName)
			if err != nil {
				errChan <- err
				detailsChan <- nil
				return
			}
			detailsChan <- details
			errChan <- nil
		}()

		// Load node groups
		go func() {
			nodeGroups, err := m.awsClient.ListNodeGroups(ctx, clusterName)
			if err != nil {
				errChan <- err
				nodeGroupsChan <- nil
				return
			}
			nodeGroupsChan <- nodeGroups
			errChan <- nil
		}()

		// Load addons
		go func() {
			addons, err := m.awsClient.ListAddons(ctx, clusterName)
			if err != nil {
				errChan <- err
				addonsChan <- nil
				return
			}
			addonsChan <- addons
			errChan <- nil
		}()

		// Wait for all to complete
		details := <-detailsChan
		nodeGroups := <-nodeGroupsChan
		addons := <-addonsChan

		// Check for errors
		var firstErr error
		for i := 0; i < 3; i++ {
			if err := <-errChan; err != nil && firstErr == nil {
				firstErr = err
			}
		}

		return eksClusterDetailsLoadedMsg{
			details:    details,
			nodeGroups: nodeGroups,
			addons:     addons,
			err:        firstErr,
		}
	}
}

func (m model) updateKubeconfig(clusterName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.UpdateKubeconfig(ctx, clusterName)
		return kubeconfigUpdatedMsg{clusterName: clusterName, err: err}
	}
}

func (m model) loadS3Objects(bucket, prefix string, continuationToken *string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := m.awsClient.ListObjects(ctx, bucket, prefix, continuationToken)
		return objectsLoadedMsg{result: result, err: err}
	}
}

func (m model) loadS3ObjectDetails(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		details, err := m.awsClient.GetObjectDetails(ctx, bucket, key)
		return objectDetailsLoadedMsg{details: details, err: err}
	}
}

func (m model) downloadS3Object(bucket, key, localPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.DownloadObject(ctx, bucket, key, localPath)
		return fileOperationCompletedMsg{operation: "download", err: err}
	}
}

func (m model) uploadS3Object(bucket, key, localPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.UploadObject(ctx, bucket, key, localPath)
		return fileOperationCompletedMsg{operation: "upload", err: err}
	}
}

func (m model) deleteS3Object(bucket, key string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.DeleteObject(ctx, bucket, key)
		return s3ActionCompletedMsg{action: "delete object", err: err}
	}
}

func (m model) deleteS3Bucket(bucket string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.DeleteBucket(ctx, bucket)
		return s3ActionCompletedMsg{action: "delete bucket", err: err}
	}
}

func (m model) createS3Bucket(bucket, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.CreateBucket(ctx, bucket, region)
		return s3ActionCompletedMsg{action: "create bucket", err: err}
	}
}

func (m model) copyS3Object(sourceBucket, sourceKey, destBucket, destKey string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.awsClient.CopyObject(ctx, sourceBucket, sourceKey, destBucket, destKey)
		return s3ActionCompletedMsg{action: "copy object", err: err}
	}
}

func (m model) generatePresignedURL(bucket, key string, expiration int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		url, err := m.awsClient.GeneratePresignedURL(ctx, bucket, key, expiration)
		return presignedURLGeneratedMsg{url: url, err: err}
	}
}

func (m model) loadBucketPolicy(bucket string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		policy, err := m.awsClient.GetBucketPolicy(ctx, bucket)
		return bucketPolicyLoadedMsg{policy: policy, err: err}
	}
}

func (m model) loadBucketVersioning(bucket string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		versioning, err := m.awsClient.GetBucketVersioning(ctx, bucket)
		return bucketVersioningLoadedMsg{versioning: versioning, err: err}
	}
}

func (m model) loadEC2InstanceDetails(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		details, err := m.awsClient.GetInstanceDetails(ctx, instanceID)
		return instanceDetailsLoadedMsg{details: details, err: err}
	}
}

func (m model) loadInstanceStatus(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		status, err := m.awsClient.GetInstanceStatus(ctx, instanceID)
		return instanceStatusLoadedMsg{status: status, err: err}
	}
}

func (m model) loadInstanceMetrics(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		metrics, err := m.awsClient.GetInstanceMetrics(ctx, instanceID)
		return instanceMetricsLoadedMsg{metrics: metrics, err: err}
	}
}

func (m model) loadSSMStatus(instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		status, err := m.awsClient.CheckSSMConnectivity(ctx, instanceID)
		return ssmStatusLoadedMsg{status: status, err: err}
	}
}

func (m model) performInstanceAction(action string, instanceID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error

		switch action {
		case "start":
			err = m.awsClient.StartInstance(ctx, instanceID)
		case "stop":
			err = m.awsClient.StopInstance(ctx, instanceID)
		case "reboot":
			err = m.awsClient.RebootInstance(ctx, instanceID)
		case "terminate":
			err = m.awsClient.TerminateInstance(ctx, instanceID)
		}

		return instanceActionCompletedMsg{action: action, err: err}
	}
}

func (m model) performBulkAction(action string, instanceIDs []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		successCount := 0
		failureCount := 0

		for _, instanceID := range instanceIDs {
			var err error
			switch action {
			case "start":
				err = m.awsClient.StartInstance(ctx, instanceID)
			case "stop":
				err = m.awsClient.StopInstance(ctx, instanceID)
			case "reboot":
				err = m.awsClient.RebootInstance(ctx, instanceID)
			case "terminate":
				err = m.awsClient.TerminateInstance(ctx, instanceID)
			}

			if err != nil {
				failureCount++
			} else {
				successCount++
			}
		}

		return bulkActionCompletedMsg{
			action:       action,
			successCount: successCount,
			failureCount: failureCount,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle auth method selection screen
	if m.currentScreen == authMethodScreen {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.selectedAuthMethod > 0 {
					m.selectedAuthMethod--
				}
			case "down", "j":
				if m.selectedAuthMethod < maxAuthMethod {
					m.selectedAuthMethod++
				}
			case "enter":
				// Save selected auth method
				switch m.selectedAuthMethod {
				case 0: // Environment Variables
					authConfig := &aws.AuthConfig{
						Method: aws.AuthMethodEnv,
					}
					if err := aws.SaveAuthConfig(authConfig); err != nil {
						m.statusMessage = fmt.Sprintf("Failed to save config: %v", err)
						return m, nil
					}
					m.authConfig = authConfig
					m.currentScreen = ec2Screen
					m.loading = true
					return m, m.initAWSClient
				case 1: // AWS Profile
					m.currentScreen = authProfileScreen
					m.configuringProfile = true
					return m, nil
				case 2: // SSO
					m.currentScreen = ssoConfigScreen
					m.configuringSSO = true
					return m, nil
				}
			case "esc":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	// Handle AWS Profile configuration
	if m.configuringProfile && m.currentScreen == authProfileScreen {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				profileName := m.profileInput.Value()
				if profileName == "" {
					profileName = "default"
				}
				if err := aws.ValidateProfileName(profileName); err != nil {
					m.statusMessage = fmt.Sprintf("Invalid profile name: %v", err)
					return m, nil
				}
				// Save profile config
				authConfig := &aws.AuthConfig{
					Method:      aws.AuthMethodProfile,
					ProfileName: profileName,
				}
				if err := aws.SaveAuthConfig(authConfig); err != nil {
					m.statusMessage = fmt.Sprintf("Failed to save config: %v", err)
					return m, nil
				}
				m.authConfig = authConfig
				m.configuringProfile = false
				m.currentScreen = ec2Screen
				m.loading = true
				m.statusMessage = fmt.Sprintf("Using AWS profile: %s", profileName)
				return m, m.initAWSClient
			case "esc":
				// Go back to auth method selection
				m.currentScreen = authMethodScreen
				m.configuringProfile = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.profileInput, cmd = m.profileInput.Update(msg)
		return m, cmd
	}

	// Handle SSO URL configuration
	if m.configuringSSO && m.currentScreen == ssoConfigScreen {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				ssoURL := m.ssoURLInput.Value()
				if err := aws.ValidateSSOStartURL(ssoURL); err != nil {
					m.statusMessage = fmt.Sprintf("Invalid SSO URL: %v", err)
					return m, nil
				}
				// Save SSO config
				authConfig := &aws.AuthConfig{
					Method:      aws.AuthMethodSSO,
					SSOStartURL: ssoURL,
					SSORegion:   aws.DefaultSSORegion,
				}
				if err := aws.SaveAuthConfig(authConfig); err != nil {
					m.statusMessage = fmt.Sprintf("Failed to save config: %v", err)
					return m, nil
				}
				m.authConfig = authConfig
				m.configuringSSO = false
				m.currentScreen = ec2Screen
				m.loading = true
				m.statusMessage = "SSO configuration saved"
				return m, m.initAWSClient
			case "esc":
				// Go back to auth method selection
				m.currentScreen = authMethodScreen
				m.configuringSSO = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.ssoURLInput, cmd = m.ssoURLInput.Update(msg)
		return m, cmd
	}

	// Handle S3 delete confirmation dialog
	if m.s3ConfirmDelete {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Check if typed value matches the item to delete
				if m.deleteConfirmInput.Value() == m.s3DeleteKey {
					m.loading = true
					m.s3ConfirmDelete = false
					m.deleteConfirmInput.SetValue("")
					if m.s3DeleteTarget == "object" {
						return m, m.deleteS3Object(m.s3CurrentBucket, m.s3DeleteKey)
					} else if m.s3DeleteTarget == "bucket" {
						return m, m.deleteS3Bucket(m.s3DeleteKey)
					}
				} else {
					m.statusMessage = "Name doesn't match - delete cancelled"
					m.s3ConfirmDelete = false
					m.s3DeleteTarget = ""
					m.s3DeleteKey = ""
					m.deleteConfirmInput.SetValue("")
					return m, nil
				}
			case "esc":
				m.s3ConfirmDelete = false
				m.s3DeleteTarget = ""
				m.s3DeleteKey = ""
				m.deleteConfirmInput.SetValue("")
				m.statusMessage = "Delete cancelled"
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.deleteConfirmInput, cmd = m.deleteConfirmInput.Update(msg)
		return m, cmd
	}

	// Handle EC2 confirmation dialog
	if m.showingConfirm {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				m.loading = true
				// Check if this is a bulk action
				if len(m.ec2SelectedInstances) > 0 && m.currentScreen == ec2Screen {
					var instanceIDs []string
					for id := range m.ec2SelectedInstances {
						instanceIDs = append(instanceIDs, id)
					}
					return m, m.performBulkAction(m.confirmAction, instanceIDs)
				}
				return m, m.performInstanceAction(m.confirmAction, m.confirmInstanceID)
			case "n", "N", "esc":
				m.showingConfirm = false
				m.confirmAction = ""
				m.confirmInstanceID = ""
				return m, nil
			}
		}
		return m, nil
	}

	// Handle VIM modes (search/command)
	if m.vimState.Mode == vim.SearchMode || m.vimState.Mode == vim.CommandMode {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Handle tab completion in command mode
			if m.vimState.Mode == vim.CommandMode && msg.String() == "tab" {
				completed, isComplete := vim.CompleteCommand(m.vimState.CommandBuffer)
				m.vimState.CommandBuffer = completed
				if !isComplete {
					// Show suggestions
					m.commandSuggestions = vim.GetCommandSuggestions(completed)
				} else {
					m.commandSuggestions = nil
				}
				return m, nil
			}

			if m.vimState.HandleKey(msg) {
				// Update suggestions as user types in command mode
				if m.vimState.Mode == vim.CommandMode {
					m.commandSuggestions = vim.GetCommandSuggestions(m.vimState.CommandBuffer)
				}

				// Apply search on every keypress in search mode (instant search)
				if m.vimState.Mode == vim.SearchMode {
					if m.vimState.SearchQuery != "" {
						// Update the search query and apply immediately
						m.vimState.LastSearch = m.vimState.SearchQuery
						m.applyVimSearch()
					} else {
						// Empty search query - clear the search
						m.vimState.LastSearch = ""
						m.vimState.SearchResults = []int{}
						m.ec2FilteredInstances = nil
						m.s3FilteredBuckets = nil
						m.s3FilteredObjects = nil
					}
				}

				// If search mode was just completed, apply the search
				if m.vimState.Mode == vim.NormalMode && m.vimState.LastSearch != "" {
					m.applyVimSearch()
				}
				// If command mode was just completed, execute the command
				if m.vimState.Mode == vim.NormalMode && m.vimState.CommandBuffer != "" {
					cmd := m.executeVimCommand(m.vimState.CommandBuffer)
					m.vimState.CommandBuffer = ""
					m.commandSuggestions = nil
					return m, cmd
				}
				return m, nil
			}
		}
		return m, nil
	}

	// Handle filtering (legacy filter mode)
	if m.filtering {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.filter = m.filterInput.Value()
				m.filtering = false
				return m, nil
			case "esc":
				m.filtering = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case *aws.Client:
		m.awsClient = msg
		// Clear stale data when switching accounts/regions
		m.ec2Instances = nil
		m.s3Buckets = nil
		m.eksClusters = nil
		m.clearSearch()
		if m.autoRefresh {
			return m, tea.Batch(m.loadEC2Instances, tickCmd())
		}
		return m, m.loadEC2Instances

	case tickMsg:
		// Auto-refresh EC2 instances if enabled and on EC2 screen
		if m.autoRefresh && m.currentScreen == ec2Screen {
			return m, tea.Batch(m.loadEC2Instances, tickCmd())
		}
		return m, tickCmd()

	case bulkActionCompletedMsg:
		m.loading = false
		m.showingConfirm = false
		if msg.failureCount > 0 {
			m.statusMessage = fmt.Sprintf("Bulk %s: %d succeeded, %d failed", msg.action, msg.successCount, msg.failureCount)
		} else {
			m.statusMessage = fmt.Sprintf("Bulk %s: %d instances succeeded", msg.action, msg.successCount)
		}
		// Clear selections
		m.ec2SelectedInstances = make(map[string]bool)
		// Refresh instances list
		return m, m.loadEC2Instances

	case instancesLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.ec2Instances = msg.instances
			m.ec2SelectedIndex = 0 // Reset selection
		}
		return m, nil

	case instanceDetailsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.ec2InstanceDetails = msg.details
			m.currentScreen = ec2DetailsScreen
			// Load additional information for details view
			instanceID := msg.details.ID
			return m, tea.Batch(
				m.loadInstanceStatus(instanceID),
				m.loadInstanceMetrics(instanceID),
				m.loadSSMStatus(instanceID),
			)
		}
		return m, nil

	case instanceStatusLoadedMsg:
		if msg.err == nil {
			m.ec2InstanceStatus = msg.status
		}
		return m, nil

	case instanceMetricsLoadedMsg:
		if msg.err == nil {
			m.ec2InstanceMetrics = msg.metrics
		}
		return m, nil

	case ssmStatusLoadedMsg:
		if msg.err == nil {
			m.ec2SSMStatus = msg.status
		}
		return m, nil

	case instanceActionCompletedMsg:
		m.loading = false
		m.showingConfirm = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Successfully %sed instance", msg.action)
		}
		// Refresh instances list
		return m, m.loadEC2Instances

	case bucketsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.s3Buckets = msg.buckets
			m.s3SelectedIndex = 0
		}
		return m, nil

	case eksClustersLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.eksClusters = msg.clusters
			m.eksSelectedIndex = 0
		}
		return m, nil

	case eksClusterDetailsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.eksClusterDetails = msg.details
			m.eksNodeGroups = msg.nodeGroups
			m.eksAddons = msg.addons
			m.currentScreen = eksDetailsScreen
			m.viewportOffset = 0
		}
		return m, nil

	case kubeconfigUpdatedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error updating kubeconfig: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Updated kubeconfig for cluster: %s", msg.clusterName)
		}
		return m, nil

	case ssoAuthCompletedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("SSO authentication failed: %v", msg.err)
			m.err = msg.err
			return m, nil
		}
		m.ssoAuthenticator = msg.authenticator
		m.statusMessage = "SSO authentication successful - loading accounts..."
		// Load available accounts
		return m, m.loadSSOAccounts()

	case ssoAccountsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to load accounts: %v", msg.err)
			m.err = msg.err
			return m, nil
		}
		m.ssoAccounts = msg.accounts
		m.ssoSelectedIndex = 0
		m.currentScreen = accountScreen
		m.statusMessage = fmt.Sprintf("Loaded %d accounts - select one to continue", len(msg.accounts))
		return m, nil

	case accountSwitchedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to switch account: %v", msg.err)
			m.err = msg.err
			return m, nil
		}
		// Update client and account info
		m.awsClient = msg.client
		m.currentAccountID = msg.accountID
		m.currentAccountName = msg.accountName
		// Clear stale data when switching accounts
		m.ec2Instances = nil
		m.s3Buckets = nil
		m.eksClusters = nil
		m.clearSearch()
		// Clear any previous errors
		m.err = nil
		// Switch to EC2 screen and load instances
		m.currentScreen = ec2Screen
		m.viewportOffset = 0
		m.loading = true // Show loading state
		m.statusMessage = fmt.Sprintf("Switched to account: %s", msg.accountName)
		return m, m.loadEC2Instances

	case objectsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.s3Objects = msg.result.Objects
			m.s3NextContinuationToken = msg.result.NextContinuationToken
			m.s3IsTruncated = msg.result.IsTruncated
			m.s3ObjectSelectedIndex = 0
			m.currentScreen = s3BrowseScreen
		}
		return m, nil

	case objectDetailsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.s3ObjectDetails = msg.details
			m.currentScreen = s3ObjectDetailsScreen
		}
		return m, nil

	case fileOperationCompletedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error %sing: %v", msg.operation, msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Successfully %sed file", msg.operation)
		}
		return m, nil

	case s3ActionCompletedMsg:
		m.loading = false
		m.s3ConfirmDelete = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Successfully completed: %s", msg.action)
		}
		// Refresh appropriate view
		if m.s3DeleteTarget == "bucket" {
			return m, m.loadS3Buckets
		} else if m.s3DeleteTarget == "object" {
			return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
		}
		return m, nil

	case presignedURLGeneratedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error generating URL: %v", msg.err)
		} else {
			m.s3PresignedURL = msg.url
			// Copy to clipboard on macOS
			if runtime.GOOS == "darwin" {
				cmd := exec.Command("pbcopy")
				cmd.Stdin = strings.NewReader(msg.url)
				if err := cmd.Run(); err == nil {
					m.statusMessage = "Presigned URL copied to clipboard"
				} else {
					m.statusMessage = "Presigned URL generated (displayed below)"
				}
			} else {
				m.statusMessage = "Presigned URL generated (displayed below)"
			}
		}
		return m, nil

	case bucketPolicyLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.s3BucketPolicy = fmt.Sprintf("Error: %v", msg.err)
		} else if msg.policy == "" {
			m.s3BucketPolicy = "No bucket policy set"
		} else {
			m.s3BucketPolicy = msg.policy
		}
		m.s3ShowingInfo = true
		m.s3InfoType = "policy"
		return m, nil

	case bucketVersioningLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.s3BucketVersioning = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.s3BucketVersioning = msg.versioning
		}
		m.s3ShowingInfo = true
		m.s3InfoType = "versioning"
		return m, nil

	case launchSSMSessionMsg:
		// Store the SSM session info in the model so we can access it after quit
		m.ssmInstanceID = msg.instanceID
		m.ssmRegion = msg.region
		m.statusMessage = fmt.Sprintf("Launching SSM session for %s...", msg.instanceID)
		// Quit the program to launch SSM in current terminal
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Don't quit if we're in details view, go back instead
			if m.currentScreen == ec2DetailsScreen {
				m.currentScreen = ec2Screen
				m.ec2InstanceDetails = nil
				m.viewportOffset = 0
				return m, nil
			} else if m.currentScreen == s3BrowseScreen {
				m.currentScreen = s3Screen
				m.s3Objects = nil
				m.s3CurrentBucket = ""
				m.s3CurrentPrefix = ""
				m.viewportOffset = 0
				return m, nil
			} else if m.currentScreen == s3ObjectDetailsScreen {
				m.currentScreen = s3BrowseScreen
				m.s3ObjectDetails = nil
				m.viewportOffset = 0
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			// ESC key to dismiss S3 info popup or clear presigned URL or clear search or go back from details view
			if m.s3ShowingInfo {
				m.s3ShowingInfo = false
				m.s3InfoType = ""
				m.s3BucketPolicy = ""
				m.s3BucketVersioning = ""
				m.s3PresignedURL = ""
				return m, nil
			}
			// Clear presigned URL if showing
			if m.s3PresignedURL != "" {
				m.s3PresignedURL = ""
				m.statusMessage = "Presigned URL cleared"
				return m, nil
			}
			// Clear active search filter
			if m.vimState.LastSearch != "" {
				m.vimState.LastSearch = ""
				m.vimState.SearchResults = []int{}
				m.ec2FilteredInstances = nil
				m.s3FilteredBuckets = nil
				m.s3FilteredObjects = nil
				m.statusMessage = "Search cleared"
				return m, nil
			}
			if m.currentScreen == ec2DetailsScreen {
				m.currentScreen = ec2Screen
				m.ec2InstanceDetails = nil
				return m, nil
			} else if m.currentScreen == s3BrowseScreen {
				// If we're in a subfolder, go to parent folder
				// Otherwise go back to bucket list
				if m.s3CurrentPrefix != "" {
					// Calculate parent prefix
					// Remove trailing slash if present
					prefix := strings.TrimSuffix(m.s3CurrentPrefix, "/")
					// Find last slash to get parent
					lastSlash := strings.LastIndex(prefix, "/")
					if lastSlash >= 0 {
						// Go to parent folder
						m.s3CurrentPrefix = prefix[:lastSlash+1]
					} else {
						// We're at root level, go to root
						m.s3CurrentPrefix = ""
					}
					m.loading = true
					m.viewportOffset = 0
					return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
				} else {
					// We're at bucket root, go back to bucket list
					m.currentScreen = s3Screen
					m.s3Objects = nil
					m.s3CurrentBucket = ""
					m.s3CurrentPrefix = ""
					return m, nil
				}
			} else if m.currentScreen == s3ObjectDetailsScreen {
				m.currentScreen = s3BrowseScreen
				m.s3ObjectDetails = nil
				return m, nil
			} else if m.currentScreen == eksDetailsScreen {
				m.currentScreen = eksScreen
				m.eksClusterDetails = nil
				m.eksNodeGroups = nil
				m.eksAddons = nil
				return m, nil
			}
		case "k", "up", "j", "down", "g", "G", "ctrl+g", "ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f", "pgup", "pgdown":
			// VIM-style navigation
			action := vim.ParseNavigation(msg)
			m.handleVimNavigation(action)
		case "/":
			// Enter VIM search mode
			m.vimState.EnterSearchMode()
			return m, nil
		case "n":
			// Next search result (VIM-style)
			if m.vimState.LastSearch != "" {
				if idx := m.vimState.NextMatch(); idx >= 0 {
					m.setSelectedIndex(idx)
				}
			} else if m.currentScreen == s3BrowseScreen && m.s3IsTruncated && m.s3NextContinuationToken != nil {
				// Load next page in S3 browser (if no active search)
				m.loading = true
				return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, m.s3NextContinuationToken)
			}
		case "N":
			// Previous search result
			if m.vimState.LastSearch != "" {
				if idx := m.vimState.PrevMatch(); idx >= 0 {
					m.setSelectedIndex(idx)
				}
			}
		case ":":
			// Enter VIM command mode
			m.vimState.EnterCommandMode()
			return m, nil
		case "enter", "i":
			// Enter key to view instance details or browse S3 bucket, or view object details
			if m.currentScreen == ec2Screen {
				// Use filtered list if active
				instances := m.ec2Instances
				if len(m.ec2FilteredInstances) > 0 {
					instances = m.ec2FilteredInstances
				}
				if len(instances) > 0 && m.ec2SelectedIndex < len(instances) {
					selectedInstance := instances[m.ec2SelectedIndex]
					m.loading = true
					m.viewportOffset = 0
					return m, m.loadEC2InstanceDetails(selectedInstance.ID)
				}
			} else if m.currentScreen == s3Screen {
				// Browse bucket contents - use filtered list if active
				buckets := m.s3Buckets
				if len(m.s3FilteredBuckets) > 0 {
					buckets = m.s3FilteredBuckets
				}
				if len(buckets) > 0 && m.s3SelectedIndex < len(buckets) {
					selectedBucket := buckets[m.s3SelectedIndex]
					m.s3CurrentBucket = selectedBucket.Name
					m.s3CurrentPrefix = ""
					// Clear search when entering a bucket
					m.vimState.LastSearch = ""
					m.vimState.SearchResults = []int{}
					m.s3FilteredBuckets = nil
					m.loading = true
					m.viewportOffset = 0
					return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
				}
			} else if m.currentScreen == s3BrowseScreen {
				// Use filtered list if active
				objects := m.s3Objects
				if len(m.s3FilteredObjects) > 0 {
					objects = m.s3FilteredObjects
				}
				if len(objects) > 0 && m.s3ObjectSelectedIndex < len(objects) {
					selectedObject := objects[m.s3ObjectSelectedIndex]
					if selectedObject.IsFolder {
						// Navigate into folder
						m.s3CurrentPrefix = selectedObject.Key
						// Clear search when entering a folder
						m.vimState.LastSearch = ""
						m.vimState.SearchResults = []int{}
						m.s3FilteredObjects = nil
						m.loading = true
						m.viewportOffset = 0
						return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
					} else {
						// View file details
						m.loading = true
						m.viewportOffset = 0
						return m, m.loadS3ObjectDetails(m.s3CurrentBucket, selectedObject.Key)
					}
				}
			} else if m.currentScreen == eksScreen {
				// View EKS cluster details - use filtered list if active
				clusters := m.eksClusters
				if len(m.eksFilteredClusters) > 0 {
					clusters = m.eksFilteredClusters
				}
				if len(clusters) > 0 && m.eksSelectedIndex < len(clusters) {
					selectedCluster := clusters[m.eksSelectedIndex]
					m.loading = true
					m.viewportOffset = 0
					return m, m.loadEKSClusterDetails(selectedCluster.Name)
				}
			} else if m.currentScreen == accountScreen {
				// Switch to selected AWS account
				accounts := m.ssoAccounts
				if len(m.ssoFilteredAccounts) > 0 {
					accounts = m.ssoFilteredAccounts
				}
				if len(accounts) > 0 && m.ssoSelectedIndex < len(accounts) {
					selectedAccount := accounts[m.ssoSelectedIndex]
					// Clear search when switching accounts
					m.clearSearch()
					m.loading = true
					m.viewportOffset = 0
					m.statusMessage = fmt.Sprintf("Switching to account: %s (%s)", selectedAccount.AccountName, selectedAccount.AccountID)
					return m, m.switchToSSOAccount(selectedAccount, m.config.Region)
				}
			}
		case "c":
			// Find the index of the current region
			currentIndex := -1
			for i, r := range m.config.Regions {
				if r == m.config.Region {
					currentIndex = i
					break
				}
			}
			// Cycle to the next region
			if currentIndex != -1 {
				nextIndex := (currentIndex + 1) % len(m.config.Regions)
				m.config.Region = m.config.Regions[nextIndex]
				m.loading = true
				return m, m.initAWSClient
			}

		case "tab":
			// Tab cycles through main screens (not details)
			m.clearSearch() // Clear search when switching screens
			if m.currentScreen == ec2Screen {
				m.currentScreen = s3Screen
			} else if m.currentScreen == s3Screen {
				m.currentScreen = eksScreen
				// Load EKS clusters when switching to EKS screen
				if len(m.eksClusters) == 0 {
					m.loading = true
					return m, m.loadEKSClusters
				}
			} else if m.currentScreen == eksScreen {
				m.currentScreen = ec2Screen
			}
		case "r":
			// Refresh current view
			if m.currentScreen == ec2Screen {
				m.loading = true
				return m, m.loadEC2Instances
			} else if m.currentScreen == s3Screen {
				m.loading = true
				return m, m.loadS3Buckets
			} else if m.currentScreen == s3BrowseScreen {
				m.loading = true
				return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
			} else if m.currentScreen == eksScreen {
				m.loading = true
				return m, m.loadEKSClusters
			}
		case "K":
			// Update kubeconfig for selected EKS cluster
			if m.currentScreen == eksScreen {
				// Use filtered list if active
				clusters := m.eksClusters
				if len(m.eksFilteredClusters) > 0 {
					clusters = m.eksFilteredClusters
				}
				if len(clusters) > 0 && m.eksSelectedIndex < len(clusters) {
					selectedCluster := clusters[m.eksSelectedIndex]
					m.loading = true
					m.statusMessage = fmt.Sprintf("Updating kubeconfig for %s...", selectedCluster.Name)
					return m, m.updateKubeconfig(selectedCluster.Name)
				}
			} else if m.currentScreen == eksDetailsScreen && m.eksClusterDetails != nil {
				// Update kubeconfig from details screen
				m.loading = true
				m.statusMessage = fmt.Sprintf("Updating kubeconfig for %s...", m.eksClusterDetails.Name)
				return m, m.updateKubeconfig(m.eksClusterDetails.Name)
			}
		case "backspace", "h":
			// Go up one level in S3 browser
			if m.currentScreen == s3BrowseScreen {
				if m.s3CurrentPrefix == "" {
					// At root, go back to bucket list
					m.currentScreen = s3Screen
					m.s3Objects = nil
					m.s3CurrentBucket = ""
					m.viewportOffset = 0
					return m, nil
				}
				// Remove last directory from prefix
				parts := strings.Split(strings.TrimSuffix(m.s3CurrentPrefix, "/"), "/")
				if len(parts) > 1 {
					m.s3CurrentPrefix = strings.Join(parts[:len(parts)-1], "/") + "/"
				} else {
					m.s3CurrentPrefix = ""
				}
				m.loading = true
				m.viewportOffset = 0
				return m, m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
			}

		case "d":
			// Download selected S3 object
			if m.currentScreen == s3BrowseScreen {
				// Use filtered list if active
				objects := m.s3Objects
				if len(m.s3FilteredObjects) > 0 {
					objects = m.s3FilteredObjects
				}
				if len(objects) > 0 && m.s3ObjectSelectedIndex < len(objects) {
					selectedObject := objects[m.s3ObjectSelectedIndex]
					if !selectedObject.IsFolder {
						// Extract just the filename from the key
						fileName := selectedObject.Key
						if strings.Contains(fileName, "/") {
							parts := strings.Split(fileName, "/")
							fileName = parts[len(parts)-1]
						}
						m.loading = true
						m.statusMessage = fmt.Sprintf("Downloading %s...", fileName)
						return m, m.downloadS3Object(m.s3CurrentBucket, selectedObject.Key, fileName)
					}
				}
			} else if m.currentScreen == s3ObjectDetailsScreen && m.s3ObjectDetails != nil {
				// Download from object details view
				fileName := m.s3ObjectDetails.Key
				if strings.Contains(fileName, "/") {
					parts := strings.Split(fileName, "/")
					fileName = parts[len(parts)-1]
				}
				m.loading = true
				m.statusMessage = fmt.Sprintf("Downloading %s...", fileName)
				return m, m.downloadS3Object(m.s3CurrentBucket, m.s3ObjectDetails.Key, fileName)
			}
		case "u":
			// Upload file to S3 (prompt for file path)
			// For now, we'll just show a message that upload requires file path
			// In a full implementation, we'd add a text input for the file path
			if m.currentScreen == s3BrowseScreen {
				m.statusMessage = "Upload: Feature requires interactive file picker (coming soon)"
			}
		case "D":
			// Delete S3 object or bucket
			if m.currentScreen == s3BrowseScreen {
				// Use filtered list if active
				objects := m.s3Objects
				if len(m.s3FilteredObjects) > 0 {
					objects = m.s3FilteredObjects
				}
				if len(objects) > 0 && m.s3ObjectSelectedIndex < len(objects) {
					selectedObject := objects[m.s3ObjectSelectedIndex]
					if !selectedObject.IsFolder {
						m.s3ConfirmDelete = true
						m.s3DeleteTarget = "object"
						m.s3DeleteKey = selectedObject.Key
						m.deleteConfirmInput.SetValue("")
						m.deleteConfirmInput.Focus()
						m.statusMessage = fmt.Sprintf("Type the object name to confirm deletion: %s", selectedObject.Key)
						return m, nil
					}
				}
			} else if m.currentScreen == s3Screen {
				// Use filtered list if active
				buckets := m.s3Buckets
				if len(m.s3FilteredBuckets) > 0 {
					buckets = m.s3FilteredBuckets
				}
				if len(buckets) > 0 && m.s3SelectedIndex < len(buckets) {
					selectedBucket := buckets[m.s3SelectedIndex]
					m.s3ConfirmDelete = true
					m.s3DeleteTarget = "bucket"
					m.s3DeleteKey = selectedBucket.Name
					m.deleteConfirmInput.SetValue("")
					m.deleteConfirmInput.Focus()
					m.statusMessage = fmt.Sprintf("Type the bucket name to confirm deletion (bucket must be empty!): %s", selectedBucket.Name)
					return m, nil
				}
			}
		case "p":
			// Generate presigned URL or view bucket policy
			if m.currentScreen == s3BrowseScreen {
				// Use filtered list if active
				objects := m.s3Objects
				if len(m.s3FilteredObjects) > 0 {
					objects = m.s3FilteredObjects
				}
				if len(objects) > 0 && m.s3ObjectSelectedIndex < len(objects) {
					selectedObject := objects[m.s3ObjectSelectedIndex]
					if !selectedObject.IsFolder {
						m.loading = true
						// Generate presigned URL with 1 hour expiration
						return m, m.generatePresignedURL(m.s3CurrentBucket, selectedObject.Key, 3600)
					}
				}
			} else if m.currentScreen == s3Screen {
				// Use filtered list if active
				buckets := m.s3Buckets
				if len(m.s3FilteredBuckets) > 0 {
					buckets = m.s3FilteredBuckets
				}
				if len(buckets) > 0 && m.s3SelectedIndex < len(buckets) {
					selectedBucket := buckets[m.s3SelectedIndex]
					m.loading = true
					return m, m.loadBucketPolicy(selectedBucket.Name)
				}
			} else if m.currentScreen == s3ObjectDetailsScreen && m.s3ObjectDetails != nil {
				m.loading = true
				return m, m.generatePresignedURL(m.s3CurrentBucket, m.s3ObjectDetails.Key, 3600)
			}
		case "v":
			// View bucket versioning
			if m.currentScreen == s3Screen {
				// Use filtered list if active
				buckets := m.s3Buckets
				if len(m.s3FilteredBuckets) > 0 {
					buckets = m.s3FilteredBuckets
				}
				if len(buckets) > 0 && m.s3SelectedIndex < len(buckets) {
					selectedBucket := buckets[m.s3SelectedIndex]
					m.loading = true
					return m, m.loadBucketVersioning(selectedBucket.Name)
				}
			}
		case "f":
			// Only filter on EC2 list screen
			if m.currentScreen == ec2Screen {
				m.filtering = true
				m.filterInput.Focus()
				return m, nil
			}
		case " ":
			// Toggle instance selection (space bar)
			if m.currentScreen == ec2Screen {
				// Use filtered list if active
				instances := m.ec2Instances
				if len(m.ec2FilteredInstances) > 0 {
					instances = m.ec2FilteredInstances
				}
				if len(instances) > 0 && m.ec2SelectedIndex < len(instances) {
					instanceID := instances[m.ec2SelectedIndex].ID
					if m.ec2SelectedInstances[instanceID] {
						delete(m.ec2SelectedInstances, instanceID)
					} else {
						m.ec2SelectedInstances[instanceID] = true
					}
					return m, nil
				}
			}
		case "a":
			// Toggle auto-refresh
			if m.currentScreen == ec2Screen {
				m.autoRefresh = !m.autoRefresh
				if m.autoRefresh {
					m.statusMessage = "Auto-refresh enabled (30s)"
					return m, tickCmd()
				} else {
					m.statusMessage = "Auto-refresh disabled"
				}
				return m, nil
			}
		case "x":
			// Clear all selections
			if m.currentScreen == ec2Screen {
				m.ec2SelectedInstances = make(map[string]bool)
				m.statusMessage = "Cleared all selections"
				return m, nil
			}
		case "y":
			// Copy instance ID or IP to clipboard
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				instance := m.ec2Instances[m.ec2SelectedIndex]
				// Try to copy public IP, fallback to private IP, then instance ID
				toCopy := instance.PublicIP
				if toCopy == "" {
					toCopy = instance.PrivateIP
				}
				if toCopy == "" {
					toCopy = instance.ID
				}
				m.copyToClipboard = toCopy
				m.statusMessage = fmt.Sprintf("Copied to clipboard: %s", toCopy)
				return m, func() tea.Msg {
					// Try to copy to clipboard using xclip or pbcopy
					return nil
				}
			}
		case "s":
			// Start instance (single or bulk)
			if m.currentScreen == ec2Screen && len(m.ec2SelectedInstances) > 0 {
				// Bulk action
				var instanceIDs []string
				for id := range m.ec2SelectedInstances {
					instanceIDs = append(instanceIDs, id)
				}
				m.loading = true
				return m, m.performBulkAction("start", instanceIDs)
			}
			var instanceID string
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				instanceID = m.ec2Instances[m.ec2SelectedIndex].ID
			} else if m.currentScreen == ec2DetailsScreen && m.ec2InstanceDetails != nil {
				instanceID = m.ec2InstanceDetails.ID
			}
			if instanceID != "" {
				m.showingConfirm = true
				m.confirmAction = "start"
				m.confirmInstanceID = instanceID
				return m, nil
			}
		case "S":
			// Stop instance (single or bulk)
			if m.currentScreen == ec2Screen && len(m.ec2SelectedInstances) > 0 {
				// Bulk action
				var instanceIDs []string
				for id := range m.ec2SelectedInstances {
					instanceIDs = append(instanceIDs, id)
				}
				m.showingConfirm = true
				m.confirmAction = "stop"
				m.confirmInstanceID = fmt.Sprintf("%d instances", len(instanceIDs))
				return m, nil
			}
			var instanceID string
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				instanceID = m.ec2Instances[m.ec2SelectedIndex].ID
			} else if m.currentScreen == ec2DetailsScreen && m.ec2InstanceDetails != nil {
				instanceID = m.ec2InstanceDetails.ID
			}
			if instanceID != "" {
				m.showingConfirm = true
				m.confirmAction = "stop"
				m.confirmInstanceID = instanceID
				return m, nil
			}
		case "R":
			// Reboot instance (single or bulk)
			if m.currentScreen == ec2Screen && len(m.ec2SelectedInstances) > 0 {
				// Bulk action
				var instanceIDs []string
				for id := range m.ec2SelectedInstances {
					instanceIDs = append(instanceIDs, id)
				}
				m.showingConfirm = true
				m.confirmAction = "reboot"
				m.confirmInstanceID = fmt.Sprintf("%d instances", len(instanceIDs))
				return m, nil
			}
			var instanceID string
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				instanceID = m.ec2Instances[m.ec2SelectedIndex].ID
			} else if m.currentScreen == ec2DetailsScreen && m.ec2InstanceDetails != nil {
				instanceID = m.ec2InstanceDetails.ID
			}
			if instanceID != "" {
				m.showingConfirm = true
				m.confirmAction = "reboot"
				m.confirmInstanceID = instanceID
				return m, nil
			}
		case "t":
			// Terminate instance (single or bulk)
			if m.currentScreen == ec2Screen && len(m.ec2SelectedInstances) > 0 {
				// Bulk action
				var instanceIDs []string
				for id := range m.ec2SelectedInstances {
					instanceIDs = append(instanceIDs, id)
				}
				m.showingConfirm = true
				m.confirmAction = "terminate"
				m.confirmInstanceID = fmt.Sprintf("%d instances", len(instanceIDs))
				return m, nil
			}
			var instanceID string
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				instanceID = m.ec2Instances[m.ec2SelectedIndex].ID
			} else if m.currentScreen == ec2DetailsScreen && m.ec2InstanceDetails != nil {
				instanceID = m.ec2InstanceDetails.ID
			}
			if instanceID != "" {
				m.showingConfirm = true
				m.confirmAction = "terminate"
				m.confirmInstanceID = instanceID
				return m, nil
			}
		case "C":
			// Launch SSM session (only in details view with SSM connected)
			if m.currentScreen == ec2DetailsScreen && m.ec2InstanceDetails != nil && m.ec2SSMStatus != nil && m.ec2SSMStatus.Connected {
				// Return a message that will trigger SSM session launch
				return m, func() tea.Msg {
					return launchSSMSessionMsg{
						instanceID: m.ec2InstanceDetails.ID,
						region:     m.awsClient.GetRegion(),
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// Helper function to clear all search state
func (m *model) clearSearch() {
	m.vimState.LastSearch = ""
	m.vimState.SearchResults = []int{}
	m.ec2FilteredInstances = nil
	m.s3FilteredBuckets = nil
	m.s3FilteredObjects = nil
	m.ssoFilteredAccounts = nil
	m.eksFilteredClusters = nil
}

// Helper functions for VIM navigation
func (m *model) handleVimNavigation(action vim.NavigationAction) {
	// For detail screens, handle viewport scrolling instead of item navigation
	if m.currentScreen == ec2DetailsScreen || m.currentScreen == s3ObjectDetailsScreen || m.currentScreen == eksDetailsScreen {
		m.handleDetailViewScroll(action)
		return
	}

	var listLength int
	var currentIndex int

	// Determine current list and index (use filtered list if active)
	switch m.currentScreen {
	case accountScreen:
		if len(m.ssoFilteredAccounts) > 0 {
			listLength = len(m.ssoFilteredAccounts)
		} else if m.vimState.LastSearch != "" {
			// Search active but no results
			return
		} else {
			listLength = len(m.ssoAccounts)
		}
		currentIndex = m.ssoSelectedIndex
	case ec2Screen:
		if len(m.ec2FilteredInstances) > 0 {
			listLength = len(m.ec2FilteredInstances)
		} else if m.vimState.LastSearch != "" {
			// Search active but no results
			return
		} else {
			listLength = len(m.ec2Instances)
		}
		currentIndex = m.ec2SelectedIndex
	case s3Screen:
		if len(m.s3FilteredBuckets) > 0 {
			listLength = len(m.s3FilteredBuckets)
		} else if m.vimState.LastSearch != "" {
			// Search active but no results
			return
		} else {
			listLength = len(m.s3Buckets)
		}
		currentIndex = m.s3SelectedIndex
	case s3BrowseScreen:
		if len(m.s3FilteredObjects) > 0 {
			listLength = len(m.s3FilteredObjects)
		} else if m.vimState.LastSearch != "" {
			// Search active but no results
			return
		} else {
			listLength = len(m.s3Objects)
		}
		currentIndex = m.s3ObjectSelectedIndex
	case eksScreen:
		if len(m.eksFilteredClusters) > 0 {
			listLength = len(m.eksFilteredClusters)
		} else if m.vimState.LastSearch != "" {
			// Search active but no results
			return
		} else {
			listLength = len(m.eksClusters)
		}
		currentIndex = m.eksSelectedIndex
	default:
		return // No navigation for other screens
	}

	if listLength == 0 {
		return
	}

	// Calculate new index
	newIndex := vim.CalculateNewIndex(action, currentIndex, listLength, m.pageSize)

	// Set the new index
	m.setSelectedIndex(newIndex)

	// Ensure the selected item is visible in the viewport
	m.ensureVisible(newIndex, listLength)
}

// handleDetailViewScroll handles scrolling in detail views
func (m *model) handleDetailViewScroll(action vim.NavigationAction) {
	// For detail views, we scroll the viewport by lines
	switch action {
	case vim.MoveUp:
		if m.viewportOffset > 0 {
			m.viewportOffset--
		}
	case vim.MoveDown:
		m.viewportOffset++
		// Max will be clamped by renderWithViewport
	case vim.MoveTop:
		m.viewportOffset = 0
	case vim.MoveBottom:
		// Set to a large number, renderWithViewport will clamp it
		m.viewportOffset = 10000
	case vim.MoveHalfPageUp:
		m.viewportOffset -= m.pageSize / 2
		if m.viewportOffset < 0 {
			m.viewportOffset = 0
		}
	case vim.MoveHalfPageDown:
		m.viewportOffset += m.pageSize / 2
	case vim.MovePageUp:
		m.viewportOffset -= m.pageSize
		if m.viewportOffset < 0 {
			m.viewportOffset = 0
		}
	case vim.MovePageDown:
		m.viewportOffset += m.pageSize
	}
}

func (m *model) setSelectedIndex(index int) {
	switch m.currentScreen {
	case accountScreen:
		if index >= 0 && index < len(m.ssoAccounts) {
			m.ssoSelectedIndex = index
		}
	case ec2Screen:
		if index >= 0 && index < len(m.ec2Instances) {
			m.ec2SelectedIndex = index
		}
	case s3Screen:
		if index >= 0 && index < len(m.s3Buckets) {
			m.s3SelectedIndex = index
		}
	case s3BrowseScreen:
		if index >= 0 && index < len(m.s3Objects) {
			m.s3ObjectSelectedIndex = index
		}
	case eksScreen:
		if index >= 0 && index < len(m.eksClusters) {
			m.eksSelectedIndex = index
		}
	}
}

func (m *model) applyVimSearch() {
	// Build searchable strings for current view
	var searchItems []string

	switch m.currentScreen {
	case accountScreen:
		for _, acc := range m.ssoAccounts {
			var sb strings.Builder
			sb.WriteString(acc.AccountID)
			sb.WriteString(" ")
			sb.WriteString(acc.AccountName)
			sb.WriteString(" ")
			sb.WriteString(acc.RoleName)
			sb.WriteString(" ")
			sb.WriteString(acc.EmailAddress)
			searchItems = append(searchItems, strings.ToLower(sb.String()))
		}
	case ec2Screen:
		for _, inst := range m.ec2Instances {
			searchItems = append(searchItems,
				strings.ToLower(inst.ID+" "+inst.Name+" "+inst.State+" "+inst.InstanceType+" "+inst.PublicIP+" "+inst.PrivateIP))
		}
	case s3Screen:
		for _, bucket := range m.s3Buckets {
			searchItems = append(searchItems, strings.ToLower(bucket.Name+" "+bucket.Region))
		}
	case s3BrowseScreen:
		for _, obj := range m.s3Objects {
			searchItems = append(searchItems, strings.ToLower(obj.Key))
		}
	case eksScreen:
		for _, cluster := range m.eksClusters {
			searchItems = append(searchItems, strings.ToLower(cluster.Name+" "+cluster.Version+" "+cluster.Status+" "+cluster.Region))
		}
	default:
		return
	}

	// Perform search
	m.vimState.SearchItems(searchItems)

	// Filter the view to only show matching items
	if len(m.vimState.SearchResults) > 0 {
		switch m.currentScreen {
		case accountScreen:
			m.ssoFilteredAccounts = make([]aws.SSOAccount, 0, len(m.vimState.SearchResults))
			for _, idx := range m.vimState.SearchResults {
				m.ssoFilteredAccounts = append(m.ssoFilteredAccounts, m.ssoAccounts[idx])
			}
		case ec2Screen:
			m.ec2FilteredInstances = make([]aws.Instance, 0, len(m.vimState.SearchResults))
			for _, idx := range m.vimState.SearchResults {
				m.ec2FilteredInstances = append(m.ec2FilteredInstances, m.ec2Instances[idx])
			}
		case s3Screen:
			m.s3FilteredBuckets = make([]aws.Bucket, 0, len(m.vimState.SearchResults))
			for _, idx := range m.vimState.SearchResults {
				m.s3FilteredBuckets = append(m.s3FilteredBuckets, m.s3Buckets[idx])
			}
		case s3BrowseScreen:
			m.s3FilteredObjects = make([]aws.S3Object, 0, len(m.vimState.SearchResults))
			for _, idx := range m.vimState.SearchResults {
				m.s3FilteredObjects = append(m.s3FilteredObjects, m.s3Objects[idx])
			}
		case eksScreen:
			m.eksFilteredClusters = make([]aws.EKSCluster, 0, len(m.vimState.SearchResults))
			for _, idx := range m.vimState.SearchResults {
				m.eksFilteredClusters = append(m.eksFilteredClusters, m.eksClusters[idx])
			}
		}

		// Reset selection to first filtered result
		m.setSelectedIndex(0)
		m.statusMessage = fmt.Sprintf("Showing %d matching results (ESC or :cf to clear)", len(m.vimState.SearchResults))
	} else {
		m.statusMessage = "No matches found"
		// Clear filtered lists to show "no matches"
		switch m.currentScreen {
		case accountScreen:
			m.ssoFilteredAccounts = []aws.SSOAccount{}
		case ec2Screen:
			m.ec2FilteredInstances = []aws.Instance{}
		case s3Screen:
			m.s3FilteredBuckets = []aws.Bucket{}
		case s3BrowseScreen:
			m.s3FilteredObjects = []aws.S3Object{}
		case eksScreen:
			m.eksFilteredClusters = []aws.EKSCluster{}
		}
	}
}

func (m *model) executeVimCommand(commandStr string) tea.Cmd {
	cmd := vim.ParseCommand(commandStr)

	switch cmd.Name {
	case vim.CmdQuit, "quit":
		// Quit current view or app
		if m.currentScreen == ec2DetailsScreen {
			m.currentScreen = ec2Screen
			m.ec2InstanceDetails = nil
		} else if m.currentScreen == s3BrowseScreen {
			m.currentScreen = s3Screen
			m.s3Objects = nil
			m.s3CurrentBucket = ""
			m.s3CurrentPrefix = ""
		} else if m.currentScreen == s3ObjectDetailsScreen {
			m.currentScreen = s3BrowseScreen
			m.s3ObjectDetails = nil
		} else {
			return tea.Quit
		}
		return nil

	case vim.CmdRefresh, "refresh":
		// Refresh current view
		m.loading = true
		if m.currentScreen == ec2Screen {
			return m.loadEC2Instances
		} else if m.currentScreen == s3Screen {
			return m.loadS3Buckets
		} else if m.currentScreen == s3BrowseScreen {
			return m.loadS3Objects(m.s3CurrentBucket, m.s3CurrentPrefix, nil)
		}

	case vim.CmdSelectAll:
		// Select all instances (EC2 only)
		if m.currentScreen == ec2Screen {
			for _, inst := range m.ec2Instances {
				m.ec2SelectedInstances[inst.ID] = true
			}
			m.statusMessage = fmt.Sprintf("Selected all %d instances", len(m.ec2Instances))
		}

	case vim.CmdDeselectAll:
		// Deselect all instances
		if m.currentScreen == ec2Screen {
			m.ec2SelectedInstances = make(map[string]bool)
			m.statusMessage = "Cleared all selections"
		}

	case vim.CmdClearFilter, "clearfilter":
		// Clear filter and reset filtered lists
		m.filter = ""
		m.vimState.LastSearch = ""
		m.vimState.SearchResults = []int{}
		m.ec2FilteredInstances = nil
		m.s3FilteredBuckets = nil
		m.s3FilteredObjects = nil
		m.statusMessage = "Filter cleared"

	case vim.CmdHelp, "h", "?":
		// Show help message
		m.statusMessage = "VIM commands: :q (quit), :r (refresh), :sa (select all), :da (deselect all), :cf (clear filter), :ec2/:s3/:eks (switch service)"

	case vim.CmdEC2:
		// Switch to EC2 service
		m.clearSearch() // Clear search when switching screens
		m.currentScreen = ec2Screen
		m.viewportOffset = 0
		if len(m.ec2Instances) == 0 {
			m.loading = true
			return m.loadEC2Instances
		}
		m.statusMessage = "Switched to EC2"

	case vim.CmdS3:
		// Switch to S3 service
		m.clearSearch() // Clear search when switching screens
		m.currentScreen = s3Screen
		m.viewportOffset = 0
		if len(m.s3Buckets) == 0 {
			m.loading = true
			return m.loadS3Buckets
		}
		m.statusMessage = "Switched to S3"

	case vim.CmdEKS:
		// Switch to EKS service
		m.clearSearch() // Clear search when switching screens
		m.currentScreen = eksScreen
		m.viewportOffset = 0
		if len(m.eksClusters) == 0 {
			m.loading = true
			return m.loadEKSClusters
		}
		m.statusMessage = "Switched to EKS"

	case vim.CmdAccount, "acc":
		// Switch to account selection screen
		// Only works with SSO auth method
		if m.authConfig == nil || m.authConfig.Method != aws.AuthMethodSSO {
			m.statusMessage = "Account switching only available with SSO authentication"
			return nil
		}

		if m.ssoAuthenticator == nil {
			// Start SSO authentication flow
			m.loading = true
			m.statusMessage = "Starting SSO authentication - opening browser..."
			return m.authenticateSSO(m.authConfig.SSOStartURL, m.authConfig.SSORegion)
		}
		// Show account selection screen
		m.currentScreen = accountScreen
		m.viewportOffset = 0
		if len(m.ssoAccounts) == 0 {
			m.loading = true
			return m.loadSSOAccounts()
		}
		m.statusMessage = "Account selection"

	case vim.CmdRegion:
		// Cycle to next region
		if m.config == nil || len(m.config.Regions) == 0 {
			m.statusMessage = "No regions configured"
			return nil
		}
		// Find current region index
		currentIndex := -1
		for i, r := range m.config.Regions {
			if r == m.config.Region {
				currentIndex = i
				break
			}
		}
		// Cycle to next region
		if currentIndex != -1 {
			nextIndex := (currentIndex + 1) % len(m.config.Regions)
			m.config.Region = m.config.Regions[nextIndex]
			m.loading = true
			m.statusMessage = fmt.Sprintf("Switching to region: %s", m.config.Region)
			return m.initAWSClient
		} else {
			m.statusMessage = "Current region not found in config"
			return nil
		}

	default:
		m.statusMessage = fmt.Sprintf("Unknown command: %s", cmd.Name)
	}

	return nil
}

// ensureVisible adjusts viewport offset to keep the selected item visible
func (m *model) ensureVisible(selectedIndex, listLength int) {
	if listLength == 0 {
		m.viewportOffset = 0
		return
	}

	// Calculate available height for the list data rows only
	// Account for: k9s header (9 lines), content border/padding (4 lines),
	//              table title+header (2 lines), footer info (2 lines), breadcrumb (3 lines),
	//              vim command line (2 lines)
	availableHeight := m.height - 22
	if availableHeight < 5 {
		availableHeight = 5 // Minimum viewport size
	}

	// Ensure selected item is visible
	if selectedIndex < m.viewportOffset {
		// Selected item is above viewport, scroll up
		m.viewportOffset = selectedIndex
	} else if selectedIndex >= m.viewportOffset+availableHeight {
		// Selected item is below viewport, scroll down
		m.viewportOffset = selectedIndex - availableHeight + 1
	}

	// Clamp viewport offset
	maxOffset := listLength - availableHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.viewportOffset > maxOffset {
		m.viewportOffset = maxOffset
	}
	if m.viewportOffset < 0 {
		m.viewportOffset = 0
	}
}

// getVisibleRange returns the start and end indices for items to display
func (m *model) getVisibleRange(listLength int) (int, int) {
	if listLength == 0 {
		return 0, 0
	}

	// Match the calculation in ensureVisible - MUST BE THE SAME
	availableHeight := m.height - 22
	if availableHeight < 5 {
		availableHeight = 5
	}

	start := m.viewportOffset
	end := m.viewportOffset + availableHeight
	if end > listLength {
		end = listLength
	}

	return start, end
}

// renderWithViewport takes a multi-line string and returns only visible lines
func (m *model) renderWithViewport(content string) string {
	lines := strings.Split(content, "\n")

	// Calculate available height for content
	// Must match the calculations in ensureVisible and getVisibleRange
	// Account for: k9s header (9 lines), content border/padding (4 lines),
	//              breadcrumb (3 lines), vim command line (2 lines), scroll indicator (1 line)
	availableHeight := m.height - 19
	if availableHeight < 10 {
		availableHeight = 10 // Minimum viewport
	}

	totalLines := len(lines)
	if totalLines <= availableHeight {
		// Content fits, no need to scroll
		return content
	}

	// Apply viewport
	start := m.viewportOffset
	end := start + availableHeight
	if end > totalLines {
		end = totalLines
	}
	if start >= totalLines {
		start = totalLines - availableHeight
		if start < 0 {
			start = 0
		}
	}

	visibleLines := lines[start:end]
	result := strings.Join(visibleLines, "\n")

	// Add scroll indicator
	scrollInfo := fmt.Sprintf("\n[Lines %d-%d of %d] (j/k to scroll)", start+1, end, totalLines)
	result += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(scrollInfo)

	return result
}

func (m model) View() string {
	var s string

	// K9s-style header: left sidebar with context info, center/right with key hints
	s += m.renderK9sHeader() + "\n"

	// Content area
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2).
		Width(m.width - 2) // Full width minus small margin

	// Only apply MaxHeight for main screens that need viewport control
	// Auth screens should show full content
	if m.currentScreen >= ec2Screen {
		// Set max height to prevent overflow and top clipping
		// Leave room for header (~7 lines) + border (2) + padding (2) + breadcrumb (1) + status (1) = ~13 total
		// But be less aggressive - use height - 8 to give more content space
		maxContentHeight := m.height - 8
		if maxContentHeight < 15 {
			maxContentHeight = 15
		}
		contentStyle = contentStyle.MaxHeight(maxContentHeight)
	}

	var content string
	switch m.currentScreen {
	case authMethodScreen:
		content = m.renderAuthMethodSelection()
	case authProfileScreen:
		content = m.renderProfileConfig()
	case ssoConfigScreen:
		content = m.renderSSOConfig()
	case accountScreen:
		content = m.renderAccountSelection()
	case ec2Screen:
		content = m.renderEC2()
	case ec2DetailsScreen:
		content = m.renderEC2Details()
	case s3Screen:
		content = m.renderS3()
	case s3BrowseScreen:
		content = m.renderS3Browse()
	case s3ObjectDetailsScreen:
		content = m.renderS3ObjectDetails()
	case eksScreen:
		content = m.renderEKS()
	case eksDetailsScreen:
		content = m.renderEKSDetails()
	}

	if m.filtering {
		s += "\n" + m.filterInput.View()
	}

	s += contentStyle.Render(content) + "\n"

	// Show delete confirmation input
	if m.s3ConfirmDelete {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)
		s += "\n" + confirmStyle.Render("DELETE CONFIRMATION")
		s += "\n" + m.deleteConfirmInput.View()
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Press ESC to cancel")
	}

	// Show VIM mode indicator
	if m.vimState.Mode == vim.SearchMode {
		searchStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("3")).
			Background(lipgloss.Color("0"))
		s += "\n" + searchStyle.Render("/"+m.vimState.SearchQuery)
	} else if m.vimState.Mode == vim.CommandMode {
		commandStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			Background(lipgloss.Color("0"))
		s += "\n" + commandStyle.Render(":"+m.vimState.CommandBuffer)

		// Show command suggestions
		if len(m.commandSuggestions) > 0 {
			suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			suggestions := strings.Join(m.commandSuggestions, ", ")
			s += "\n" + suggestionStyle.Render("  suggestions: "+suggestions)
		}
	}

	// Show confirmation dialog
	if m.showingConfirm {
		confirmStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("3")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(1, 2)

		actionText := m.confirmAction
		if m.confirmAction == "terminate" {
			actionText = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render("TERMINATE")
		}

		confirmMsg := fmt.Sprintf("Are you sure you want to %s instance %s?\n\n(y)es / (n)o",
			actionText, m.confirmInstanceID)
		s += "\n" + confirmStyle.Render(confirmMsg)
	}

	// Show S3 info popup (bucket policy, versioning, presigned URL)
	if m.s3ShowingInfo {
		infoStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(1, 2).
			Width(80)

		var infoContent string
		if m.s3InfoType == "policy" {
			infoContent = "Bucket Policy:\n\n" + m.s3BucketPolicy
		} else if m.s3InfoType == "versioning" {
			infoContent = "Bucket Versioning:\n\n" + m.s3BucketVersioning
		}
		s += "\n" + infoStyle.Render(infoContent)
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("Press ESC to close")
	}

	// Show presigned URL (only on non-macOS platforms, since macOS auto-copies)
	if m.s3PresignedURL != "" && runtime.GOOS != "darwin" {
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
		s += "\n" + labelStyle.Render("Presigned URL (1 hour): ") + urlStyle.Render(m.s3PresignedURL)
	}

	// Show status message
	if m.statusMessage != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		s += "\n" + statusStyle.Render(m.statusMessage)
	}

	// K9s-style breadcrumb navigation at bottom
	s += "\n" + m.renderK9sBreadcrumb()

	return s
}

// renderK9sHeader creates a k9s-style header with context info and key hints
func (m model) renderK9sHeader() string {
	// K9s color scheme
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))                 // Yellow/orange
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))                 // White
	keyHintKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Bold(true) // Magenta
	keyHintActionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))           // Gray

	// Left side: Context information
	var leftSide strings.Builder

	// Service name (like "Context" in k9s)
	var serviceName, viewName string
	switch m.currentScreen {
	case authMethodScreen:
		serviceName = "Setup"
		viewName = "Authentication"
	case authProfileScreen:
		serviceName = "Setup"
		viewName = "AWS Profile"
	case ssoConfigScreen:
		serviceName = "Setup"
		viewName = "SSO Configuration"
	case accountScreen:
		serviceName = "AWS"
		viewName = "Account Selection"
	case ec2Screen:
		serviceName = "EC2"
		viewName = "Instances"
	case ec2DetailsScreen:
		serviceName = "EC2"
		viewName = "Details"
	case s3Screen:
		serviceName = "S3"
		viewName = "Buckets"
	case s3BrowseScreen:
		serviceName = "S3"
		viewName = "Objects"
	case s3ObjectDetailsScreen:
		serviceName = "S3"
		viewName = "Object Details"
	case eksScreen:
		serviceName = "EKS"
		viewName = "Clusters"
	case eksDetailsScreen:
		serviceName = "EKS"
		viewName = "Cluster Details"
	}

	leftSide.WriteString(labelStyle.Render("Service: ") + valueStyle.Render(serviceName) + "\n")
	leftSide.WriteString(labelStyle.Render("View:    ") + valueStyle.Render(viewName) + "\n")

	if m.awsClient != nil {
		leftSide.WriteString(labelStyle.Render("Region:  ") + valueStyle.Render(m.awsClient.GetRegion()) + "\n")

		// Display account information if available
		if m.currentAccountName != "" {
			leftSide.WriteString(labelStyle.Render("Account: ") + valueStyle.Render(m.currentAccountName))
			if m.currentAccountID != "" {
				leftSide.WriteString(valueStyle.Render(fmt.Sprintf(" (%s)", m.currentAccountID)))
			}
			leftSide.WriteString("\n")
		} else if m.awsClient.GetAccountID() != "" {
			leftSide.WriteString(labelStyle.Render("Account: ") + valueStyle.Render(m.awsClient.GetAccountID()) + "\n")
		}
	}

	// Add version info (like K9s Rev)
	leftSide.WriteString(labelStyle.Render("lazyaws: ") + valueStyle.Render("v0.1.0") + "\n")

	// Right side: Key hints based on current screen
	var keyHints []string
	switch m.currentScreen {
	case ec2Screen:
		keyHints = []string{
			keyHintKeyStyle.Render("<enter>") + " " + keyHintActionStyle.Render("Details"),
			keyHintKeyStyle.Render("<space>") + " " + keyHintActionStyle.Render("Select"),
			keyHintKeyStyle.Render("<s>") + " " + keyHintActionStyle.Render("Start"),
			keyHintKeyStyle.Render("<S>") + " " + keyHintActionStyle.Render("Stop"),
			keyHintKeyStyle.Render("<R>") + " " + keyHintActionStyle.Render("Reboot"),
			keyHintKeyStyle.Render("<t>") + " " + keyHintActionStyle.Render("Terminate"),
			keyHintKeyStyle.Render("<:>") + " " + keyHintActionStyle.Render("Command"),
			keyHintKeyStyle.Render("</>") + " " + keyHintActionStyle.Render("Search"),
		}
	case ec2DetailsScreen:
		keyHints = []string{
			keyHintKeyStyle.Render("<s>") + " " + keyHintActionStyle.Render("Start"),
			keyHintKeyStyle.Render("<S>") + " " + keyHintActionStyle.Render("Stop"),
			keyHintKeyStyle.Render("<R>") + " " + keyHintActionStyle.Render("Reboot"),
			keyHintKeyStyle.Render("<t>") + " " + keyHintActionStyle.Render("Terminate"),
			keyHintKeyStyle.Render("<esc>") + " " + keyHintActionStyle.Render("Back"),
		}
		if m.ec2SSMStatus != nil && m.ec2SSMStatus.Connected {
			keyHints = append(keyHints, keyHintKeyStyle.Render("<C>")+" "+keyHintActionStyle.Render("SSM Connect"))
		}
	case s3Screen:
		keyHints = []string{
			keyHintKeyStyle.Render("<enter>") + " " + keyHintActionStyle.Render("Browse"),
			keyHintKeyStyle.Render("<D>") + " " + keyHintActionStyle.Render("Delete"),
			keyHintKeyStyle.Render("<p>") + " " + keyHintActionStyle.Render("Policy"),
			keyHintKeyStyle.Render("<v>") + " " + keyHintActionStyle.Render("Versioning"),
			keyHintKeyStyle.Render("<:>") + " " + keyHintActionStyle.Render("Command"),
			keyHintKeyStyle.Render("</>") + " " + keyHintActionStyle.Render("Search"),
		}
	case s3BrowseScreen:
		keyHints = []string{
			keyHintKeyStyle.Render("<enter>") + " " + keyHintActionStyle.Render("Open"),
			keyHintKeyStyle.Render("<d>") + " " + keyHintActionStyle.Render("Download"),
			keyHintKeyStyle.Render("<D>") + " " + keyHintActionStyle.Render("Delete"),
			keyHintKeyStyle.Render("<h>") + " " + keyHintActionStyle.Render("Up"),
			keyHintKeyStyle.Render("<:>") + " " + keyHintActionStyle.Render("Command"),
			keyHintKeyStyle.Render("</>") + " " + keyHintActionStyle.Render("Search"),
		}
	case s3ObjectDetailsScreen:
		keyHints = []string{
			keyHintKeyStyle.Render("<d>") + " " + keyHintActionStyle.Render("Download"),
			keyHintKeyStyle.Render("<p>") + " " + keyHintActionStyle.Render("Presigned URL"),
			keyHintKeyStyle.Render("<esc>") + " " + keyHintActionStyle.Render("Back"),
		}
	case eksScreen:
		keyHints = []string{
			keyHintKeyStyle.Render("<enter>") + " " + keyHintActionStyle.Render("Details"),
			keyHintKeyStyle.Render("<K>") + " " + keyHintActionStyle.Render("Update Kubeconfig"),
			keyHintKeyStyle.Render("<:>") + " " + keyHintActionStyle.Render("Command"),
		}
	case eksDetailsScreen:
		keyHints = []string{
			keyHintKeyStyle.Render("<K>") + " " + keyHintActionStyle.Render("Update Kubeconfig"),
			keyHintKeyStyle.Render("<esc>") + " " + keyHintActionStyle.Render("Back"),
		}
	}

	// ASCII art logo (simplified version for lazyaws)
	logo := `  _
 | | __ _ ____ _   _
 | |/ _  |_  /| | | |
 | | (_| |/ / | |_| |
 |_|__,_/___| ___,_|
   __ ___      _____
  / _  |-|-|/| / __|
 | (_| | V  V ||__ |
  __,_| |/|/| |___/`

	// Layout: left side, center spacing, key hints (2 columns), right side logo
	leftContent := leftSide.String()

	// Format key hints in columns
	var rightSide strings.Builder
	for i := 0; i < len(keyHints); i += 2 {
		if i < len(keyHints) {
			rightSide.WriteString(keyHints[i])
			if i+1 < len(keyHints) {
				rightSide.WriteString("  " + keyHints[i+1])
			}
			rightSide.WriteString("\n")
		}
	}

	// Combine left and right with proper spacing
	leftLines := strings.Split(leftContent, "\n")
	rightLines := strings.Split(rightSide.String(), "\n")
	logoLines := strings.Split(logo, "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}
	if len(logoLines) > maxLines {
		maxLines = len(logoLines)
	}

	var header strings.Builder
	for i := 0; i < maxLines; i++ {
		// Build the line in a temporary buffer to measure and truncate if needed
		var line strings.Builder

		// Left side (fixed width ~30 chars)
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		line.WriteString(left)

		// Padding to align
		leftWidth := lipgloss.Width(left)
		padding := 30 - leftWidth
		if padding > 0 {
			line.WriteString(strings.Repeat(" ", padding))
		}

		// Right side key hints (middle section)
		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}
		line.WriteString(right)

		// Logo on far right - but ensure we don't overflow terminal width
		if i < len(logoLines) && m.width > 0 {
			rightWidth := lipgloss.Width(right)
			// Calculate how much space we've used: left (30) + right
			usedWidth := 30 + rightWidth
			// Only add logo if we have room (need at least 30 chars for logo)
			if usedWidth+30 < m.width {
				logoPadding := 90 - rightWidth
				if logoPadding > 0 && logoPadding < 100 { // Sanity check
					line.WriteString(strings.Repeat(" ", logoPadding))
				}
				line.WriteString(labelStyle.Render(logoLines[i]))
			}
		}

		// Truncate line to terminal width if needed
		lineStr := line.String()
		lineWidth := lipgloss.Width(lineStr)
		if m.width > 0 && lineWidth > m.width {
			// Line is too long, truncate it
			// This is tricky with ANSI codes, so just skip the logo for this line
			var safeLine strings.Builder
			safeLine.WriteString(left)
			if padding > 0 {
				safeLine.WriteString(strings.Repeat(" ", padding))
			}
			safeLine.WriteString(right)
			lineStr = safeLine.String()
		}

		header.WriteString(lineStr)
		// Add ANSI clear-to-end-of-line to prevent any artifacts
		header.WriteString("\x1b[K")
		header.WriteString("\n")
	}

	return header.String()
}

// renderK9sBreadcrumb creates a k9s-style breadcrumb navigation at the bottom
func (m model) renderK9sBreadcrumb() string {
	breadcrumbStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("220")).
		Bold(true).
		Padding(0, 1)

	var breadcrumbs []string

	switch m.currentScreen {
	case ec2Screen:
		breadcrumbs = []string{"<ec2>", "<instances>"}
	case ec2DetailsScreen:
		breadcrumbs = []string{"<ec2>", "<instances>", "<details>"}
	case s3Screen:
		breadcrumbs = []string{"<s3>", "<buckets>"}
	case s3BrowseScreen:
		if m.s3CurrentBucket != "" {
			breadcrumbs = []string{"<s3>", "<" + m.s3CurrentBucket + ">"}
			if m.s3CurrentPrefix != "" {
				breadcrumbs = append(breadcrumbs, "<"+m.s3CurrentPrefix+">")
			}
		}
	case s3ObjectDetailsScreen:
		breadcrumbs = []string{"<s3>", "<object>", "<details>"}
	case eksScreen:
		breadcrumbs = []string{"<eks>", "<clusters>"}
	case eksDetailsScreen:
		if m.eksClusterDetails != nil {
			breadcrumbs = []string{"<eks>", "<clusters>", "<" + m.eksClusterDetails.Name + ">"}
		} else {
			breadcrumbs = []string{"<eks>", "<clusters>", "<details>"}
		}
	}

	var result strings.Builder
	for i, bc := range breadcrumbs {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(breadcrumbStyle.Render(bc))
	}

	return result.String()
}

func (m model) renderAuthMethodSelection() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("51")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	var content strings.Builder

	content.WriteString(titleStyle.Render("Welcome to lazyaws!") + "\n\n")
	content.WriteString(labelStyle.Render("Choose your AWS authentication method:") + "\n\n")

	// Option 0: Environment Variables
	envAvailable := aws.CheckEnvVarsAvailable()
	envText := "Environment Variables"
	if envAvailable {
		envText += " (detected)"
	}
	if m.selectedAuthMethod == 0 {
		content.WriteString(selectedStyle.Render("> "+envText) + "\n")
	} else {
		content.WriteString(normalStyle.Render("  "+envText) + "\n")
	}
	content.WriteString(instructionStyle.Render("  Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY") + "\n\n")

	// Option 1: AWS Profile
	profileText := "AWS Profile"
	if m.selectedAuthMethod == 1 {
		content.WriteString(selectedStyle.Render("> "+profileText) + "\n")
	} else {
		content.WriteString(normalStyle.Render("  "+profileText) + "\n")
	}
	content.WriteString(instructionStyle.Render("  Uses credentials from ~/.aws/config or ~/.aws/credentials") + "\n\n")

	// Option 2: SSO
	ssoText := "AWS SSO (IAM Identity Center)"
	if m.selectedAuthMethod == 2 {
		content.WriteString(selectedStyle.Render("> "+ssoText) + "\n")
	} else {
		content.WriteString(normalStyle.Render("  "+ssoText) + "\n")
	}
	content.WriteString(instructionStyle.Render("  Uses AWS Single Sign-On for multi-account access") + "\n\n\n")

	content.WriteString(instructionStyle.Render("Use ↑/↓ or j/k to select | Enter to continue | ESC to quit") + "\n")

	return content.String()
}

func (m model) renderProfileConfig() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	var content strings.Builder

	content.WriteString(titleStyle.Render("AWS Profile Configuration") + "\n\n")
	content.WriteString(labelStyle.Render("Enter the name of your AWS profile:") + "\n\n")

	content.WriteString(instructionStyle.Render("This should match a profile name in:") + "\n")
	content.WriteString(instructionStyle.Render("  • ~/.aws/credentials") + "\n")
	content.WriteString(instructionStyle.Render("  • ~/.aws/config") + "\n\n")

	content.WriteString(instructionStyle.Render("Common profile names: default, dev, prod, staging") + "\n\n")

	content.WriteString(labelStyle.Render("Profile name:") + "\n")
	content.WriteString(m.profileInput.View() + "\n\n")

	if m.statusMessage != "" {
		content.WriteString(errorStyle.Render(m.statusMessage) + "\n\n")
	}

	content.WriteString(instructionStyle.Render("Press Enter to save | ESC to go back") + "\n")

	return content.String()
}

func (m model) renderSSOConfig() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	var content strings.Builder

	content.WriteString(titleStyle.Render("AWS SSO Configuration") + "\n\n")
	content.WriteString(labelStyle.Render("To use lazyaws, you need to configure your AWS SSO (IAM Identity Center) start URL.") + "\n\n")

	content.WriteString(instructionStyle.Render("This is typically a URL like:") + "\n")
	content.WriteString(instructionStyle.Render("  https://d-xxxxxxxxxx.awsapps.com/start") + "\n\n")

	content.WriteString(instructionStyle.Render("You can find this URL in:") + "\n")
	content.WriteString(instructionStyle.Render("  • AWS IAM Identity Center console") + "\n")
	content.WriteString(instructionStyle.Render("  • Your AWS SSO login page") + "\n")
	content.WriteString(instructionStyle.Render("  • Your organization's AWS access portal") + "\n\n")

	content.WriteString(labelStyle.Render("Enter your SSO Start URL:") + "\n")
	content.WriteString(m.ssoURLInput.View() + "\n\n")

	if m.statusMessage != "" {
		content.WriteString(errorStyle.Render(m.statusMessage) + "\n\n")
	}

	content.WriteString(instructionStyle.Render("Press Enter to save | ESC to go back") + "\n")

	return content.String()
}

func (m model) renderAccountSelection() string {
	title := lipgloss.NewStyle().Bold(true).Render("AWS Account Selection")
	if m.vimState.LastSearch != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(fmt.Sprintf(" [search: %s]", m.vimState.LastSearch))
	}

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading accounts...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Use filtered accounts if VIM search is active, otherwise use all accounts
	accounts := m.ssoAccounts
	if len(m.ssoFilteredAccounts) > 0 {
		accounts = m.ssoFilteredAccounts
	} else if m.vimState.LastSearch != "" {
		// Search is active but no results
		accounts = []aws.SSOAccount{}
	}

	if len(accounts) == 0 {
		if m.vimState.LastSearch != "" {
			return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No accounts match your search")
		}
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No accounts found")
	}

	// Ensure selected item is visible and get viewport range
	m.ensureVisible(m.ssoSelectedIndex, len(accounts))
	start, end := m.getVisibleRange(len(accounts))

	// Build table header (k9s style)
	var content strings.Builder

	// Title with count - k9s style
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true) // Cyan
	searchInfo := ""
	if m.vimState.LastSearch != "" {
		searchInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render("(" + m.vimState.LastSearch + ")")
	}
	tableTitle := fmt.Sprintf("AWS-Accounts%s[%d]", searchInfo, len(accounts))
	titleText := titleStyle.Render(tableTitle)

	// Center the title with dashes on both sides
	titleWidth := len(tableTitle)
	totalWidth := 120
	dashesWidth := (totalWidth - titleWidth - 2) / 2
	if dashesWidth < 1 {
		dashesWidth = 1
	}

	content.WriteString(strings.Repeat("─", dashesWidth) + " ")
	content.WriteString(titleText)
	content.WriteString(" " + strings.Repeat("─", dashesWidth) + "\n")

	// Table header - k9s uses uppercase and symbols
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
	content.WriteString(headerStyle.Render(fmt.Sprintf("%-30s %-20s %-35s %-30s",
		"ACCOUNT NAME", "ACCOUNT ID", "ROLE", "EMAIL")) + "\n")

	// Build table rows (only visible items)
	for i := start; i < end; i++ {
		account := accounts[i]

		// Build row with proper spacing
		row := fmt.Sprintf("%-30s %-20s %-35s %-30s",
			truncate(account.AccountName, 30),
			account.AccountID,
			truncate(account.RoleName, 35),
			truncate(account.EmailAddress, 30),
		)

		if i == m.ssoSelectedIndex {
			// Highlight the selected row - k9s style with cyan background
			for len(row) < 118 {
				row += " "
			}
			selectedStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("51")).
				Foreground(lipgloss.Color("0")).
				Bold(true)
			content.WriteString(selectedStyle.Render(row) + "\n")
		} else {
			// Normal row
			normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
			content.WriteString(normalStyle.Render(row) + "\n")
		}
	}

	// Add scroll indicators
	if start > 0 || end < len(accounts) {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d | Use j/k or ↓/↑ to navigate]", start+1, end, len(accounts))
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(scrollInfo))
	}

	return content.String()
}

func (m model) renderEC2() string {
	title := lipgloss.NewStyle().Bold(true).Render("EC2 Instances")
	if m.filter != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf(" (filtered by: %s)", m.filter))
	}
	if m.vimState.LastSearch != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(fmt.Sprintf(" [search: %s]", m.vimState.LastSearch))
	}

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading instances...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Use filtered instances if VIM search is active, otherwise use legacy filter or all instances
	var filteredInstances []aws.Instance
	if len(m.ec2FilteredInstances) > 0 {
		filteredInstances = m.ec2FilteredInstances
	} else if m.vimState.LastSearch != "" {
		// Search is active but no results
		filteredInstances = []aws.Instance{}
	} else if m.filter == "" {
		filteredInstances = m.ec2Instances
	} else {
		if strings.Contains(m.filter, "=") {
			parts := strings.SplitN(m.filter, "=", 2)
			tagKey := parts[0]
			tagValue := parts[1]
			for _, inst := range m.ec2Instances {
				for _, tag := range inst.Tags {
					if tag.Key == tagKey && strings.Contains(strings.ToLower(tag.Value), strings.ToLower(tagValue)) {
						filteredInstances = append(filteredInstances, inst)
						break
					}
				}
			}
		} else {
			for _, inst := range m.ec2Instances {
				if strings.Contains(strings.ToLower(inst.State), strings.ToLower(m.filter)) || strings.Contains(strings.ToLower(inst.Name), strings.ToLower(m.filter)) || strings.Contains(strings.ToLower(inst.ID), strings.ToLower(m.filter)) {
					filteredInstances = append(filteredInstances, inst)
				}
			}
		}
	}

	if len(filteredInstances) == 0 {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No instances found")
	}

	// Ensure selected item is visible and get viewport range
	m.ensureVisible(m.ec2SelectedIndex, len(filteredInstances))
	start, end := m.getVisibleRange(len(filteredInstances))

	// Build table header (k9s style)
	var content strings.Builder

	// Title with count - k9s style
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true) // Cyan
	searchInfo := ""
	if m.vimState.LastSearch != "" {
		searchInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render("(" + m.vimState.LastSearch + ")")
	}
	tableTitle := fmt.Sprintf("EC2-Instances%s[%d]", searchInfo, len(filteredInstances))
	titleText := titleStyle.Render(tableTitle)

	// Center the title with dashes on both sides
	// Use actual string width not ANSI-coded width
	titleWidth := len(tableTitle)                    // Visual width without ANSI codes
	totalWidth := 100                                // Reasonable fixed width for the table
	dashesWidth := (totalWidth - titleWidth - 2) / 2 // -2 for spaces around title
	if dashesWidth < 1 {
		dashesWidth = 1
	}

	content.WriteString(strings.Repeat("─", dashesWidth) + " ")
	content.WriteString(titleText)
	content.WriteString(" " + strings.Repeat("─", dashesWidth) + "\n")

	// Table header - k9s uses uppercase and symbols
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
	content.WriteString(headerStyle.Render(fmt.Sprintf("%-1s  %-20s %-30s %-15s %-15s %-15s",
		"✓", "INSTANCE ID", "NAME", "STATE", "TYPE", "IP")) + "\n")

	// Build table rows (only visible items)
	for i := start; i < end; i++ {
		inst := filteredInstances[i]
		name := inst.Name
		if name == "" {
			name = "-" // Don't use lipgloss styling - it breaks alignment
		}

		ip := inst.PublicIP
		if ip == "" {
			ip = inst.PrivateIP
		}
		if ip == "" {
			ip = "-"
		}

		// Check if instance is selected for bulk action
		checkmarkStr := " "
		if m.ec2SelectedInstances[inst.ID] {
			checkmarkStr = "✓"
		}

		// Build row with proper spacing (format first, then apply colors to specific fields)
		// Don't use styled strings in sprintf as ANSI codes break alignment
		row := fmt.Sprintf("%-1s  %-20s %-30s %-15s %-15s %-15s",
			checkmarkStr,
			inst.ID,
			truncate(name, 30),
			inst.State,
			inst.InstanceType,
			ip,
		)

		// Apply state color to the state field within the row
		// (we'll handle this differently to maintain alignment)

		if i == m.ec2SelectedIndex {
			// Highlight the selected row - k9s style with cyan background
			// Ensure row is padded to exactly 98 characters (the column widths sum)
			// This prevents any terminal interpretation issues
			for len(row) < 98 {
				row += " "
			}
			// Use ANSI codes directly to avoid lipgloss adding extra width
			// \x1b[48;5;51m = cyan background, \x1b[38;5;0m = black foreground, \x1b[1m = bold, \x1b[0m = reset
			row = "\x1b[48;5;51m\x1b[38;5;0m\x1b[1m" + row + "\x1b[0m"
		}

		content.WriteString(row + "\n")
	}

	selectedCount := len(m.ec2SelectedInstances)

	// Show scroll position and total
	scrollInfo := fmt.Sprintf("\nShowing %d-%d of %d instances", start+1, end, len(filteredInstances))
	content.WriteString(scrollInfo)

	if selectedCount > 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(fmt.Sprintf(" | Selected: %d", selectedCount)))
	}
	if m.autoRefresh {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(" | Auto-refresh: ON"))
	}

	return content.String()
}

func (m model) renderEC2Details() string {
	title := lipgloss.NewStyle().Bold(true).Render("EC2 Instance Details")

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading instance details...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.ec2InstanceDetails == nil {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No instance details available")
	}

	details := m.ec2InstanceDetails
	var content strings.Builder
	content.WriteString(title + "\n\n")

	// Section styling
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle := lipgloss.NewStyle()

	// Basic Information
	content.WriteString(sectionStyle.Render("Basic Information") + "\n")
	content.WriteString(labelStyle.Render("  Instance ID:     ") + valueStyle.Render(details.ID) + "\n")
	content.WriteString(labelStyle.Render("  Name:            ") + valueStyle.Render(details.Name) + "\n")
	content.WriteString(labelStyle.Render("  State:           ") + getStateStyle(details.State).Render(details.State) + "\n")
	content.WriteString(labelStyle.Render("  Instance Type:   ") + valueStyle.Render(details.InstanceType) + "\n")
	content.WriteString(labelStyle.Render("  Architecture:    ") + valueStyle.Render(details.Architecture) + "\n")
	if details.Platform != "" {
		content.WriteString(labelStyle.Render("  Platform:        ") + valueStyle.Render(details.Platform) + "\n")
	}
	if details.LaunchTime != "" {
		content.WriteString(labelStyle.Render("  Launch Time:     ") + valueStyle.Render(details.LaunchTime) + "\n")
	}
	if details.KeyName != "" {
		content.WriteString(labelStyle.Render("  Key Name:        ") + valueStyle.Render(details.KeyName) + "\n")
	}
	content.WriteString("\n")

	// Instance Type Specifications
	if details.InstanceTypeInfo != nil {
		typeInfo := details.InstanceTypeInfo
		content.WriteString(sectionStyle.Render("Instance Type Specifications") + "\n")
		if typeInfo.VCpus > 0 {
			content.WriteString(labelStyle.Render("  vCPUs:           ") + valueStyle.Render(fmt.Sprintf("%d", typeInfo.VCpus)) + "\n")
		}
		if typeInfo.Memory > 0 {
			memoryGB := float64(typeInfo.Memory) / 1024.0
			content.WriteString(labelStyle.Render("  Memory:          ") + valueStyle.Render(fmt.Sprintf("%.2f GiB", memoryGB)) + "\n")
		}
		if typeInfo.NetworkPerformance != "" {
			content.WriteString(labelStyle.Render("  Network:         ") + valueStyle.Render(typeInfo.NetworkPerformance) + "\n")
		}
		if typeInfo.StorageType != "" {
			storageInfo := typeInfo.StorageType
			if typeInfo.InstanceStorageGB > 0 {
				storageInfo += fmt.Sprintf(" (%d GB)", typeInfo.InstanceStorageGB)
			}
			content.WriteString(labelStyle.Render("  Storage:         ") + valueStyle.Render(storageInfo) + "\n")
		}
		if typeInfo.EbsOptimized {
			content.WriteString(labelStyle.Render("  EBS Optimized:   ") + valueStyle.Render("Yes") + "\n")
		}
		content.WriteString("\n")
	}

	// Network Information
	content.WriteString(sectionStyle.Render("Network Information") + "\n")
	content.WriteString(labelStyle.Render("  VPC ID:          ") + valueStyle.Render(details.VpcID) + "\n")
	content.WriteString(labelStyle.Render("  Subnet ID:       ") + valueStyle.Render(details.SubnetID) + "\n")
	content.WriteString(labelStyle.Render("  Availability Zone: ") + valueStyle.Render(details.AZ) + "\n")
	if details.PublicIP != "" {
		content.WriteString(labelStyle.Render("  Public IP:       ") + valueStyle.Render(details.PublicIP) + "\n")
	}
	if details.PrivateIP != "" {
		content.WriteString(labelStyle.Render("  Private IP:      ") + valueStyle.Render(details.PrivateIP) + "\n")
	}
	content.WriteString("\n")

	// Security Groups
	if len(details.SecurityGroups) > 0 {
		content.WriteString(sectionStyle.Render("Security Groups") + "\n")
		for _, sg := range details.SecurityGroups {
			content.WriteString(labelStyle.Render("  • ") + valueStyle.Render(fmt.Sprintf("%s (%s)", sg.Name, sg.ID)) + "\n")
		}
		content.WriteString("\n")
	}

	// Block Devices
	if len(details.BlockDevices) > 0 {
		content.WriteString(sectionStyle.Render("Block Devices") + "\n")
		for _, bd := range details.BlockDevices {
			volumeInfo := bd.VolumeID
			if bd.VolumeSize > 0 {
				volumeInfo += fmt.Sprintf(" (%d GB, %s)", bd.VolumeSize, bd.VolumeType)
			}
			if bd.DeleteOnTermination {
				volumeInfo += " [Delete on Termination]"
			}
			content.WriteString(labelStyle.Render(fmt.Sprintf("  %s: ", bd.DeviceName)) + valueStyle.Render(volumeInfo) + "\n")
		}
		content.WriteString("\n")
	}

	// Network Interfaces
	if len(details.NetworkInterfaces) > 0 {
		content.WriteString(sectionStyle.Render("Network Interfaces") + "\n")
		for i, ni := range details.NetworkInterfaces {
			content.WriteString(labelStyle.Render(fmt.Sprintf("  Interface %d:\n", i+1)))
			content.WriteString(labelStyle.Render("    ID:           ") + valueStyle.Render(ni.ID) + "\n")
			content.WriteString(labelStyle.Render("    MAC Address:  ") + valueStyle.Render(ni.MacAddress) + "\n")
			content.WriteString(labelStyle.Render("    Private IP:   ") + valueStyle.Render(ni.PrivateIP) + "\n")
			if ni.PublicIP != "" {
				content.WriteString(labelStyle.Render("    Public IP:    ") + valueStyle.Render(ni.PublicIP) + "\n")
			}
			content.WriteString(labelStyle.Render("    Subnet:       ") + valueStyle.Render(ni.SubnetID) + "\n")
			if len(ni.SecurityGroups) > 0 {
				content.WriteString(labelStyle.Render("    Security Groups: "))
				sgNames := make([]string, len(ni.SecurityGroups))
				for j, sg := range ni.SecurityGroups {
					sgNames[j] = sg.Name
				}
				content.WriteString(valueStyle.Render(strings.Join(sgNames, ", ")) + "\n")
			}
			content.WriteString("\n")
		}
	}

	// Health Status
	if m.ec2InstanceStatus != nil {
		content.WriteString(sectionStyle.Render("Health Status") + "\n")

		// System status
		systemStatusColor := lipgloss.Color("1") // Red by default
		if m.ec2InstanceStatus.SystemStatusOk {
			systemStatusColor = lipgloss.Color("2") // Green
		}
		systemStatusStyle := lipgloss.NewStyle().Foreground(systemStatusColor)
		content.WriteString(labelStyle.Render("  System Status:   ") +
			systemStatusStyle.Render(m.ec2InstanceStatus.SystemStatus) + "\n")

		// Instance status
		instanceStatusColor := lipgloss.Color("1") // Red by default
		if m.ec2InstanceStatus.InstanceStatusOk {
			instanceStatusColor = lipgloss.Color("2") // Green
		}
		instanceStatusStyle := lipgloss.NewStyle().Foreground(instanceStatusColor)
		content.WriteString(labelStyle.Render("  Instance Status: ") +
			instanceStatusStyle.Render(m.ec2InstanceStatus.InstanceStatus) + "\n")

		// Scheduled events
		if len(m.ec2InstanceStatus.ScheduledEvents) > 0 {
			content.WriteString(labelStyle.Render("  Scheduled Events:\n"))
			for _, event := range m.ec2InstanceStatus.ScheduledEvents {
				content.WriteString(labelStyle.Render(fmt.Sprintf("    • %s: %s\n",
					event.Code, event.Description)))
				if event.NotBefore != "" {
					content.WriteString(labelStyle.Render(fmt.Sprintf("      Start: %s\n",
						event.NotBefore)))
				}
			}
		}
		content.WriteString("\n")
	}

	// CloudWatch Metrics
	if m.ec2InstanceMetrics != nil {
		content.WriteString(sectionStyle.Render("CloudWatch Metrics (Last 5 Minutes)") + "\n")
		content.WriteString(labelStyle.Render("  CPU Utilization: ") +
			valueStyle.Render(fmt.Sprintf("%.2f%%", m.ec2InstanceMetrics.CPUUtilization)) + "\n")
		content.WriteString(labelStyle.Render("  Network In:      ") +
			valueStyle.Render(fmt.Sprintf("%.2f MB", m.ec2InstanceMetrics.NetworkIn/1024/1024)) + "\n")
		content.WriteString(labelStyle.Render("  Network Out:     ") +
			valueStyle.Render(fmt.Sprintf("%.2f MB", m.ec2InstanceMetrics.NetworkOut/1024/1024)) + "\n")
		if m.ec2InstanceMetrics.DiskReadBytes > 0 || m.ec2InstanceMetrics.DiskWriteBytes > 0 {
			content.WriteString(labelStyle.Render("  Disk Read:       ") +
				valueStyle.Render(fmt.Sprintf("%.2f MB", m.ec2InstanceMetrics.DiskReadBytes/1024/1024)) + "\n")
			content.WriteString(labelStyle.Render("  Disk Write:      ") +
				valueStyle.Render(fmt.Sprintf("%.2f MB", m.ec2InstanceMetrics.DiskWriteBytes/1024/1024)) + "\n")
		}
		content.WriteString("\n")
	}

	// SSM Connectivity
	if m.ec2SSMStatus != nil {
		content.WriteString(sectionStyle.Render("Systems Manager (SSM)") + "\n")
		if m.ec2SSMStatus.Connected {
			connectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
			content.WriteString(labelStyle.Render("  Status:          ") +
				connectStyle.Render("Connected") + "\n")
			content.WriteString(labelStyle.Render("  Ping Status:     ") +
				valueStyle.Render(m.ec2SSMStatus.PingStatus) + "\n")
			if m.ec2SSMStatus.AgentVersion != "" {
				content.WriteString(labelStyle.Render("  Agent Version:   ") +
					valueStyle.Render(m.ec2SSMStatus.AgentVersion) + "\n")
			}
			if m.ec2SSMStatus.PlatformName != "" {
				content.WriteString(labelStyle.Render("  Platform:        ") +
					valueStyle.Render(m.ec2SSMStatus.PlatformName) + "\n")
			}
			if m.ec2SSMStatus.LastPingTime != "" {
				content.WriteString(labelStyle.Render("  Last Ping:       ") +
					valueStyle.Render(m.ec2SSMStatus.LastPingTime) + "\n")
			}
			// Add hint about connecting
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true)
			content.WriteString(labelStyle.Render("  ") +
				hintStyle.Render("Press 'C' to open SSM session in new terminal") + "\n")
		} else {
			disconnectStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
			content.WriteString(labelStyle.Render("  Status:          ") +
				disconnectStyle.Render("Not Connected") + "\n")
			content.WriteString(labelStyle.Render("  Note:            ") +
				valueStyle.Render("SSM agent may not be installed or configured") + "\n")
		}
		content.WriteString("\n")
	}

	// Additional Information
	content.WriteString(sectionStyle.Render("Additional Information") + "\n")
	content.WriteString(labelStyle.Render("  Root Device:     ") + valueStyle.Render(details.RootDeviceType) + "\n")
	if details.Monitoring != "" {
		content.WriteString(labelStyle.Render("  Monitoring:      ") + valueStyle.Render(details.Monitoring) + "\n")
	}
	if details.IamInstanceProfile != "" {
		content.WriteString(labelStyle.Render("  IAM Role:        ") + valueStyle.Render(details.IamInstanceProfile) + "\n")
	}
	content.WriteString("\n")

	// Tags
	if len(details.Tags) > 0 {
		content.WriteString(sectionStyle.Render("Tags") + "\n")
		for _, tag := range details.Tags {
			if tag.Key != "Name" { // Skip Name tag as it's already shown
				content.WriteString(labelStyle.Render(fmt.Sprintf("  %s: ", tag.Key)) + valueStyle.Render(tag.Value) + "\n")
			}
		}
	}

	return m.renderWithViewport(content.String())
}

func (m model) renderS3() string {
	title := lipgloss.NewStyle().Bold(true).Render("S3 Buckets")
	if m.vimState.LastSearch != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(fmt.Sprintf(" [search: %s]", m.vimState.LastSearch))
	}

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading buckets...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Use filtered buckets if VIM search is active
	buckets := m.s3Buckets
	if len(m.s3FilteredBuckets) > 0 {
		buckets = m.s3FilteredBuckets
	} else if m.vimState.LastSearch != "" {
		// Search is active but no results
		buckets = []aws.Bucket{}
	}

	if len(buckets) == 0 {
		if m.vimState.LastSearch != "" {
			return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No buckets match your search")
		}
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No buckets found")
	}

	// Ensure selected item is visible and get viewport range
	m.ensureVisible(m.s3SelectedIndex, len(buckets))
	start, end := m.getVisibleRange(len(buckets))

	// Build table header (k9s style)
	var content strings.Builder

	// Title with count - k9s style
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true) // Cyan
	searchInfo := ""
	if m.vimState.LastSearch != "" {
		searchInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render("(" + m.vimState.LastSearch + ")")
	}
	tableTitle := fmt.Sprintf("S3-Buckets%s[%d]", searchInfo, len(buckets))
	titleText := titleStyle.Render(tableTitle)

	// Center the title with dashes on both sides
	titleWidth := len(tableTitle)
	totalWidth := 100
	dashesWidth := (totalWidth - titleWidth - 2) / 2
	if dashesWidth < 1 {
		dashesWidth = 1
	}

	content.WriteString(strings.Repeat("─", dashesWidth) + " ")
	content.WriteString(titleText)
	content.WriteString(" " + strings.Repeat("─", dashesWidth) + "\n")

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
	content.WriteString(headerStyle.Render(fmt.Sprintf("%-40s %-25s %-20s",
		"BUCKET NAME", "CREATION DATE", "REGION")) + "\n")

	// Build table rows (only visible items)
	for i := start; i < end; i++ {
		bucket := buckets[i]
		creationDate := bucket.CreationDate
		if creationDate == "" {
			creationDate = "-"
		}

		region := bucket.Region
		if region == "" {
			region = "-"
		}

		// Highlight selected row
		row := fmt.Sprintf("%-40s %-25s %-20s",
			truncate(bucket.Name, 40),
			creationDate,
			region,
		)

		if i == m.s3SelectedIndex {
			// Highlight the selected row - k9s style with cyan background
			// Use ANSI codes directly to avoid lipgloss adding extra width
			// \x1b[K clears to end of line with background color
			row = "\x1b[48;5;51m\x1b[38;5;0m\x1b[1m" + row + "\x1b[K\x1b[0m"
		}

		content.WriteString(row + "\n")
	}

	// Show scroll position and total
	content.WriteString(fmt.Sprintf("\nShowing %d-%d of %d buckets", start+1, end, len(buckets)))

	return content.String()
}

func (m model) renderS3Browse() string {
	// Build breadcrumb
	breadcrumbStyle := lipgloss.NewStyle().Bold(true)
	bucketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	var breadcrumbs strings.Builder
	breadcrumbs.WriteString(bucketStyle.Render(m.s3CurrentBucket))

	if m.s3CurrentPrefix != "" {
		parts := strings.Split(strings.TrimSuffix(m.s3CurrentPrefix, "/"), "/")
		for _, part := range parts {
			if part != "" {
				breadcrumbs.WriteString(separatorStyle.Render(" > "))
				breadcrumbs.WriteString(part)
			}
		}
	}

	title := breadcrumbStyle.Render("S3 Browser: ") + breadcrumbs.String()
	if m.vimState.LastSearch != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(fmt.Sprintf(" [search: %s]", m.vimState.LastSearch))
	}

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading objects...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Use filtered objects if VIM search is active
	objects := m.s3Objects
	if len(m.s3FilteredObjects) > 0 {
		objects = m.s3FilteredObjects
	} else if m.vimState.LastSearch != "" {
		// Search is active but no results
		objects = []aws.S3Object{}
	}

	if len(objects) == 0 {
		if m.vimState.LastSearch != "" {
			return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No objects match your search")
		}
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No objects found (empty folder)")
	}

	// Ensure selected item is visible and get viewport range
	m.ensureVisible(m.s3ObjectSelectedIndex, len(objects))
	start, end := m.getVisibleRange(len(objects))

	// Build table header (k9s style)
	var content strings.Builder

	// Title with count - k9s style
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true) // Cyan
	searchInfo := ""
	if m.vimState.LastSearch != "" {
		searchInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render("(" + m.vimState.LastSearch + ")")
	}

	// Breadcrumb for location
	bucketPath := m.s3CurrentBucket
	if m.s3CurrentPrefix != "" {
		bucketPath += "/" + strings.TrimSuffix(m.s3CurrentPrefix, "/")
	}

	tableTitle := fmt.Sprintf("S3-Objects(%s)%s[%d]", bucketPath, searchInfo, len(objects))
	titleText := titleStyle.Render(tableTitle)

	// Center the title with dashes on both sides
	titleWidth := len(tableTitle)
	totalWidth := 100
	dashesWidth := (totalWidth - titleWidth - 2) / 2
	if dashesWidth < 1 {
		dashesWidth = 1
	}

	content.WriteString(strings.Repeat("─", dashesWidth) + " ")
	content.WriteString(titleText)
	content.WriteString(" " + strings.Repeat("─", dashesWidth) + "\n")

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
	content.WriteString(headerStyle.Render(fmt.Sprintf("%-6s %-50s %-15s %-25s %-20s",
		"TYPE", "NAME", "SIZE", "LAST MODIFIED", "STORAGE CLASS")) + "\n")

	// Build table rows (only visible items)
	for i := start; i < end; i++ {
		obj := objects[i]
		var typeIcon, name, size, lastModified, storageClass string

		if obj.IsFolder {
			typeIcon = "DIR"
			// Extract folder name from full key
			folderName := strings.TrimSuffix(obj.Key, "/")
			if m.s3CurrentPrefix != "" {
				folderName = strings.TrimPrefix(folderName, m.s3CurrentPrefix)
			}
			name = folderName + "/"
			size = "-"
			lastModified = "-"
			storageClass = "-"
		} else {
			typeIcon = "FILE"
			// Extract file name from full key
			fileName := obj.Key
			if m.s3CurrentPrefix != "" {
				fileName = strings.TrimPrefix(fileName, m.s3CurrentPrefix)
			}
			name = fileName
			size = formatBytes(obj.Size)
			lastModified = obj.LastModified
			storageClass = obj.StorageClass
		}

		// Style for folders
		if obj.IsFolder {
			typeIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render(typeIcon)
			name = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Render(name)
		}

		// Highlight selected row
		row := fmt.Sprintf("%-6s %-50s %-15s %-25s %-20s",
			typeIcon,
			truncate(name, 50),
			size,
			lastModified,
			truncate(storageClass, 20),
		)

		if i == m.s3ObjectSelectedIndex {
			// Highlight the selected row - k9s style with cyan background
			// Use ANSI codes directly to avoid lipgloss adding extra width
			// \x1b[K clears to end of line with background color
			row = "\x1b[48;5;51m\x1b[38;5;0m\x1b[1m" + row + "\x1b[K\x1b[0m"
		}

		content.WriteString(row + "\n")
	}

	// Pagination and scroll info
	content.WriteString(fmt.Sprintf("\nShowing %d-%d of %d objects", start+1, end, len(objects)))
	if m.s3IsTruncated {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(" (more available - press 'n' for next page)"))
	}

	return content.String()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (m model) renderS3ObjectDetails() string {
	title := lipgloss.NewStyle().Bold(true).Render("S3 Object Details")

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading object details...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.s3ObjectDetails == nil {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No object details available")
	}

	details := m.s3ObjectDetails
	var content strings.Builder
	content.WriteString(title + "\n\n")

	// Section styling
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle := lipgloss.NewStyle()

	// Basic Information
	content.WriteString(sectionStyle.Render("Object Information") + "\n")
	content.WriteString(labelStyle.Render("  Bucket:          ") + valueStyle.Render(m.s3CurrentBucket) + "\n")
	content.WriteString(labelStyle.Render("  Key:             ") + valueStyle.Render(details.Key) + "\n")
	content.WriteString(labelStyle.Render("  Size:            ") + valueStyle.Render(formatBytes(details.Size)) + "\n")
	if details.LastModified != "" {
		content.WriteString(labelStyle.Render("  Last Modified:   ") + valueStyle.Render(details.LastModified) + "\n")
	}
	content.WriteString(labelStyle.Render("  Storage Class:   ") + valueStyle.Render(details.StorageClass) + "\n")
	if details.ContentType != "" {
		content.WriteString(labelStyle.Render("  Content Type:    ") + valueStyle.Render(details.ContentType) + "\n")
	}
	if details.ETag != "" {
		content.WriteString(labelStyle.Render("  ETag:            ") + valueStyle.Render(details.ETag) + "\n")
	}
	content.WriteString("\n")

	// Metadata
	if len(details.Metadata) > 0 {
		content.WriteString(sectionStyle.Render("Metadata") + "\n")
		for key, value := range details.Metadata {
			content.WriteString(labelStyle.Render(fmt.Sprintf("  %s: ", key)) + valueStyle.Render(value) + "\n")
		}
		content.WriteString("\n")
	}

	// Tags
	if len(details.Tags) > 0 {
		content.WriteString(sectionStyle.Render("Tags") + "\n")
		for key, value := range details.Tags {
			content.WriteString(labelStyle.Render(fmt.Sprintf("  %s: ", key)) + valueStyle.Render(value) + "\n")
		}
		content.WriteString("\n")
	}

	// Actions hint
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	content.WriteString(hintStyle.Render("Press 'd' to download this file") + "\n")

	return m.renderWithViewport(content.String())
}

func (m model) renderEKS() string {
	title := lipgloss.NewStyle().Bold(true).Render("EKS Clusters")
	if m.vimState.LastSearch != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(fmt.Sprintf(" [search: %s]", m.vimState.LastSearch))
	}

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading clusters...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Use filtered clusters if VIM search is active, otherwise use all clusters
	clusters := m.eksClusters
	if len(m.eksFilteredClusters) > 0 {
		clusters = m.eksFilteredClusters
	} else if m.vimState.LastSearch != "" {
		// Search is active but no results
		clusters = []aws.EKSCluster{}
	}

	if len(clusters) == 0 {
		if m.vimState.LastSearch != "" {
			return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No clusters match your search")
		}
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No EKS clusters found")
	}

	// Ensure selected item is visible and get viewport range
	m.ensureVisible(m.eksSelectedIndex, len(clusters))
	start, end := m.getVisibleRange(len(clusters))

	// Build table header (k9s style)
	var content strings.Builder

	// Title with count - k9s style
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true) // Cyan
	searchInfo := ""
	if m.vimState.LastSearch != "" {
		searchInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Render("(" + m.vimState.LastSearch + ")")
	}
	tableTitle := fmt.Sprintf("EKS-Clusters%s[%d]", searchInfo, len(clusters))
	titleText := titleStyle.Render(tableTitle)

	// Center the title with dashes on both sides
	titleWidth := len(tableTitle)
	totalWidth := 100
	dashesWidth := (totalWidth - titleWidth - 2) / 2
	if dashesWidth < 1 {
		dashesWidth = 1
	}

	content.WriteString(strings.Repeat("─", dashesWidth) + " ")
	content.WriteString(titleText)
	content.WriteString(" " + strings.Repeat("─", dashesWidth) + "\n")

	// Table header - k9s uses uppercase and symbols
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
	content.WriteString(headerStyle.Render(fmt.Sprintf("%-30s %-15s %-15s %-10s %-30s",
		"CLUSTER NAME", "VERSION", "STATUS", "NODES", "REGION")) + "\n")

	// Build table rows (only visible items)
	for i := start; i < end; i++ {
		cluster := clusters[i]

		// Build row with proper spacing
		row := fmt.Sprintf("%-30s %-15s %-15s %-10d %-30s",
			truncate(cluster.Name, 30),
			cluster.Version,
			cluster.Status,
			cluster.NodeCount,
			cluster.Region,
		)

		if i == m.eksSelectedIndex {
			// Highlight the selected row - k9s style with cyan background
			for len(row) < 98 {
				row += " "
			}
			row = "\x1b[48;5;51m\x1b[38;5;0m\x1b[1m" + row + "\x1b[0m"
		}

		content.WriteString(row + "\n")
	}

	// Footer info
	content.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	content.WriteString(footerStyle.Render(fmt.Sprintf("Showing %d-%d of %d clusters", start+1, end, len(clusters))))

	return content.String()
}

func (m model) renderEKSDetails() string {
	title := lipgloss.NewStyle().Bold(true).Render("EKS Cluster Details")

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading cluster details...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.eksClusterDetails == nil {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("No cluster details available")
	}

	details := m.eksClusterDetails
	var content strings.Builder
	content.WriteString(title + "\n\n")

	// Section styling
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle := lipgloss.NewStyle()

	// Basic Information
	content.WriteString(sectionStyle.Render("Cluster Information") + "\n")
	content.WriteString(labelStyle.Render("  Name:            ") + valueStyle.Render(details.Name) + "\n")
	content.WriteString(labelStyle.Render("  Version:         ") + valueStyle.Render(details.Version) + "\n")
	content.WriteString(labelStyle.Render("  Status:          ") + getEKSStatusStyle(details.Status).Render(details.Status) + "\n")
	if details.Endpoint != "" {
		content.WriteString(labelStyle.Render("  Endpoint:        ") + valueStyle.Render(details.Endpoint) + "\n")
	}
	content.WriteString(labelStyle.Render("  Region:          ") + valueStyle.Render(details.Region) + "\n")
	if details.CreatedAt != "" {
		content.WriteString(labelStyle.Render("  Created:         ") + valueStyle.Render(details.CreatedAt) + "\n")
	}
	if details.PlatformVersion != "" {
		content.WriteString(labelStyle.Render("  Platform:        ") + valueStyle.Render(details.PlatformVersion) + "\n")
	}
	content.WriteString("\n")

	// IAM and Security
	content.WriteString(sectionStyle.Render("IAM & Security") + "\n")
	if details.RoleArn != "" {
		content.WriteString(labelStyle.Render("  Role ARN:        ") + valueStyle.Render(details.RoleArn) + "\n")
	}
	content.WriteString("\n")

	// Network Configuration
	content.WriteString(sectionStyle.Render("Network Configuration") + "\n")
	if details.VpcId != "" {
		content.WriteString(labelStyle.Render("  VPC ID:          ") + valueStyle.Render(details.VpcId) + "\n")
	}
	if len(details.SubnetIds) > 0 {
		content.WriteString(labelStyle.Render("  Subnets:         ") + valueStyle.Render(fmt.Sprintf("%d configured", len(details.SubnetIds))) + "\n")
		for i, subnet := range details.SubnetIds {
			if i < 3 { // Show first 3
				content.WriteString(labelStyle.Render("                   ") + valueStyle.Render(subnet) + "\n")
			} else if i == 3 {
				content.WriteString(labelStyle.Render("                   ") + valueStyle.Render(fmt.Sprintf("... and %d more", len(details.SubnetIds)-3)) + "\n")
				break
			}
		}
	}
	if len(details.SecurityGroupIds) > 0 {
		content.WriteString(labelStyle.Render("  Security Groups: ") + valueStyle.Render(fmt.Sprintf("%d configured", len(details.SecurityGroupIds))) + "\n")
		for i, sg := range details.SecurityGroupIds {
			if i < 3 { // Show first 3
				content.WriteString(labelStyle.Render("                   ") + valueStyle.Render(sg) + "\n")
			} else if i == 3 {
				content.WriteString(labelStyle.Render("                   ") + valueStyle.Render(fmt.Sprintf("... and %d more", len(details.SecurityGroupIds)-3)) + "\n")
				break
			}
		}
	}
	content.WriteString("\n")

	// Logging
	if len(details.EnabledLogTypes) > 0 {
		content.WriteString(sectionStyle.Render("Logging") + "\n")
		content.WriteString(labelStyle.Render("  Enabled Types:   ") + valueStyle.Render(strings.Join(details.EnabledLogTypes, ", ")) + "\n")
		content.WriteString("\n")
	}

	// Node Groups
	if len(m.eksNodeGroups) > 0 {
		content.WriteString(sectionStyle.Render(fmt.Sprintf("Node Groups (%d)", len(m.eksNodeGroups))) + "\n")

		// Table header
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
		content.WriteString("  " + headerStyle.Render(fmt.Sprintf("%-25s %-15s %-20s %-15s",
			"NAME", "STATUS", "INSTANCE TYPES", "SIZE (D/Min/Max)")) + "\n")

		for _, ng := range m.eksNodeGroups {
			instanceTypes := strings.Join(ng.InstanceTypes, ", ")
			if len(instanceTypes) > 20 {
				instanceTypes = instanceTypes[:17] + "..."
			}
			sizeInfo := fmt.Sprintf("%d/%d/%d", ng.DesiredSize, ng.MinSize, ng.MaxSize)

			content.WriteString("  " + fmt.Sprintf("%-25s %-15s %-20s %-15s",
				truncate(ng.Name, 25),
				ng.Status,
				instanceTypes,
				sizeInfo,
			) + "\n")
		}
		content.WriteString("\n")
	}

	// Add-ons
	if len(m.eksAddons) > 0 {
		content.WriteString(sectionStyle.Render(fmt.Sprintf("Add-ons (%d)", len(m.eksAddons))) + "\n")

		// Table header
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true)
		content.WriteString("  " + headerStyle.Render(fmt.Sprintf("%-25s %-15s %-15s",
			"NAME", "VERSION", "STATUS")) + "\n")

		for _, addon := range m.eksAddons {
			content.WriteString("  " + fmt.Sprintf("%-25s %-15s %-15s",
				truncate(addon.Name, 25),
				truncate(addon.Version, 15),
				addon.Status,
			) + "\n")
		}
		content.WriteString("\n")
	}

	// Tags
	if len(details.Tags) > 0 {
		content.WriteString(sectionStyle.Render(fmt.Sprintf("Tags (%d)", len(details.Tags))) + "\n")
		count := 0
		for key, value := range details.Tags {
			if count < 10 { // Show first 10 tags
				content.WriteString(labelStyle.Render(fmt.Sprintf("  %-20s ", key)) + valueStyle.Render(value) + "\n")
				count++
			} else {
				content.WriteString(labelStyle.Render(fmt.Sprintf("  ... and %d more tags", len(details.Tags)-10)) + "\n")
				break
			}
		}
	}

	return m.renderWithViewport(content.String())
}

func getEKSStatusStyle(status string) lipgloss.Style {
	switch strings.ToUpper(status) {
	case "ACTIVE":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
	case "CREATING", "UPDATING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
	case "DELETING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	case "FAILED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	}
}

func getStateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
	case "terminated", "terminating":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	case "pending", "stopping":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // Blue
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v", err)
		os.Exit(1)
	}

	// Main loop: run the TUI, and if SSM session is requested, run it and restart
	for {
		p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}

		// Check if we should launch an SSM session
		m, ok := finalModel.(model)
		if !ok || m.ssmInstanceID == "" {
			// Normal exit, no SSM session to launch
			break
		}

		// Launch SSM session in the current terminal
		fmt.Printf("Connecting to instance %s via SSM...\n", m.ssmInstanceID)

		// Create the SSM command
		// Let the session-manager-plugin handle terminal mode itself
		ssmCmd := exec.Command("aws", "ssm", "start-session", "--target", m.ssmInstanceID, "--region", m.ssmRegion)
		ssmCmd.Stdin = os.Stdin
		ssmCmd.Stdout = os.Stdout
		ssmCmd.Stderr = os.Stderr

		// Run the command - let it handle signals naturally
		err = ssmCmd.Run()

		if err != nil {
			// Only show error if it's not a normal exit
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Exit code 130 means Ctrl+C was pressed - this is normal
				if exitErr.ExitCode() != 130 {
					fmt.Printf("\nSSM session ended with code %d\n", exitErr.ExitCode())
				}
			} else {
				fmt.Printf("\nSSM session error: %v\n", err)
				fmt.Println("Press Enter to return to lazyaws...")
				fmt.Scanln()
			}
		}

		fmt.Println("\nReturning to lazyaws...")
		// Loop continues and restarts the TUI
	}
}
