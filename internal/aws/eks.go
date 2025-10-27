package aws

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

// EKSCluster represents an EKS cluster with relevant information
type EKSCluster struct {
	Name      string
	Version   string
	Status    string
	Endpoint  string
	Region    string
	CreatedAt string
	NodeCount int
	Arn       string
}

// EKSClusterDetails contains detailed information about an EKS cluster
type EKSClusterDetails struct {
	Name                    string
	Version                 string
	Status                  string
	Endpoint                string
	Region                  string
	CreatedAt               string
	Arn                     string
	RoleArn                 string
	CertificateAuthority    string
	VpcId                   string
	SubnetIds               []string
	SecurityGroupIds        []string
	EnabledLogTypes         []string
	PlatformVersion         string
	Tags                    map[string]string
}

// EKSNodeGroup represents an EKS node group
type EKSNodeGroup struct {
	Name             string
	Status           string
	InstanceTypes    []string
	DesiredSize      int32
	MinSize          int32
	MaxSize          int32
	AmiType          string
	CreatedAt        string
	Version          string
	NodeRole         string
}

// EKSAddon represents an EKS cluster add-on
type EKSAddon struct {
	Name             string
	Version          string
	Status           string
	Health           string
	CreatedAt        string
	ModifiedAt       string
	ServiceAccountRole string
}

// ListEKSClusters retrieves all EKS clusters in the current region
func (c *Client) ListEKSClusters(ctx context.Context) ([]EKSCluster, error) {
	input := &eks.ListClustersInput{}
	result, err := c.EKS.ListClusters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
	}

	var clusters []EKSCluster
	for _, clusterName := range result.Clusters {
		// Get detailed information for each cluster
		details, err := c.GetEKSClusterDetails(ctx, clusterName)
		if err != nil {
			// If we can't get details, still add the cluster with basic info
			clusters = append(clusters, EKSCluster{
				Name:   clusterName,
				Status: "unknown",
				Region: c.Region,
			})
			continue
		}

		// Count node groups for this cluster
		nodeCount, _ := c.countNodeGroupNodes(ctx, clusterName)

		clusters = append(clusters, EKSCluster{
			Name:      details.Name,
			Version:   details.Version,
			Status:    details.Status,
			Endpoint:  details.Endpoint,
			Region:    c.Region,
			CreatedAt: details.CreatedAt,
			NodeCount: nodeCount,
			Arn:       details.Arn,
		})
	}

	return clusters, nil
}

// GetEKSClusterDetails retrieves detailed information about an EKS cluster
func (c *Client) GetEKSClusterDetails(ctx context.Context, clusterName string) (*EKSClusterDetails, error) {
	input := &eks.DescribeClusterInput{
		Name: &clusterName,
	}

	result, err := c.EKS.DescribeCluster(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	cluster := result.Cluster
	details := &EKSClusterDetails{
		Name:    getString(cluster.Name),
		Version: getString(cluster.Version),
		Status:  string(cluster.Status),
		Region:  c.Region,
		Arn:     getString(cluster.Arn),
	}

	if cluster.Endpoint != nil {
		details.Endpoint = *cluster.Endpoint
	}

	if cluster.CreatedAt != nil {
		details.CreatedAt = cluster.CreatedAt.Format("2006-01-02 15:04:05")
	}

	if cluster.RoleArn != nil {
		details.RoleArn = *cluster.RoleArn
	}

	if cluster.CertificateAuthority != nil && cluster.CertificateAuthority.Data != nil {
		details.CertificateAuthority = *cluster.CertificateAuthority.Data
	}

	if cluster.ResourcesVpcConfig != nil {
		if cluster.ResourcesVpcConfig.VpcId != nil {
			details.VpcId = *cluster.ResourcesVpcConfig.VpcId
		}
		details.SubnetIds = cluster.ResourcesVpcConfig.SubnetIds
		details.SecurityGroupIds = cluster.ResourcesVpcConfig.SecurityGroupIds
	}

	if cluster.Logging != nil && cluster.Logging.ClusterLogging != nil {
		for _, logSetup := range cluster.Logging.ClusterLogging {
			if logSetup.Enabled != nil && *logSetup.Enabled {
				for _, logType := range logSetup.Types {
					details.EnabledLogTypes = append(details.EnabledLogTypes, string(logType))
				}
			}
		}
	}

	if cluster.PlatformVersion != nil {
		details.PlatformVersion = *cluster.PlatformVersion
	}

	if cluster.Tags != nil {
		details.Tags = cluster.Tags
	}

	return details, nil
}

// ListNodeGroups lists all node groups for a cluster
func (c *Client) ListNodeGroups(ctx context.Context, clusterName string) ([]EKSNodeGroup, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	}

	result, err := c.EKS.ListNodegroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list node groups: %w", err)
	}

	var nodeGroups []EKSNodeGroup
	for _, ngName := range result.Nodegroups {
		ng, err := c.GetNodeGroupDetails(ctx, clusterName, ngName)
		if err != nil {
			// If we can't get details, still add the node group with basic info
			nodeGroups = append(nodeGroups, EKSNodeGroup{
				Name:   ngName,
				Status: "unknown",
			})
			continue
		}
		nodeGroups = append(nodeGroups, *ng)
	}

	return nodeGroups, nil
}

