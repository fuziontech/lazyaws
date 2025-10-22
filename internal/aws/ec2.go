package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Instance represents an EC2 instance with relevant information
type Instance struct {
	ID           string
	Name         string
	State        string
	InstanceType string
	AZ           string
	PublicIP     string
	PrivateIP    string
}

// ListInstances retrieves all EC2 instances
func (c *Client) ListInstances(ctx context.Context) ([]Instance, error) {
	input := &ec2.DescribeInstancesInput{}
	result, err := c.EC2.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	var instances []Instance
	for _, reservation := range result.Reservations {
		for _, inst := range reservation.Instances {
			instance := Instance{
				ID:           getString(inst.InstanceId),
				State:        string(inst.State.Name),
				InstanceType: string(inst.InstanceType),
				AZ:           getString(inst.Placement.AvailabilityZone),
				PublicIP:     getString(inst.PublicIpAddress),
				PrivateIP:    getString(inst.PrivateIpAddress),
			}

			// Extract Name tag
			instance.Name = getNameTag(inst.Tags)

			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// getNameTag extracts the Name tag from EC2 tags
func getNameTag(tags []types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}

// getString safely dereferences a string pointer
func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
