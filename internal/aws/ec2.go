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
	Tags         []Tag
}

// Tag represents a key-value pair for an AWS tag
type Tag struct {
	Key   string
	Value string
}

// InstanceDetails contains comprehensive EC2 instance information
type InstanceDetails struct {
	Instance                // Embed basic instance info
	LaunchTime       string
	VpcID            string
	SubnetID         string
	KeyName          string
	Architecture     string
	Platform         string
	RootDeviceType   string
	Monitoring       string
	IamInstanceProfile string
	SecurityGroups   []SecurityGroup
	BlockDevices     []BlockDevice
	NetworkInterfaces []NetworkInterface
	InstanceTypeInfo *InstanceTypeInfo
}

// InstanceTypeInfo contains specifications for an instance type
type InstanceTypeInfo struct {
	InstanceType         string
	VCpus                int32
	Memory               int64 // in MiB
	NetworkPerformance   string
	StorageType          string
	EbsOptimized         bool
	InstanceStorageGB    int64
	SupportedArchitectures []string
}

// SecurityGroup represents a security group attached to an instance
type SecurityGroup struct {
	ID   string
	Name string
}

// BlockDevice represents an attached volume
type BlockDevice struct {
	DeviceName string
	VolumeID   string
	VolumeSize int32
	VolumeType string
	DeleteOnTermination bool
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	ID               string
	PrivateIP        string
	PublicIP         string
	SubnetID         string
	VpcID            string
	MacAddress       string
	SecurityGroups   []SecurityGroup
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

			// Extract all tags
			for _, tag := range inst.Tags {
				instance.Tags = append(instance.Tags, Tag{Key: getString(tag.Key), Value: getString(tag.Value)})
			}

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

// GetInstanceDetails retrieves detailed information for a specific EC2 instance
func (c *Client) GetInstanceDetails(ctx context.Context, instanceID string) (*InstanceDetails, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	result, err := c.EC2.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	inst := result.Reservations[0].Instances[0]

	// Build basic instance info
	details := &InstanceDetails{
		Instance: Instance{
			ID:           getString(inst.InstanceId),
			State:        string(inst.State.Name),
			InstanceType: string(inst.InstanceType),
			AZ:           getString(inst.Placement.AvailabilityZone),
			PublicIP:     getString(inst.PublicIpAddress),
			PrivateIP:    getString(inst.PrivateIpAddress),
			Name:         getNameTag(inst.Tags),
		},
		VpcID:          getString(inst.VpcId),
		SubnetID:       getString(inst.SubnetId),
		KeyName:        getString(inst.KeyName),
		Architecture:   string(inst.Architecture),
		Platform:       getString(inst.PlatformDetails),
		RootDeviceType: string(inst.RootDeviceType),
	}

	// Launch time
	if inst.LaunchTime != nil {
		details.LaunchTime = inst.LaunchTime.Format("2006-01-02 15:04:05")
	}

	// Monitoring
	if inst.Monitoring != nil {
		details.Monitoring = string(inst.Monitoring.State)
	}

	// IAM Instance Profile
	if inst.IamInstanceProfile != nil {
		details.IamInstanceProfile = getString(inst.IamInstanceProfile.Arn)
	}

	// Extract all tags
	for _, tag := range inst.Tags {
		details.Instance.Tags = append(details.Instance.Tags, Tag{
			Key:   getString(tag.Key),
			Value: getString(tag.Value),
		})
	}

	// Security Groups
	for _, sg := range inst.SecurityGroups {
		details.SecurityGroups = append(details.SecurityGroups, SecurityGroup{
			ID:   getString(sg.GroupId),
			Name: getString(sg.GroupName),
		})
	}

	// Block Devices
	for _, bd := range inst.BlockDeviceMappings {
		device := BlockDevice{
			DeviceName: getString(bd.DeviceName),
		}
		if bd.Ebs != nil {
			device.VolumeID = getString(bd.Ebs.VolumeId)
			device.DeleteOnTermination = bd.Ebs.DeleteOnTermination != nil && *bd.Ebs.DeleteOnTermination

			// Get volume details for size and type
			if device.VolumeID != "" {
				volInput := &ec2.DescribeVolumesInput{
					VolumeIds: []string{device.VolumeID},
				}
				volResult, err := c.EC2.DescribeVolumes(ctx, volInput)
				if err == nil && len(volResult.Volumes) > 0 {
					vol := volResult.Volumes[0]
					if vol.Size != nil {
						device.VolumeSize = *vol.Size
					}
					device.VolumeType = string(vol.VolumeType)
				}
			}
		}
		details.BlockDevices = append(details.BlockDevices, device)
	}

	// Network Interfaces
	for _, ni := range inst.NetworkInterfaces {
		iface := NetworkInterface{
			ID:         getString(ni.NetworkInterfaceId),
			PrivateIP:  getString(ni.PrivateIpAddress),
			SubnetID:   getString(ni.SubnetId),
			VpcID:      getString(ni.VpcId),
			MacAddress: getString(ni.MacAddress),
		}

		if ni.Association != nil {
			iface.PublicIP = getString(ni.Association.PublicIp)
		}

		// Security Groups for this network interface
		for _, sg := range ni.Groups {
			iface.SecurityGroups = append(iface.SecurityGroups, SecurityGroup{
				ID:   getString(sg.GroupId),
				Name: getString(sg.GroupName),
			})
		}

		details.NetworkInterfaces = append(details.NetworkInterfaces, iface)
	}

	// Get Instance Type Information
	typeInfo, err := c.GetInstanceTypeInfo(ctx, details.InstanceType)
	if err == nil {
		details.InstanceTypeInfo = typeInfo
	}
	// If error fetching type info, just continue without it

	return details, nil
}

// StartInstance starts a stopped EC2 instance
func (c *Client) StartInstance(ctx context.Context, instanceID string) error {
	input := &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.EC2.StartInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return nil
}

// StopInstance stops a running EC2 instance
func (c *Client) StopInstance(ctx context.Context, instanceID string) error {
	input := &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.EC2.StopInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// RebootInstance reboots an EC2 instance
func (c *Client) RebootInstance(ctx context.Context, instanceID string) error {
	input := &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.EC2.RebootInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to reboot instance: %w", err)
	}

	return nil
}

// TerminateInstance terminates an EC2 instance
func (c *Client) TerminateInstance(ctx context.Context, instanceID string) error {
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.EC2.TerminateInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

// InstanceStatus represents the health status of an EC2 instance
type InstanceStatus struct {
	InstanceID         string
	InstanceState      string
	SystemStatus       string
	InstanceStatus     string
	SystemStatusOk     bool
	InstanceStatusOk   bool
	ScheduledEvents    []ScheduledEvent
}

// ScheduledEvent represents a scheduled maintenance event
type ScheduledEvent struct {
	Code        string
	Description string
	NotBefore   string
	NotAfter    string
}

// GetInstanceStatus retrieves the health status of an EC2 instance
func (c *Client) GetInstanceStatus(ctx context.Context, instanceID string) (*InstanceStatus, error) {
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds:         []string{instanceID},
		IncludeAllInstances: &[]bool{true}[0], // Include stopped instances
	}

	result, err := c.EC2.DescribeInstanceStatus(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance status: %w", err)
	}

	if len(result.InstanceStatuses) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	status := result.InstanceStatuses[0]

	instanceStatus := &InstanceStatus{
		InstanceID:    getString(status.InstanceId),
		InstanceState: string(status.InstanceState.Name),
	}

	// System status
	if status.SystemStatus != nil && status.SystemStatus.Status != "" {
		instanceStatus.SystemStatus = string(status.SystemStatus.Status)
		instanceStatus.SystemStatusOk = (string(status.SystemStatus.Status) == "ok")
	}

	// Instance status
	if status.InstanceStatus != nil && status.InstanceStatus.Status != "" {
		instanceStatus.InstanceStatus = string(status.InstanceStatus.Status)
		instanceStatus.InstanceStatusOk = (string(status.InstanceStatus.Status) == "ok")
	}

	// Scheduled events
	for _, event := range status.Events {
		scheduledEvent := ScheduledEvent{
			Code:        string(event.Code),
			Description: getString(event.Description),
		}
		if event.NotBefore != nil {
			scheduledEvent.NotBefore = event.NotBefore.Format("2006-01-02 15:04:05")
		}
		if event.NotAfter != nil {
			scheduledEvent.NotAfter = event.NotAfter.Format("2006-01-02 15:04:05")
		}
		instanceStatus.ScheduledEvents = append(instanceStatus.ScheduledEvents, scheduledEvent)
	}

	return instanceStatus, nil
}

// GetInstanceTypeInfo retrieves detailed specifications for an instance type
func (c *Client) GetInstanceTypeInfo(ctx context.Context, instanceType string) (*InstanceTypeInfo, error) {
	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []types.InstanceType{types.InstanceType(instanceType)},
	}

	result, err := c.EC2.DescribeInstanceTypes(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance type: %w", err)
	}

	if len(result.InstanceTypes) == 0 {
		return nil, fmt.Errorf("instance type %s not found", instanceType)
	}

	typeInfo := result.InstanceTypes[0]

	info := &InstanceTypeInfo{
		InstanceType: string(typeInfo.InstanceType),
	}

	// VCPUs
	if typeInfo.VCpuInfo != nil && typeInfo.VCpuInfo.DefaultVCpus != nil {
		info.VCpus = *typeInfo.VCpuInfo.DefaultVCpus
	}

	// Memory
	if typeInfo.MemoryInfo != nil && typeInfo.MemoryInfo.SizeInMiB != nil {
		info.Memory = *typeInfo.MemoryInfo.SizeInMiB
	}

	// Network Performance
	if typeInfo.NetworkInfo != nil {
		if typeInfo.NetworkInfo.NetworkPerformance != nil {
			info.NetworkPerformance = *typeInfo.NetworkInfo.NetworkPerformance
		}
		if typeInfo.NetworkInfo.EnaSupport != "" {
			info.NetworkPerformance += " (ENA)"
		}
	}

	// EBS Optimization
	if typeInfo.EbsInfo != nil {
		if typeInfo.EbsInfo.EbsOptimizedSupport != "" {
			info.EbsOptimized = (string(typeInfo.EbsInfo.EbsOptimizedSupport) == "default" ||
								 string(typeInfo.EbsInfo.EbsOptimizedSupport) == "supported")
		}
	}

	// Instance Storage
	if typeInfo.InstanceStorageInfo != nil {
		if typeInfo.InstanceStorageInfo.TotalSizeInGB != nil {
			info.InstanceStorageGB = *typeInfo.InstanceStorageInfo.TotalSizeInGB
			info.StorageType = "Instance Store"
		}
	}
	if info.InstanceStorageGB == 0 {
		info.StorageType = "EBS Only"
	}

	// Supported Architectures
	for _, arch := range typeInfo.ProcessorInfo.SupportedArchitectures {
		info.SupportedArchitectures = append(info.SupportedArchitectures, string(arch))
	}

	return info, nil
}
