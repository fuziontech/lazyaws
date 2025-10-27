package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Bucket represents an S3 bucket with relevant information
type Bucket struct {
	Name         string
	CreationDate string
	Region       string
	Size         int64 // Total size in bytes
	ObjectCount  int64 // Total number of objects
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

// ProgressCallback is a function that receives progress updates
type ProgressCallback func(bytesTransferred int64, totalBytes int64)

// progressWriter wraps an io.Writer to track progress
type progressWriter struct {
	writer   io.Writer
	total    int64
	written  int64
	callback ProgressCallback
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.written += int64(n)
	if pw.callback != nil {
		pw.callback(pw.written, pw.total)
	}
	return n, err
}

// progressReader wraps an io.Reader to track progress
type progressReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback ProgressCallback
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.callback != nil {
		pr.callback(pr.read, pr.total)
	}
	return n, err
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

		// Set region to "loading..." initially - will be fetched async if needed
		b.Region = "-"

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
	return c.DownloadObjectWithProgress(ctx, bucketName, key, localPath, nil)
}

// DownloadObjectWithProgress downloads an S3 object with progress tracking
func (c *Client) DownloadObjectWithProgress(ctx context.Context, bucketName, key, localPath string, progressCallback ProgressCallback) error {
	// Get object size first for progress tracking
	headInput := &s3.HeadObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	}
	headResult, err := c.S3.HeadObject(ctx, headInput)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}

	objectSize := getInt64(headResult.ContentLength)

	// Create the file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// Download the object
	downloader := manager.NewDownloader(c.S3)

	// Wrap the file writer with progress tracking
	var writer io.WriterAt = file
	if progressCallback != nil {
		writer = &progressWriterAt{
			writer:   file,
			total:    objectSize,
			callback: progressCallback,
		}
	}

	_, err = downloader.Download(ctx, writer, &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}

	return nil
}

// progressWriterAt wraps an io.WriterAt to track progress
type progressWriterAt struct {
	writer   io.WriterAt
	total    int64
	written  int64
	callback ProgressCallback
}

func (pw *progressWriterAt) WriteAt(p []byte, off int64) (int, error) {
	n, err := pw.writer.WriteAt(p, off)
	pw.written += int64(n)
	if pw.callback != nil {
		pw.callback(pw.written, pw.total)
	}
	return n, err
}

// UploadObject uploads a local file to S3
func (c *Client) UploadObject(ctx context.Context, bucketName, key, localPath string) error {
	return c.UploadObjectWithProgress(ctx, bucketName, key, localPath, nil)
}

// UploadObjectWithProgress uploads a local file to S3 with progress tracking
// Automatically uses multipart upload for large files (>5MB)
func (c *Client) UploadObjectWithProgress(ctx context.Context, bucketName, key, localPath string, progressCallback ProgressCallback) error {
	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Create uploader with multipart support
	uploader := manager.NewUploader(c.S3, func(u *manager.Uploader) {
		// Use 10MB part size for multipart uploads
		u.PartSize = 10 * 1024 * 1024
	})

	// Wrap reader with progress tracking
	var reader io.Reader = file
	if progressCallback != nil {
		reader = &progressReader{
			reader:   file,
			total:    fileSize,
			callback: progressCallback,
		}
	}

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   reader,
	})
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

// DeleteObject deletes an S3 object
func (c *Client) DeleteObject(ctx context.Context, bucketName, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	}

	_, err := c.S3.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// CopyObject copies an S3 object to another location
