package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fuziontech/lazyaws/internal/aws"
	"github.com/fuziontech/lazyaws/internal/config"
)

type screen int

const (
	ec2Screen screen = iota
	ec2DetailsScreen
	s3Screen
	eksScreen
)

type model struct {
	currentScreen       screen
	width               int
	height              int
	awsClient           *aws.Client
	ec2Instances        []aws.Instance
	ec2SelectedIndex    int
	ec2InstanceDetails  *aws.InstanceDetails
	ec2InstanceStatus   *aws.InstanceStatus
	ec2InstanceMetrics  *aws.InstanceMetrics
	ec2SSMStatus        *aws.SSMConnectionStatus
	loading             bool
	err                 error
	config              *config.Config
	filterInput         textinput.Model
	filtering           bool
	filter              string
	confirmAction       string
	confirmInstanceID   string
	showingConfirm      bool
	statusMessage       string
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

func initialModel(cfg *config.Config) model {
	ti := textinput.New()
	ti.Placeholder = "<name>, <id>, state=<state> or tag:key=value"
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20

	return model{
		currentScreen: ec2Screen,
		loading:       true,
		config:        cfg,
		filterInput:   ti,
		filtering:     false,
	}
}

func (m model) Init() tea.Cmd {
	return m.initAWSClient
}

func (m model) initAWSClient() tea.Msg {
	ctx := context.Background()
	client, err := aws.NewClient(ctx, m.config)
	if err != nil {
		return instancesLoadedMsg{err: err}
	}
	return client
}

func (m model) loadEC2Instances() tea.Msg {
	ctx := context.Background()
	instances, err := m.awsClient.ListInstances(ctx)
	return instancesLoadedMsg{instances: instances, err: err}
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle confirmation dialog
	if m.showingConfirm {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				m.loading = true
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

	// Handle filtering
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Don't quit if we're in details view, go back instead
			if m.currentScreen == ec2DetailsScreen {
				m.currentScreen = ec2Screen
				m.ec2InstanceDetails = nil
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			// ESC key to go back from details view
			if m.currentScreen == ec2DetailsScreen {
				m.currentScreen = ec2Screen
				m.ec2InstanceDetails = nil
				return m, nil
			}
		case "1":
			m.currentScreen = ec2Screen
		case "2":
			m.currentScreen = s3Screen
		case "3":
			m.currentScreen = eksScreen
		case "up", "k":
			// Navigate up in EC2 instance list
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				if m.ec2SelectedIndex > 0 {
					m.ec2SelectedIndex--
				}
			}
		case "down", "j":
			// Navigate down in EC2 instance list
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				if m.ec2SelectedIndex < len(m.ec2Instances)-1 {
					m.ec2SelectedIndex++
				}
			}
		case "enter":
			// Enter key to view instance details
			if m.currentScreen == ec2Screen && len(m.ec2Instances) > 0 {
				selectedInstance := m.ec2Instances[m.ec2SelectedIndex]
				m.loading = true
				return m, m.loadEC2InstanceDetails(selectedInstance.ID)
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
			if m.currentScreen == ec2Screen {
				m.currentScreen = s3Screen
			} else if m.currentScreen == s3Screen {
				m.currentScreen = eksScreen
			} else if m.currentScreen == eksScreen {
				m.currentScreen = ec2Screen
			}
		case "r":
			// Refresh current view
			if m.currentScreen == ec2Screen {
				m.loading = true
				return m, m.loadEC2Instances
			}
		case "f":
			// Only filter on EC2 list screen
			if m.currentScreen == ec2Screen {
				m.filtering = true
				m.filterInput.Focus()
				return m, nil
			}
		case "s":
			// Start instance (works in list or details view)
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
			// Stop instance (works in list or details view)
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
			// Reboot instance (works in list or details view)
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
			// Terminate instance (works in list or details view)
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
				err := m.awsClient.LaunchSSMSession(m.ec2InstanceDetails.ID, m.awsClient.GetRegion())
				if err != nil {
					m.statusMessage = fmt.Sprintf("Failed to launch SSM session: %v", err)
				} else {
					m.statusMessage = "Launching SSM session in new terminal..."
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m model) View() string {
	var s string

	// Header with tabs and region info
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("2")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	ec2Tab := inactiveTabStyle.Render("1. EC2")
	s3Tab := inactiveTabStyle.Render("2. S3")
	eksTab := inactiveTabStyle.Render("3. EKS")

	switch m.currentScreen {
	case ec2Screen, ec2DetailsScreen:
		ec2Tab = activeTabStyle.Render("1. EC2")
	case s3Screen:
		s3Tab = activeTabStyle.Render("2. S3")
	case eksScreen:
		eksTab = activeTabStyle.Render("3. EKS")
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Top, ec2Tab, "  ", s3Tab, "  ", eksTab)

	// Add region info
	regionInfo := ""
	if m.awsClient != nil {
		regionInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render(fmt.Sprintf("  [Region: %s]", m.awsClient.GetRegion()))
	}

	s += tabs + regionInfo + "\n\n"

	// Content area
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2)

	var content string
	switch m.currentScreen {
	case ec2Screen:
		content = m.renderEC2()
	case ec2DetailsScreen:
		content = m.renderEC2Details()
	case s3Screen:
		content = m.renderS3()
	case eksScreen:
		content = m.renderEKS()
	}

	if m.filtering {
		s += "\n" + m.filterInput.View()
	}

	s += contentStyle.Render(content) + "\n"

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

	// Show status message
	if m.statusMessage != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		s += "\n" + statusStyle.Render(m.statusMessage)
	}

	// Footer
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	var helpText string
	if m.currentScreen == ec2DetailsScreen {
		// Show SSM connect option if SSM is connected
		if m.ec2SSMStatus != nil && m.ec2SSMStatus.Connected {
			helpText = "s:Start | S:Stop | R:Reboot | t:Terminate | C:SSM Connect | ESC/q: Back"
		} else {
			helpText = "s:Start | S:Stop | R:Reboot | t:Terminate | ESC/q: Back | 1/2/3: Switch Service"
		}
	} else if m.currentScreen == ec2Screen {
		helpText = "↑↓/jk: Navigate | Enter: Details | s:Start | S:Stop | R:Reboot | t:Terminate | f: Filter | q: Quit"
	} else {
		helpText = "Tab: Next | 1/2/3: Switch | c: Change Region | r: Refresh | q: Quit"
	}
	s += "\n" + helpStyle.Render(helpText)

	return s
}

func (m model) renderEC2() string {
	title := lipgloss.NewStyle().Bold(true).Render("EC2 Instances")
	if m.filter != "" {
		title += lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf(" (filtered by: %s)", m.filter))
	}

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading instances...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Filter instances
	var filteredInstances []aws.Instance
	if m.filter == "" {
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

	// Build table header
	var content strings.Builder
	content.WriteString(title + "\n\n")

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	content.WriteString(headerStyle.Render(fmt.Sprintf("%-20s %-30s %-15s %-15s %-15s\n",
		"INSTANCE ID", "NAME", "STATE", "TYPE", "IP")))
	content.WriteString(strings.Repeat("─", 100) + "\n")

	// Build table rows
	for i, inst := range filteredInstances {
		stateStyle := getStateStyle(inst.State)
		name := inst.Name
		if name == "" {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("-")
		}

		ip := inst.PublicIP
		if ip == "" {
			ip = inst.PrivateIP
		}
		if ip == "" {
			ip = "-"
		}

		// Highlight selected row
		row := fmt.Sprintf("%-20s %-30s %-15s %-15s %-15s",
			inst.ID,
			truncate(name, 30),
			stateStyle.Render(inst.State),
			inst.InstanceType,
			ip,
		)

		if i == m.ec2SelectedIndex {
			// Highlight the selected row
			selectedStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("15"))
			row = selectedStyle.Render(row)
		}

		content.WriteString(row + "\n")
	}

	content.WriteString(fmt.Sprintf("\nTotal: %d instances", len(filteredInstances)))

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

	return content.String()
}

func (m model) renderS3() string {
	return lipgloss.NewStyle().Bold(true).Render("S3 Buckets") + "\n\n" +
		"Coming soon:\n" +
		"  • List buckets\n" +
		"  • Browse objects\n" +
		"  • Upload/Download\n" +
		"  • Bucket management"
}

func (m model) renderEKS() string {
	return lipgloss.NewStyle().Bold(true).Render("EKS Clusters") + "\n\n" +
		"Coming soon:\n" +
		"  • List clusters\n" +
		"  • Configure kubectl\n" +
		"  • Node group info\n" +
		"  • Cluster details"
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

	p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
