package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMConnectionStatus represents the SSM connectivity status of an instance
type SSMConnectionStatus struct {
	InstanceID  string
	Connected   bool
	PingStatus  string
	AgentVersion string
	PlatformType string
	PlatformName string
	LastPingTime string
}

// CheckSSMConnectivity checks if an instance is reachable via SSM
func (c *Client) CheckSSMConnectivity(ctx context.Context, instanceID string) (*SSMConnectionStatus, error) {
	input := &ssm.DescribeInstanceInformationInput{
		Filters: []types.InstanceInformationStringFilter{
			{
				Key:    getString(&[]string{"InstanceIds"}[0]),
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
