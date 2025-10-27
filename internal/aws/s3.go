package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Bucket represents an S3 bucket with relevant information
type Bucket struct {
	Name         string
	CreationDate string
	Region       string
}

// ListBuckets retrieves all S3 buckets
func (c *Client) ListBuckets(ctx context.Context) ([]Bucket, error) {
	input := &s3.ListBucketsInput{}
	result, err := c.S3.ListBuckets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var buckets []Bucket
	for _, bucket := range result.Buckets {
		b := Bucket{
			Name: getString(bucket.Name),
		}

		if bucket.CreationDate != nil {
			b.CreationDate = bucket.CreationDate.Format("2006-01-02 15:04:05")
		}

		// Get bucket region
		region, err := c.GetBucketRegion(ctx, b.Name)
		if err == nil {
			b.Region = region
		} else {
			b.Region = "unknown"
		}

		buckets = append(buckets, b)
	}

	return buckets, nil
}

// GetBucketRegion retrieves the region for a specific bucket
func (c *Client) GetBucketRegion(ctx context.Context, bucketName string) (string, error) {
	input := &s3.GetBucketLocationInput{
		Bucket: &bucketName,
	}

	result, err := c.S3.GetBucketLocation(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket location: %w", err)
	}

	// Empty location means us-east-1
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}

	return string(result.LocationConstraint), nil
}
