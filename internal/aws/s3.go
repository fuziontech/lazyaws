package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Bucket represents an S3 bucket with relevant information
type Bucket struct {
	Name         string
	CreationDate string
	Region       string
}

// S3Object represents an object or folder in an S3 bucket
type S3Object struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
	IsFolder     bool
}

// S3ListResult contains the result of listing objects with pagination support
type S3ListResult struct {
	Objects               []S3Object
	NextContinuationToken *string
	IsTruncated           bool
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

// ListObjects retrieves objects in an S3 bucket with optional prefix and pagination
func (c *Client) ListObjects(ctx context.Context, bucketName, prefix string, continuationToken *string) (*S3ListResult, error) {
	delimiter := "/"
	input := &s3.ListObjectsV2Input{
		Bucket:    &bucketName,
		Prefix:    &prefix,
		Delimiter: &delimiter,     // Use delimiter to show folders
		MaxKeys:   getInt32(1000), // Limit to 1000 objects per page
	}

	if continuationToken != nil {
		input.ContinuationToken = continuationToken
	}

	result, err := c.S3.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var objects []S3Object

	// Add folders (common prefixes)
	for _, commonPrefix := range result.CommonPrefixes {
		if commonPrefix.Prefix != nil {
			// Extract folder name from prefix
			folderKey := getString(commonPrefix.Prefix)
			objects = append(objects, S3Object{
				Key:      folderKey,
				IsFolder: true,
			})
		}
	}

	// Add files (objects)
	for _, obj := range result.Contents {
		key := getString(obj.Key)

		// Skip the prefix itself if it matches exactly
		if key == prefix {
			continue
		}

		storageClass := ""
		if obj.StorageClass != "" {
			storageClass = string(obj.StorageClass)
		} else {
			storageClass = string(types.ObjectStorageClassStandard)
		}

		lastModified := ""
		if obj.LastModified != nil {
			lastModified = obj.LastModified.Format("2006-01-02 15:04:05")
		}

		objects = append(objects, S3Object{
			Key:          key,
			Size:         getInt64(obj.Size),
			LastModified: lastModified,
			StorageClass: storageClass,
			IsFolder:     false,
		})
	}

	return &S3ListResult{
		Objects:               objects,
		NextContinuationToken: result.NextContinuationToken,
		IsTruncated:           getBool(result.IsTruncated),
	}, nil
}

// Helper function to get string pointer value
func getBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// Helper function to get int64 pointer value
func getInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// Helper function to create int32 pointer
func getInt32(i int32) *int32 {
	return &i
}