func (c *Client) CopyObject(ctx context.Context, sourceBucket, sourceKey, destBucket, destKey string) error {
	copySource := fmt.Sprintf("%s/%s", sourceBucket, sourceKey)
	input := &s3.CopyObjectInput{
		Bucket:     &destBucket,
		CopySource: &copySource,
		Key:        &destKey,
	}

	_, err := c.S3.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// CreateBucket creates a new S3 bucket
func (c *Client) CreateBucket(ctx context.Context, bucketName, region string) error {
	input := &s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	// If not us-east-1, set location constraint
	if region != "us-east-1" && region != "" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err := c.S3.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// DeleteBucket deletes an S3 bucket (must be empty)
func (c *Client) DeleteBucket(ctx context.Context, bucketName string) error {
	input := &s3.DeleteBucketInput{
		Bucket: &bucketName,
	}

	_, err := c.S3.DeleteBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// GetBucketPolicy retrieves the bucket policy
func (c *Client) GetBucketPolicy(ctx context.Context, bucketName string) (string, error) {
	input := &s3.GetBucketPolicyInput{
		Bucket: &bucketName,
	}

	result, err := c.S3.GetBucketPolicy(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket policy: %w", err)
	}

	if result.Policy == nil {
		return "", nil
	}

	return *result.Policy, nil
}

// GetBucketVersioning retrieves the versioning configuration
func (c *Client) GetBucketVersioning(ctx context.Context, bucketName string) (string, error) {
	input := &s3.GetBucketVersioningInput{
		Bucket: &bucketName,
	}

	result, err := c.S3.GetBucketVersioning(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket versioning: %w", err)
	}

	if result.Status == "" {
		return "Disabled", nil
	}

	return string(result.Status), nil
}

// GeneratePresignedURL generates a presigned URL for an S3 object
func (c *Client) GeneratePresignedURL(ctx context.Context, bucketName, key string, expirationSeconds int) (string, error) {
	// Create a presign client
	presignClient := s3.NewPresignClient(c.S3)

	input := &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	}

	// Generate presigned URL with expiration
	presignResult, err := presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expirationSeconds) * time.Second
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

// GetBucketSize calculates the total size and object count for a bucket
// Note: This can be slow for large buckets as it needs to list all objects
func (c *Client) GetBucketSize(ctx context.Context, bucketName string) (int64, int64, error) {
	var totalSize int64
	var objectCount int64
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket: &bucketName,
		}

		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}

		result, err := c.S3.ListObjectsV2(ctx, input)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to list objects for size calculation: %w", err)
		}

		for _, obj := range result.Contents {
			totalSize += getInt64(obj.Size)
			objectCount++
		}

		if !getBool(result.IsTruncated) {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return totalSize, objectCount, nil
}

// ListObjectsWithFilter retrieves objects in an S3 bucket with filtering support
func (c *Client) ListObjectsWithFilter(ctx context.Context, bucketName, prefix, pattern string, continuationToken *string) (*S3ListResult, error) {
	// First, get all objects with the given prefix
	result, err := c.ListObjects(ctx, bucketName, prefix, continuationToken)
	if err != nil {
		return nil, err
	}

	// If no pattern specified, return all results
	if pattern == "" {
		return result, nil
	}

	// Filter objects by pattern (simple contains match for now)
	var filteredObjects []S3Object
	for _, obj := range result.Objects {
		// Simple pattern matching - check if key contains the pattern
		// Could be enhanced with regex or glob patterns
		if containsIgnoreCase(obj.Key, pattern) {
			filteredObjects = append(filteredObjects, obj)
		}
	}

	result.Objects = filteredObjects
	return result, nil
}

// Helper function for case-insensitive string matching
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// SyncLocalToS3 syncs a local directory to an S3 bucket (like aws s3 sync)
func (c *Client) SyncLocalToS3(ctx context.Context, localDir, bucketName, s3Prefix string, progressCallback ProgressCallback) error {
	// Walk through local directory
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path and S3 key
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Convert Windows paths to S3 format
		s3Key := filepath.ToSlash(relPath)
		if s3Prefix != "" {
			s3Key = s3Prefix + "/" + s3Key
		}

		// Check if object exists and compare modification time
		shouldUpload := true
		headInput := &s3.HeadObjectInput{
			Bucket: &bucketName,
			Key:    &s3Key,
		}

		headResult, err := c.S3.HeadObject(ctx, headInput)
		if err == nil {
			// Object exists, check if local file is newer
			if headResult.LastModified != nil && !info.ModTime().After(*headResult.LastModified) {
				shouldUpload = false
			}
		}

		if shouldUpload {
			err = c.UploadObjectWithProgress(ctx, bucketName, s3Key, path, progressCallback)
			if err != nil {
				return fmt.Errorf("failed to upload %s: %w", path, err)
			}
		}

		return nil
	})
}

