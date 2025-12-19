package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

// Version is set at build time.
var Version = "dev"

// ECRPlugin implements the Relicta plugin interface for AWS ECR.
type ECRPlugin struct{}

// Config holds the plugin configuration.
type Config struct {
	// AWS Configuration
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Profile         string
	RoleARN         string

	// ECR Configuration
	RegistryID string
	Repository string
	Public     bool

	// Image Configuration
	SourceImage string
	Tags        []string

	// Behavior
	CreateRepo bool
	DryRun     bool
}

// GetInfo returns plugin metadata.
func (p *ECRPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		Name:        "ecr",
		Version:     Version,
		Description: "Push container images to AWS Elastic Container Registry (ECR)",
		Hooks: []plugin.Hook{
			plugin.HookPostPublish,
		},
	}
}

// Validate validates the plugin configuration.
func (p *ECRPlugin) Validate(ctx context.Context, config map[string]any) (*plugin.ValidateResponse, error) {
	vb := helpers.NewValidationBuilder()
	cfg := p.parseConfig(config)

	// Region is required
	if cfg.Region == "" {
		vb.AddError("region", "AWS region is required")
	}

	// Repository is required
	if cfg.Repository == "" {
		vb.AddError("repository", "ECR repository name is required")
	}

	// Source image is required
	if cfg.SourceImage == "" {
		vb.AddError("source_image", "source image is required")
	}

	// At least one tag required
	if len(cfg.Tags) == 0 {
		vb.AddError("tags", "at least one image tag is required")
	}

	// Validate tags format
	for i, tag := range cfg.Tags {
		if tag == "" {
			vb.AddError(fmt.Sprintf("tags[%d]", i), "tag cannot be empty")
		}
	}

	return vb.Build(), nil
}

// Execute runs the plugin logic.
func (p *ECRPlugin) Execute(ctx context.Context, req plugin.ExecuteRequest) (*plugin.ExecuteResponse, error) {
	cfg := p.parseConfig(req.Config)
	cfg.DryRun = cfg.DryRun || req.DryRun

	// Process tag templates
	tags := p.processTags(cfg.Tags, &req.Context)

	// Create ECR client
	client, err := NewECRClient(ctx, &AWSConfig{
		Region:          cfg.Region,
		AccessKeyID:     cfg.AccessKeyID,
		SecretAccessKey: cfg.SecretAccessKey,
		SessionToken:    cfg.SessionToken,
		Profile:         cfg.Profile,
		RoleARN:         cfg.RoleARN,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ECR client: %w", err)
	}

	// Build registry URL
	registryURL, err := client.GetRegistryURL(ctx, cfg.RegistryID, cfg.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry URL: %w", err)
	}

	// Create repository if needed
	if cfg.CreateRepo {
		if cfg.DryRun {
			fmt.Printf("[dry-run] Would create repository: %s\n", cfg.Repository)
		} else {
			if err := client.CreateRepositoryIfNotExists(ctx, cfg.Repository, cfg.Public); err != nil {
				return nil, fmt.Errorf("failed to create repository: %w", err)
			}
		}
	}

	// Authenticate with ECR
	authToken, err := client.GetAuthorizationToken(ctx, cfg.RegistryID, cfg.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization token: %w", err)
	}

	// Create Docker client
	docker := NewDockerClient()

	// Login to ECR
	if cfg.DryRun {
		fmt.Printf("[dry-run] Would login to ECR registry: %s\n", registryURL)
	} else {
		if err := docker.Login(ctx, registryURL, authToken); err != nil {
			return nil, fmt.Errorf("failed to login to ECR: %w", err)
		}
	}

	// Tag and push images
	pushedImages := []string{}
	for _, tag := range tags {
		targetImage := fmt.Sprintf("%s/%s:%s", registryURL, cfg.Repository, tag)

		if cfg.DryRun {
			fmt.Printf("[dry-run] Would tag %s as %s\n", cfg.SourceImage, targetImage)
			fmt.Printf("[dry-run] Would push %s\n", targetImage)
		} else {
			// Tag the image
			if err := docker.Tag(ctx, cfg.SourceImage, targetImage); err != nil {
				return nil, fmt.Errorf("failed to tag image: %w", err)
			}

			// Push the image
			if err := docker.Push(ctx, targetImage); err != nil {
				return nil, fmt.Errorf("failed to push image: %w", err)
			}

			fmt.Printf("Pushed: %s\n", targetImage)
		}

		pushedImages = append(pushedImages, targetImage)
	}

	return &plugin.ExecuteResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully pushed %d image(s) to ECR", len(pushedImages)),
		Outputs: map[string]any{
			"registry":      registryURL,
			"repository":    cfg.Repository,
			"tags":          tags,
			"pushed_images": pushedImages,
		},
	}, nil
}

// parseConfig parses the raw configuration into a Config struct.
func (p *ECRPlugin) parseConfig(raw map[string]any) *Config {
	parser := helpers.NewConfigParser(raw)

	tags := parser.GetStringSlice("tags", nil)
	if len(tags) == 0 {
		// Default to version tag
		tags = []string{"{{.Version}}"}
	}

	return &Config{
		// AWS Configuration
		Region:          parser.GetString("region", "AWS_REGION", ""),
		AccessKeyID:     parser.GetString("access_key_id", "AWS_ACCESS_KEY_ID", ""),
		SecretAccessKey: parser.GetString("secret_access_key", "AWS_SECRET_ACCESS_KEY", ""),
		SessionToken:    parser.GetString("session_token", "AWS_SESSION_TOKEN", ""),
		Profile:         parser.GetString("profile", "AWS_PROFILE", ""),
		RoleARN:         parser.GetString("role_arn", "AWS_ROLE_ARN", ""),

		// ECR Configuration
		RegistryID: parser.GetString("registry_id", "AWS_ACCOUNT_ID", ""),
		Repository: parser.GetString("repository", "ECR_REPOSITORY", ""),
		Public:     parser.GetBool("public", false),

		// Image Configuration
		SourceImage: parser.GetString("source_image", "", ""),
		Tags:        tags,

		// Behavior
		CreateRepo: parser.GetBool("create_repository", false),
		DryRun:     parser.GetBool("dry_run", false),
	}
}

// processTags processes tag templates with release context.
func (p *ECRPlugin) processTags(tags []string, ctx *plugin.ReleaseContext) []string {
	processed := make([]string, 0, len(tags))

	for _, tag := range tags {
		processed = append(processed, p.processTemplate(tag, ctx))
	}

	return processed
}

// processTemplate replaces template variables with actual values.
func (p *ECRPlugin) processTemplate(tmpl string, ctx *plugin.ReleaseContext) string {
	result := tmpl

	// Replace common template variables
	result = strings.ReplaceAll(result, "{{.Version}}", ctx.Version)
	result = strings.ReplaceAll(result, "{{.PreviousVersion}}", ctx.PreviousVersion)
	result = strings.ReplaceAll(result, "{{.TagName}}", ctx.TagName)
	result = strings.ReplaceAll(result, "{{.ReleaseType}}", ctx.ReleaseType)

	// Handle branch name
	if ctx.Branch != "" {
		// Sanitize branch name for Docker tag (replace / with -)
		safeBranch := strings.ReplaceAll(ctx.Branch, "/", "-")
		result = strings.ReplaceAll(result, "{{.Branch}}", safeBranch)
	}

	return result
}
