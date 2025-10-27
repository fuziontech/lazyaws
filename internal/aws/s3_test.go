package aws

import (
	"testing"
)

func TestBucket(t *testing.T) {
	// Test Bucket struct creation
	bucket := Bucket{
		Name:         "test-bucket",
		CreationDate: "2024-01-01 12:00:00",
		Region:       "us-east-1",
	}

	if bucket.Name != "test-bucket" {
		t.Errorf("Expected bucket name 'test-bucket', got '%s'", bucket.Name)
	}

	if bucket.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", bucket.Region)
	}
}

func TestBucketEmptyRegion(t *testing.T) {
	// Test bucket with empty region
	bucket := Bucket{
		Name:   "test-bucket",
		Region: "",
	}

	if bucket.Region != "" {
		t.Errorf("Expected empty region, got '%s'", bucket.Region)
	}
}