// SyncS3ToLocal syncs an S3 bucket prefix to a local directory
func (c *Client) SyncS3ToLocal(ctx context.Context, bucketName, s3Prefix, localDir string, progressCallback ProgressCallback) error {
	var continuationToken *string

	for {
		// List objects with prefix
		result, err := c.ListObjects(ctx, bucketName, s3Prefix, continuationToken)
		if err != nil {
			return err
		}

		for _, obj := range result.Objects {
			// Skip folders
			if obj.IsFolder {
				continue
			}

			// Calculate local path
			relKey := strings.TrimPrefix(obj.Key, s3Prefix)
			relKey = strings.TrimPrefix(relKey, "/")
			localPath := filepath.Join(localDir, filepath.FromSlash(relKey))

			// Create directory if needed
			localDir := filepath.Dir(localPath)
			if err := os.MkdirAll(localDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Check if local file exists and compare modification time
			shouldDownload := true
			if fileInfo, err := os.Stat(localPath); err == nil {
				// Parse object's last modified time
				objTime, _ := time.Parse("2006-01-02 15:04:05", obj.LastModified)
				if !objTime.After(fileInfo.ModTime()) {
					shouldDownload = false
				}
			}

			if shouldDownload {
				err = c.DownloadObjectWithProgress(ctx, bucketName, obj.Key, localPath, progressCallback)
				if err != nil {
					return fmt.Errorf("failed to download %s: %w", obj.Key, err)
				}
			}
		}

		if !result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return nil
}

// ListObjectVersions lists all versions of objects in a bucket
func (c *Client) ListObjectVersions(ctx context.Context, bucketName, prefix string) ([]S3ObjectVersion, error) {
	input := &s3.ListObjectVersionsInput{
		Bucket: &bucketName,
	}

	if prefix != "" {
		input.Prefix = &prefix
	}

	result, err := c.S3.ListObjectVersions(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list object versions: %w", err)
	}

	var versions []S3ObjectVersion
	for _, version := range result.Versions {
		v := S3ObjectVersion{
			Key:          getString(version.Key),
			VersionId:    getString(version.VersionId),
			IsLatest:     getBool(version.IsLatest),
			Size:         getInt64(version.Size),
			StorageClass: string(version.StorageClass),
		}

		if version.LastModified != nil {
			v.LastModified = version.LastModified.Format("2006-01-02 15:04:05")
		}

		versions = append(versions, v)
	}

	return versions, nil
}

// S3ObjectVersion represents a version of an S3 object
type S3ObjectVersion struct {
	Key          string
	VersionId    string
	IsLatest     bool
	Size         int64
	LastModified string
	StorageClass string
}

// GetObjectVersion downloads a specific version of an S3 object
func (c *Client) GetObjectVersion(ctx context.Context, bucketName, key, versionId, localPath string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	input := &s3.GetObjectInput{
		Bucket:    &bucketName,
		Key:       &key,
		VersionId: &versionId,
	}

	result, err := c.S3.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to download object version: %w", err)
	}
	defer result.Body.Close()

	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write to local file: %w", err)
	}

	return nil
}

// DeleteObjectVersion deletes a specific version of an S3 object
func (c *Client) DeleteObjectVersion(ctx context.Context, bucketName, key, versionId string) error {
	input := &s3.DeleteObjectInput{
		Bucket:    &bucketName,
		Key:       &key,
		VersionId: &versionId,
	}

	_, err := c.S3.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object version: %w", err)
	}

	return nil
}

// EnableBucketVersioning enables versioning on a bucket
func (c *Client) EnableBucketVersioning(ctx context.Context, bucketName string) error {
	enabled := types.BucketVersioningStatusEnabled
	input := &s3.PutBucketVersioningInput{
		Bucket: &bucketName,
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: enabled,
		},
	}

	_, err := c.S3.PutBucketVersioning(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to enable bucket versioning: %w", err)
	}

	return nil
}

// SuspendBucketVersioning suspends versioning on a bucket
func (c *Client) SuspendBucketVersioning(ctx context.Context, bucketName string) error {
	suspended := types.BucketVersioningStatusSuspended
	input := &s3.PutBucketVersioningInput{
		Bucket: &bucketName,
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: suspended,
		},
	}

	_, err := c.S3.PutBucketVersioning(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to suspend bucket versioning: %w", err)
	}

	return nil
}
