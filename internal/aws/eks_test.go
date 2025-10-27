package aws

import (
	"testing"
)

func TestEKSCluster(t *testing.T) {
	// Test EKSCluster struct creation
	cluster := EKSCluster{
		Name:      "test-cluster",
		Version:   "1.28",
		Status:    "ACTIVE",
		Endpoint:  "https://test.eks.amazonaws.com",
		Region:    "us-east-1",
		CreatedAt: "2024-01-01 12:00:00",
		NodeCount: 3,
		Arn:       "arn:aws:eks:us-east-1:123456789012:cluster/test-cluster",
	}

	if cluster.Name != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", cluster.Name)
	}

	if cluster.Version != "1.28" {
		t.Errorf("Expected version '1.28', got '%s'", cluster.Version)
	}

	if cluster.Status != "ACTIVE" {
		t.Errorf("Expected status 'ACTIVE', got '%s'", cluster.Status)
	}

	if cluster.NodeCount != 3 {
		t.Errorf("Expected node count 3, got %d", cluster.NodeCount)
	}

	if cluster.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", cluster.Region)
	}
}

func TestEKSClusterDetails(t *testing.T) {
	// Test EKSClusterDetails struct creation
	details := EKSClusterDetails{
		Name:                 "test-cluster",
		Version:              "1.28",
		Status:               "ACTIVE",
		Endpoint:             "https://test.eks.amazonaws.com",
		Region:               "us-east-1",
		CreatedAt:            "2024-01-01 12:00:00",
		Arn:                  "arn:aws:eks:us-east-1:123456789012:cluster/test-cluster",
		RoleArn:              "arn:aws:iam::123456789012:role/eks-role",
		CertificateAuthority: "base64-cert",
		VpcId:                "vpc-123456",
		SubnetIds:            []string{"subnet-1", "subnet-2"},
		SecurityGroupIds:     []string{"sg-123"},
		EnabledLogTypes:      []string{"api", "audit"},
		PlatformVersion:      "eks.1",
		Tags:                 map[string]string{"env": "test"},
	}

	if details.Name != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", details.Name)
	}

	if len(details.SubnetIds) != 2 {
		t.Errorf("Expected 2 subnets, got %d", len(details.SubnetIds))
	}

	if len(details.EnabledLogTypes) != 2 {
		t.Errorf("Expected 2 enabled log types, got %d", len(details.EnabledLogTypes))
	}

	if details.Tags["env"] != "test" {
		t.Errorf("Expected tag 'env' to be 'test', got '%s'", details.Tags["env"])
	}
}

func TestEKSNodeGroup(t *testing.T) {
	// Test EKSNodeGroup struct creation
	nodeGroup := EKSNodeGroup{
		Name:          "test-ng",
		Status:        "ACTIVE",
		InstanceTypes: []string{"t3.medium", "t3.large"},
		DesiredSize:   3,
		MinSize:       1,
		MaxSize:       5,
		AmiType:       "AL2_x86_64",
		CreatedAt:     "2024-01-01 12:00:00",
		Version:       "1.28",
		NodeRole:      "arn:aws:iam::123456789012:role/node-role",
	}

	if nodeGroup.Name != "test-ng" {
		t.Errorf("Expected node group name 'test-ng', got '%s'", nodeGroup.Name)
	}

	if nodeGroup.Status != "ACTIVE" {
		t.Errorf("Expected status 'ACTIVE', got '%s'", nodeGroup.Status)
	}

	if len(nodeGroup.InstanceTypes) != 2 {
		t.Errorf("Expected 2 instance types, got %d", len(nodeGroup.InstanceTypes))
	}

	if nodeGroup.DesiredSize != 3 {
		t.Errorf("Expected desired size 3, got %d", nodeGroup.DesiredSize)
	}

	if nodeGroup.MinSize != 1 {
		t.Errorf("Expected min size 1, got %d", nodeGroup.MinSize)
	}

	if nodeGroup.MaxSize != 5 {
		t.Errorf("Expected max size 5, got %d", nodeGroup.MaxSize)
	}
}

func TestEKSAddon(t *testing.T) {
	// Test EKSAddon struct creation
	addon := EKSAddon{
		Name:               "vpc-cni",
		Version:            "v1.12.0",
		Status:             "ACTIVE",
		Health:             "Healthy",
		CreatedAt:          "2024-01-01 12:00:00",
		ModifiedAt:         "2024-01-02 12:00:00",
		ServiceAccountRole: "arn:aws:iam::123456789012:role/vpc-cni-role",
	}

	if addon.Name != "vpc-cni" {
		t.Errorf("Expected addon name 'vpc-cni', got '%s'", addon.Name)
	}

	if addon.Version != "v1.12.0" {
		t.Errorf("Expected version 'v1.12.0', got '%s'", addon.Version)
	}

	if addon.Status != "ACTIVE" {
		t.Errorf("Expected status 'ACTIVE', got '%s'", addon.Status)
	}

	if addon.Health != "Healthy" {
		t.Errorf("Expected health 'Healthy', got '%s'", addon.Health)
	}
}

func TestEKSClusterWithEmptyFields(t *testing.T) {
	// Test EKSCluster with minimal fields
	cluster := EKSCluster{
		Name:   "minimal-cluster",
		Status: "CREATING",
	}

	if cluster.Name != "minimal-cluster" {
		t.Errorf("Expected cluster name 'minimal-cluster', got '%s'", cluster.Name)
	}

	if cluster.Version != "" {
		t.Errorf("Expected empty version, got '%s'", cluster.Version)
	}

	if cluster.NodeCount != 0 {
		t.Errorf("Expected node count 0, got %d", cluster.NodeCount)
	}
}

func TestEKSNodeGroupScaling(t *testing.T) {
	// Test node group scaling configuration
	tests := []struct {
		name        string
		desiredSize int32
		minSize     int32
		maxSize     int32
		valid       bool
	}{
		{"valid-scaling", 3, 1, 5, true},
		{"min-equals-desired", 2, 2, 5, true},
		{"max-equals-desired", 5, 1, 5, true},
		{"all-equal", 3, 3, 3, true},
	}

	for _, tt := range tests {
		ng := EKSNodeGroup{
			Name:        tt.name,
			DesiredSize: tt.desiredSize,
			MinSize:     tt.minSize,
			MaxSize:     tt.maxSize,
		}

		// Basic validation
		if ng.DesiredSize < ng.MinSize || ng.DesiredSize > ng.MaxSize {
			if tt.valid {
				t.Errorf("Test '%s': expected valid configuration but got invalid: desired=%d, min=%d, max=%d",
					tt.name, ng.DesiredSize, ng.MinSize, ng.MaxSize)
			}
		} else {
			if !tt.valid {
				t.Errorf("Test '%s': expected invalid configuration but got valid: desired=%d, min=%d, max=%d",
					tt.name, ng.DesiredSize, ng.MinSize, ng.MaxSize)
			}
		}
	}
}
