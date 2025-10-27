package aws

import (
	"testing"
)

func TestInstanceTypeInfo(t *testing.T) {
	// Test InstanceTypeInfo struct creation
	typeInfo := InstanceTypeInfo{
		InstanceType:       "t3.medium",
		VCpus:              2,
		Memory:             4096, // 4 GiB in MiB
		NetworkPerformance: "Up to 5 Gigabit",
		StorageType:        "EBS Only",
		EbsOptimized:       true,
	}

	if typeInfo.InstanceType != "t3.medium" {
		t.Errorf("Expected instance type 't3.medium', got '%s'", typeInfo.InstanceType)
	}

	if typeInfo.VCpus != 2 {
		t.Errorf("Expected 2 vCPUs, got %d", typeInfo.VCpus)
	}

	if typeInfo.Memory != 4096 {
		t.Errorf("Expected 4096 MiB memory, got %d", typeInfo.Memory)
	}

	if !typeInfo.EbsOptimized {
		t.Error("Expected EBS Optimized to be true")
	}
}

func TestInstanceTypeInfoMemoryConversion(t *testing.T) {
	// Test memory conversion from MiB to GiB
	typeInfo := InstanceTypeInfo{
		Memory: 8192, // 8 GiB in MiB
	}

	memoryGB := float64(typeInfo.Memory) / 1024.0
	expectedGB := 8.0

	if memoryGB != expectedGB {
		t.Errorf("Expected %.2f GiB, got %.2f GiB", expectedGB, memoryGB)
	}
}

func TestInstanceDetails(t *testing.T) {
	// Test that InstanceDetails can hold InstanceTypeInfo
	details := InstanceDetails{
		Instance: Instance{
			ID:           "i-1234567890",
			Name:         "test-instance",
			State:        "running",
			InstanceType: "t3.medium",
		},
		InstanceTypeInfo: &InstanceTypeInfo{
			InstanceType: "t3.medium",
			VCpus:        2,
			Memory:       4096,
		},
	}

	if details.InstanceTypeInfo == nil {
		t.Error("Expected InstanceTypeInfo to be set")
	}

	if details.InstanceTypeInfo.VCpus != 2 {
		t.Errorf("Expected 2 vCPUs, got %d", details.InstanceTypeInfo.VCpus)
	}
}

func TestInstanceDetailsWithoutTypeInfo(t *testing.T) {
	// Test that InstanceDetails works without InstanceTypeInfo (optional field)
	details := InstanceDetails{
		Instance: Instance{
			ID:           "i-1234567890",
			Name:         "test-instance",
			State:        "running",
			InstanceType: "t3.medium",
		},
		InstanceTypeInfo: nil,
	}

	if details.InstanceTypeInfo != nil {
		t.Error("Expected InstanceTypeInfo to be nil")
	}

	// Should still be able to access basic instance info
	if details.ID != "i-1234567890" {
		t.Errorf("Expected instance ID 'i-1234567890', got '%s'", details.ID)
	}
}
