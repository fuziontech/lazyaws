package aws

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fuziontech/lazyaws/internal/config"
)

// Client wraps AWS service clients
type Client struct {
	EC2         *ec2.Client
	S3          *s3.Client
	EKS         *eks.Client
	SSM         *ssm.Client
	CloudWatch  *cloudwatch.Client
	STS         *sts.Client
	Region      string
	AccountID   string
	AccountName string
}

// NewClient creates a new AWS client with the default configuration
func NewClient(ctx context.Context, appConfig *config.Config) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(appConfig.Region))
	if err != nil {
		return nil, err
	}

	client := &Client{
		EC2:        ec2.NewFromConfig(cfg),
		S3:         s3.NewFromConfig(cfg),
		EKS:        eks.NewFromConfig(cfg),
		SSM:        ssm.NewFromConfig(cfg),
		CloudWatch: cloudwatch.NewFromConfig(cfg),
		STS:        sts.NewFromConfig(cfg),
		Region:     cfg.Region,
	}

	// Try to get account identity
	_ = client.loadAccountIdentity(ctx)

	return client, nil
}

// NewClientWithProfile creates a new AWS client with a specific profile
func NewClientWithProfile(ctx context.Context, profile string) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, err
	}

	client := &Client{
		EC2:        ec2.NewFromConfig(cfg),
		S3:         s3.NewFromConfig(cfg),
		EKS:        eks.NewFromConfig(cfg),
		SSM:        ssm.NewFromConfig(cfg),
		CloudWatch: cloudwatch.NewFromConfig(cfg),
		STS:        sts.NewFromConfig(cfg),
		Region:     cfg.Region,
	}

	// Try to get account identity
	_ = client.loadAccountIdentity(ctx)

	return client, nil
}

// NewClientWithSSOCredentials creates a new AWS client with SSO credentials
func NewClientWithSSOCredentials(ctx context.Context, creds *SSOCredentials, region, accountName string) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)),
	)
	if err != nil {
		return nil, err
	}

	client := &Client{
		EC2:         ec2.NewFromConfig(cfg),
		S3:          s3.NewFromConfig(cfg),
		EKS:         eks.NewFromConfig(cfg),
		SSM:         ssm.NewFromConfig(cfg),
		CloudWatch:  cloudwatch.NewFromConfig(cfg),
		STS:         sts.NewFromConfig(cfg),
		Region:      cfg.Region,
		AccountName: accountName,
	}

	// Get account identity
	_ = client.loadAccountIdentity(ctx)

	return client, nil
}

// GetRegion returns the configured AWS region
func (c *Client) GetRegion() string {
	return c.Region
}

// GetAccountID returns the AWS account ID
func (c *Client) GetAccountID() string {
	return c.AccountID
}

// GetAccountName returns the AWS account name (from SSO)
func (c *Client) GetAccountName() string {
	return c.AccountName
}

// loadAccountIdentity loads the account ID using STS GetCallerIdentity
func (c *Client) loadAccountIdentity(ctx context.Context) error {
	if c.STS == nil {
		return nil
	}

	resp, err := c.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return err
	}

	if resp.Account != nil {
		c.AccountID = *resp.Account
	}

	return nil
}
