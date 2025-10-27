package aws

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMConnectionStatus represents the SSM connectivity status of an instance
type SSMConnectionStatus struct {
	InstanceID   string
	Connected    bool
	PingStatus   string
	AgentVersion string
	PlatformType string
	PlatformName string
	LastPingTime string
}

// CheckSSMConnectivity checks if an instance is reachable via SSM
func (c *Client) CheckSSMConnectivity(ctx context.Context, instanceID string) (*SSMConnectionStatus, error) {
	filterKey := "InstanceIds"
	input := &ssm.DescribeInstanceInformationInput{
		Filters: []types.InstanceInformationStringFilter{
			{
				Key:    &filterKey,
				Values: []string{instanceID},
			},
		},
	}

	result, err := c.SSM.DescribeInstanceInformation(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance information: %w", err)
	}

	status := &SSMConnectionStatus{
		InstanceID: instanceID,
		Connected:  false,
	}

	if len(result.InstanceInformationList) == 0 {
		return status, nil // Not connected to SSM
	}

	info := result.InstanceInformationList[0]
	status.Connected = true
	status.PingStatus = string(info.PingStatus)
	status.AgentVersion = getString(info.AgentVersion)
	status.PlatformType = string(info.PlatformType)
	status.PlatformName = getString(info.PlatformName)

	if info.LastPingDateTime != nil {
		status.LastPingTime = info.LastPingDateTime.Format("2006-01-02 15:04:05")
	}

	return status, nil
}

// LaunchSSMSession opens a new terminal window and starts an SSM session
func (c *Client) LaunchSSMSession(instanceID string, region string) error {
	// Detect terminal emulator
	terminal := detectTerminal()
	if terminal == "" {
		return fmt.Errorf("could not detect terminal emulator")
	}

	// Build the AWS SSM command
	ssmCommand := fmt.Sprintf("aws ssm start-session --target %s --region %s", instanceID, region)

	// Build terminal command based on detected terminal
	var cmd *exec.Cmd
	switch terminal {
	case "ghostty":
		// Ghostty uses a similar syntax to alacritty/kitty
		cmd = exec.Command("ghostty", "-e", "bash", "-c", ssmCommand+"; exec bash")
	case "gnome-terminal":
		cmd = exec.Command("gnome-terminal", "--", "bash", "-c", ssmCommand+"; exec bash")
	case "xterm":
		cmd = exec.Command("xterm", "-e", "bash -c '"+ssmCommand+"; exec bash'")
	case "konsole":
		cmd = exec.Command("konsole", "-e", "bash", "-c", ssmCommand+"; exec bash")
	case "xfce4-terminal":
		cmd = exec.Command("xfce4-terminal", "-e", "bash -c '"+ssmCommand+"; exec bash'")
	case "alacritty":
		cmd = exec.Command("alacritty", "-e", "bash", "-c", ssmCommand+"; exec bash")
	case "kitty":
		cmd = exec.Command("kitty", "bash", "-c", ssmCommand+"; exec bash")
	case "terminator":
		cmd = exec.Command("terminator", "-e", "bash -c '"+ssmCommand+"; exec bash'")
	default:
		return fmt.Errorf("unsupported terminal: %s", terminal)
	}

	// Start the terminal in the background
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to launch terminal: %w", err)
	}

	return nil
}

// detectTerminal detects the user's terminal emulator
func detectTerminal() string {
	// Check for Ghostty first using TERM_PROGRAM environment variable
	if os.Getenv("TERM_PROGRAM") == "ghostty" {
		return "ghostty"
	}

	// Check common terminal emulators
	terminals := []string{
		"ghostty",
		"gnome-terminal",
		"konsole",
		"xfce4-terminal",
		"xterm",
		"alacritty",
		"kitty",
		"terminator",
	}

	for _, term := range terminals {
		if _, err := exec.LookPath(term); err == nil {
			return term
		}
	}

	// Check TERM environment variable as fallback
	if term := os.Getenv("TERM"); term != "" {
		return term
	}

	return ""
}

// StartPortForward starts an SSM port forwarding session
func (c *Client) StartPortForward(instanceID string, region string, localPort int, remotePort int, remoteHost string) error {
	// Detect terminal emulator
	terminal := detectTerminal()
	if terminal == "" {
		return fmt.Errorf("could not detect terminal emulator")
	}

	// Build the AWS SSM port forward command
	var ssmCommand string
	if remoteHost == "" || remoteHost == "localhost" {
		// Standard port forwarding
		ssmCommand = fmt.Sprintf("aws ssm start-session --target %s --region %s --document-name AWS-StartPortForwardingSession --parameters 'localPortNumber=%d,portNumber=%d'",
			instanceID, region, localPort, remotePort)
	} else {
		// Port forwarding to a remote host
		ssmCommand = fmt.Sprintf("aws ssm start-session --target %s --region %s --document-name AWS-StartPortForwardingSessionToRemoteHost --parameters 'host=%s,localPortNumber=%d,portNumber=%d'",
			instanceID, region, remoteHost, localPort, remotePort)
	}

	// Build terminal command based on detected terminal
	var cmd *exec.Cmd
	switch terminal {
	case "ghostty":
		cmd = exec.Command("ghostty", "-e", "bash", "-c", ssmCommand+"; echo 'Press Enter to close'; read")
	case "gnome-terminal":
		cmd = exec.Command("gnome-terminal", "--", "bash", "-c", ssmCommand+"; echo 'Press Enter to close'; read")
	case "xterm":
		cmd = exec.Command("xterm", "-e", "bash -c '"+ssmCommand+"; echo Press Enter to close; read'")
	case "konsole":
		cmd = exec.Command("konsole", "-e", "bash", "-c", ssmCommand+"; echo 'Press Enter to close'; read")
	case "xfce4-terminal":
		cmd = exec.Command("xfce4-terminal", "-e", "bash -c '"+ssmCommand+"; echo Press Enter to close; read'")
	case "alacritty":
		cmd = exec.Command("alacritty", "-e", "bash", "-c", ssmCommand+"; echo 'Press Enter to close'; read")
	case "kitty":
		cmd = exec.Command("kitty", "bash", "-c", ssmCommand+"; echo 'Press Enter to close'; read")
	case "terminator":
		cmd = exec.Command("terminator", "-e", "bash -c '"+ssmCommand+"; echo Press Enter to close; read'")
	default:
		return fmt.Errorf("unsupported terminal: %s", terminal)
	}

	// Start the terminal in the background
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to launch terminal: %w", err)
	}

	return nil
}
