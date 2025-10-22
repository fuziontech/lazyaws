package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fuziontech/lazyaws/internal/aws"
)

type screen int

const (
	ec2Screen screen = iota
	s3Screen
	eksScreen
)

type model struct {
	currentScreen screen
	width         int
	height        int
	awsClient     *aws.Client
	ec2Instances  []aws.Instance
	loading       bool
	err           error
}

type instancesLoadedMsg struct {
	instances []aws.Instance
	err       error
}

func initialModel() model {
	return model{
		currentScreen: ec2Screen,
		loading:       true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		initAWSClient,
		loadEC2Instances,
	)
}

func initAWSClient() tea.Msg {
	ctx := context.Background()
	client, err := aws.NewClient(ctx)
	if err != nil {
		return instancesLoadedMsg{err: err}
	}
	return client
}

func loadEC2Instances() tea.Msg {
	ctx := context.Background()
	client, err := aws.NewClient(ctx)
	if err != nil {
		return instancesLoadedMsg{err: err}
	}

	instances, err := client.ListInstances(ctx)
	return instancesLoadedMsg{instances: instances, err: err}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *aws.Client:
		m.awsClient = msg
		return m, nil

	case instancesLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.ec2Instances = msg.instances
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.currentScreen = ec2Screen
		case "2":
			m.currentScreen = s3Screen
		case "3":
			m.currentScreen = eksScreen
		case "tab":
			m.currentScreen = (m.currentScreen + 1) % 3
		case "r":
			// Refresh current view
			if m.currentScreen == ec2Screen {
				m.loading = true
				return m, loadEC2Instances
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
	case ec2Screen:
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
	case s3Screen:
		content = m.renderS3()
	case eksScreen:
		content = m.renderEKS()
	}

	s += contentStyle.Render(content) + "\n"

	// Footer
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s += "\n" + helpStyle.Render("Tab: Next | 1/2/3: Switch | r: Refresh | q: Quit")

	return s
}

func (m model) renderEC2() string {
	title := lipgloss.NewStyle().Bold(true).Render("EC2 Instances")

	if m.loading {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Loading instances...")
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		return title + "\n\n" + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if len(m.ec2Instances) == 0 {
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
	for _, inst := range m.ec2Instances {
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

		content.WriteString(fmt.Sprintf("%-20s %-30s %-15s %-15s %-15s\n",
			inst.ID,
			truncate(name, 30),
			stateStyle.Render(inst.State),
			inst.InstanceType,
			ip,
		))
	}

	content.WriteString(fmt.Sprintf("\nTotal: %d instances", len(m.ec2Instances)))

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
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
