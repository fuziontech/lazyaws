package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps AWS service clients
type Client struct {
	EC2    *ec2.Client
	S3     *s3.Client
	EKS    *eks.Client
	Region string
}

// NewClient creates a new AWS client with the default configuration
func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		EC2:    ec2.NewFromConfig(cfg),
		S3:     s3.NewFromConfig(cfg),
		EKS:    eks.NewFromConfig(cfg),
		Region: cfg.Region,
	}, nil
}

// NewClientWithProfile creates a new AWS client with a specific profile
func NewClientWithProfile(ctx context.Context, profile string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		EC2:    ec2.NewFromConfig(cfg),
		S3:     s3.NewFromConfig(cfg),
		EKS:    eks.NewFromConfig(cfg),
		Region: cfg.Region,
	}, nil
}

// GetRegion returns the configured AWS region
func (c *Client) GetRegion() string {
	return c.Region
}
