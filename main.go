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
	config        *config.Config
	filterInput   textinput.Model
	filtering     bool
	filter        string
}

type instancesLoadedMsg struct {
	instances []aws.Instance
	err       error
}

func initialModel(cfg *config.Config) model {
	ti := textinput.New()
	ti.Placeholder = "tag:key=value"
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.currentScreen = (m.currentScreen + 1) % 3
		case "r":
			// Refresh current view
			if m.currentScreen == ec2Screen {
				m.loading = true
				return m, m.loadEC2Instances
			}
		case "f":
			m.filtering = true
			m.filterInput.Focus()
			return m, nil
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

	if m.filtering {
		s += "\n" + m.filterInput.View()
	}

	s += contentStyle.Render(content) + "\n"

	// Footer
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s += "\n" + helpStyle.Render("Tab: Next | 1/2/3: Switch | c: Change Region | r: Refresh | f: Filter | q: Quit")

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
				if strings.Contains(strings.ToLower(inst.State), strings.ToLower(m.filter)) {
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
	for _, inst := range filteredInstances {
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

	content.WriteString(fmt.Sprintf("\nTotal: %d instances", len(filteredInstances)))

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
