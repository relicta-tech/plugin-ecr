package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// AWSConfig holds AWS configuration.
type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Profile         string
	RoleARN         string
}

// AuthToken holds ECR authentication credentials.
type AuthToken struct {
	Username string
	Password string
	Endpoint string
}

// ECRClient provides ECR operations.
type ECRClient struct {
	config *AWSConfig
}

// NewECRClient creates a new ECR client.
func NewECRClient(ctx context.Context, config *AWSConfig) (*ECRClient, error) {
	if config.Region == "" {
		return nil, fmt.Errorf("AWS region is required")
	}

	return &ECRClient{
		config: config,
	}, nil
}

// GetRegistryURL returns the ECR registry URL.
func (c *ECRClient) GetRegistryURL(ctx context.Context, registryID string, public bool) (string, error) {
	if public {
		return "public.ecr.aws", nil
	}

	// For private ECR, construct the URL
	// Format: <account_id>.dkr.ecr.<region>.amazonaws.com
	accountID := registryID
	if accountID == "" {
		// In a real implementation, we would call STS GetCallerIdentity
		// For now, require explicit registry ID
		return "", fmt.Errorf("registry_id (AWS account ID) is required for private ECR")
	}

	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, c.config.Region), nil
}

// GetAuthorizationToken gets an ECR authorization token.
func (c *ECRClient) GetAuthorizationToken(ctx context.Context, registryID string, public bool) (*AuthToken, error) {
	// In a real implementation, this would call:
	// - ECR GetAuthorizationToken for private registries
	// - ECR Public GetAuthorizationToken for public registries
	//
	// The response contains a base64-encoded token in format: AWS:<password>
	// This is decoded and used for docker login

	// For this implementation, we'll use the AWS CLI approach
	// which handles credentials via the default credential chain

	registryURL, err := c.GetRegistryURL(ctx, registryID, public)
	if err != nil {
		return nil, err
	}

	// Placeholder - in production, this would use AWS SDK
	// The auth token format from ECR is: base64(AWS:<temporary_password>)
	return &AuthToken{
		Username: "AWS",
		Password: "", // Would be populated by AWS SDK call
		Endpoint: registryURL,
	}, nil
}

// CreateRepositoryIfNotExists creates a repository if it doesn't exist.
func (c *ECRClient) CreateRepositoryIfNotExists(ctx context.Context, name string, public bool) error {
	// In a real implementation, this would:
	// 1. Try to describe the repository
	// 2. If it doesn't exist (RepositoryNotFoundException), create it
	// 3. Handle public vs private repository creation differently

	fmt.Printf("Ensuring repository exists: %s\n", name)
	return nil
}

// DecodeAuthToken decodes an ECR authorization token.
func DecodeAuthToken(token string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode auth token: %w", err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth token format")
	}

	return parts[0], parts[1], nil
}