// GetNodeGroupDetails retrieves detailed information about a node group
func (c *Client) GetNodeGroupDetails(ctx context.Context, clusterName, nodeGroupName string) (*EKSNodeGroup, error) {
	input := &eks.DescribeNodegroupInput{
		ClusterName:   &clusterName,
		NodegroupName: &nodeGroupName,
	}

	result, err := c.EKS.DescribeNodegroup(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe node group: %w", err)
	}

	ng := result.Nodegroup
	details := &EKSNodeGroup{
		Name:   getString(ng.NodegroupName),
		Status: string(ng.Status),
	}

	if ng.InstanceTypes != nil {
		details.InstanceTypes = ng.InstanceTypes
	}

	if ng.ScalingConfig != nil {
		if ng.ScalingConfig.DesiredSize != nil {
			details.DesiredSize = *ng.ScalingConfig.DesiredSize
		}
		if ng.ScalingConfig.MinSize != nil {
			details.MinSize = *ng.ScalingConfig.MinSize
		}
		if ng.ScalingConfig.MaxSize != nil {
			details.MaxSize = *ng.ScalingConfig.MaxSize
		}
	}

	if ng.AmiType != "" {
		details.AmiType = string(ng.AmiType)
	}

	if ng.CreatedAt != nil {
		details.CreatedAt = ng.CreatedAt.Format("2006-01-02 15:04:05")
	}

	if ng.Version != nil {
		details.Version = *ng.Version
	}

	if ng.NodeRole != nil {
		details.NodeRole = *ng.NodeRole
	}

	return details, nil
}

// ListAddons lists all add-ons for a cluster
func (c *Client) ListAddons(ctx context.Context, clusterName string) ([]EKSAddon, error) {
	input := &eks.ListAddonsInput{
		ClusterName: &clusterName,
	}

	result, err := c.EKS.ListAddons(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list add-ons: %w", err)
	}

	var addons []EKSAddon
	for _, addonName := range result.Addons {
		addon, err := c.GetAddonDetails(ctx, clusterName, addonName)
		if err != nil {
			// If we can't get details, still add the add-on with basic info
			addons = append(addons, EKSAddon{
				Name:   addonName,
				Status: "unknown",
			})
			continue
		}
		addons = append(addons, *addon)
	}

	return addons, nil
}

// GetAddonDetails retrieves detailed information about an add-on
func (c *Client) GetAddonDetails(ctx context.Context, clusterName, addonName string) (*EKSAddon, error) {
	input := &eks.DescribeAddonInput{
		ClusterName: &clusterName,
		AddonName:   &addonName,
	}

	result, err := c.EKS.DescribeAddon(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe add-on: %w", err)
	}

	addon := result.Addon
	details := &EKSAddon{
		Name:    getString(addon.AddonName),
		Version: getString(addon.AddonVersion),
		Status:  string(addon.Status),
	}

	if addon.Health != nil {
		for _, issue := range addon.Health.Issues {
			if issue.Code != "" {
				details.Health = string(issue.Code)
				break
			}
		}
		if details.Health == "" {
			details.Health = "Healthy"
		}
	}

	if addon.CreatedAt != nil {
		details.CreatedAt = addon.CreatedAt.Format("2006-01-02 15:04:05")
	}

	if addon.ModifiedAt != nil {
		details.ModifiedAt = addon.ModifiedAt.Format("2006-01-02 15:04:05")
	}

	if addon.ServiceAccountRoleArn != nil {
		details.ServiceAccountRole = *addon.ServiceAccountRoleArn
	}

	return details, nil
}

// UpdateKubeconfig updates the local kubeconfig file with the cluster configuration
func (c *Client) UpdateKubeconfig(ctx context.Context, clusterName string) error {
	// Get the kubeconfig path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")

	// Use aws eks update-kubeconfig command
	cmd := exec.CommandContext(ctx, "aws", "eks", "update-kubeconfig",
		"--name", clusterName,
		"--region", c.Region,
		"--kubeconfig", kubeconfigPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update kubeconfig: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetClusterLogs retrieves cluster logs (requires CloudWatch Logs integration)
func (c *Client) GetClusterLogs(ctx context.Context, clusterName string, logType types.LogType) ([]string, error) {
	// First, check if the log type is enabled
	details, err := c.GetEKSClusterDetails(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	enabled := false
	for _, lt := range details.EnabledLogTypes {
		if lt == string(logType) {
			enabled = true
			break
		}
	}

	if !enabled {
		return nil, fmt.Errorf("log type %s is not enabled for cluster %s", logType, clusterName)
	}

	// Note: Actual log retrieval would require CloudWatch Logs integration
	// This is a placeholder that returns the log group name
	logGroupName := fmt.Sprintf("/aws/eks/%s/cluster", clusterName)
	return []string{
		fmt.Sprintf("Logs are available in CloudWatch Logs group: %s", logGroupName),
		fmt.Sprintf("Log type: %s", logType),
	}, nil
}

// countNodeGroupNodes counts the total number of nodes across all node groups
func (c *Client) countNodeGroupNodes(ctx context.Context, clusterName string) (int, error) {
	nodeGroups, err := c.ListNodeGroups(ctx, clusterName)
	if err != nil {
		return 0, err
	}

	totalNodes := 0
	for _, ng := range nodeGroups {
		totalNodes += int(ng.DesiredSize)
	}

	return totalNodes, nil
}
