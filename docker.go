package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DockerClient provides Docker CLI operations.
type DockerClient struct{}

// NewDockerClient creates a new Docker client.
func NewDockerClient() *DockerClient {
	return &DockerClient{}
}

// Login authenticates with a Docker registry.
func (d *DockerClient) Login(ctx context.Context, registry string, auth *AuthToken) error {
	if auth.Password == "" {
		// Use AWS CLI for ECR login (handles credential chain)
		return d.loginWithAWSCLI(ctx, registry)
	}

	// Use docker login with provided credentials
	cmd := exec.CommandContext(ctx, "docker", "login",
		"--username", auth.Username,
		"--password-stdin",
		registry,
	)
	cmd.Stdin = strings.NewReader(auth.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w\n%s", err, string(output))
	}

	return nil
}

// loginWithAWSCLI uses AWS CLI for ECR authentication.
func (d *DockerClient) loginWithAWSCLI(ctx context.Context, registry string) error {
	// Determine if public or private ECR
	if strings.Contains(registry, "public.ecr.aws") {
		return d.loginPublicECR(ctx)
	}
	return d.loginPrivateECR(ctx, registry)
}

// loginPrivateECR authenticates with private ECR.
func (d *DockerClient) loginPrivateECR(ctx context.Context, registry string) error {
	// Extract region from registry URL
	// Format: <account>.dkr.ecr.<region>.amazonaws.com
	parts := strings.Split(registry, ".")
	if len(parts) < 4 {
		return fmt.Errorf("invalid ECR registry URL: %s", registry)
	}
	region := parts[3]

	// Get login password from AWS CLI
	getPasswordCmd := exec.CommandContext(ctx, "aws", "ecr", "get-login-password", "--region", region)
	password, err := getPasswordCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get ECR login password: %w", err)
	}

	// Login to Docker
	loginCmd := exec.CommandContext(ctx, "docker", "login",
		"--username", "AWS",
		"--password-stdin",
		registry,
	)
	loginCmd.Stdin = strings.NewReader(string(password))

	output, err := loginCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w\n%s", err, string(output))
	}

	return nil
}

// loginPublicECR authenticates with public ECR.
func (d *DockerClient) loginPublicECR(ctx context.Context) error {
	// Get login password from AWS CLI for public ECR
	getPasswordCmd := exec.CommandContext(ctx, "aws", "ecr-public", "get-login-password", "--region", "us-east-1")
	password, err := getPasswordCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get ECR public login password: %w", err)
	}

	// Login to Docker
	loginCmd := exec.CommandContext(ctx, "docker", "login",
		"--username", "AWS",
		"--password-stdin",
		"public.ecr.aws",
	)
	loginCmd.Stdin = strings.NewReader(string(password))

	output, err := loginCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w\n%s", err, string(output))
	}

	return nil
}

// Tag tags a Docker image.
func (d *DockerClient) Tag(ctx context.Context, source, target string) error {
	cmd := exec.CommandContext(ctx, "docker", "tag", source, target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker tag failed: %w\n%s", err, string(output))
	}
	return nil
}

// Push pushes a Docker image.
func (d *DockerClient) Push(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %w\n%s", err, string(output))
	}
	return nil
}

// ImageExists checks if a Docker image exists locally.
func (d *DockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	err := cmd.Run()
	if err != nil {
		// Image doesn't exist
		return false, nil
	}
	return true, nil
}
