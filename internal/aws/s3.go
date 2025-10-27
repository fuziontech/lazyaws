package aws

import (
	"context"
	"fmt"
	"io"
	"os"

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

// S3ObjectDetails contains detailed information about an S3 object
type S3ObjectDetails struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
	ContentType  string
	ETag         string
	Metadata     map[string]string
	Tags         map[string]string
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

// DownloadObject downloads an S3 object to a local file
func (c *Client) DownloadObject(ctx context.Context, bucketName, key, localPath string) error {
	// Create the file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Download the object
	input := &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	}

	result, err := c.S3.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}
	defer result.Body.Close()

	// Copy the content to the file
	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write to local file: %w", err)
	}

	return nil
}

// UploadObject uploads a local file to S3
func (c *Client) UploadObject(ctx context.Context, bucketName, key, localPath string) error {
	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Upload the object
	input := &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   file,
	}

	_, err = c.S3.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// GetObjectDetails retrieves detailed information about an S3 object
func (c *Client) GetObjectDetails(ctx context.Context, bucketName, key string) (*S3ObjectDetails, error) {
	// Get object metadata
	headInput := &s3.HeadObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	}

	headResult, err := c.S3.HeadObject(ctx, headInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	details := &S3ObjectDetails{
		Key:      key,
		Size:     getInt64(headResult.ContentLength),
		Metadata: headResult.Metadata,
	}

	if headResult.LastModified != nil {
		details.LastModified = headResult.LastModified.Format("2006-01-02 15:04:05")
	}

	if headResult.StorageClass != "" {
		details.StorageClass = string(headResult.StorageClass)
	} else {
		details.StorageClass = string(types.ObjectStorageClassStandard)
	}

	if headResult.ContentType != nil {
		details.ContentType = *headResult.ContentType
	}

	if headResult.ETag != nil {
		details.ETag = *headResult.ETag
	}

	// Get object tags
	tagInput := &s3.GetObjectTaggingInput{
		Bucket: &bucketName,
		Key:    &key,
	}

	tagResult, err := c.S3.GetObjectTagging(ctx, tagInput)
	if err == nil && len(tagResult.TagSet) > 0 {
		details.Tags = make(map[string]string)
		for _, tag := range tagResult.TagSet {
			if tag.Key != nil && tag.Value != nil {
				details.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return details, nil
}
