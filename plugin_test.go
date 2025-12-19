package main

import (
	"context"
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

func TestGetInfo(t *testing.T) {
	p := &ECRPlugin{}
	info := p.GetInfo()

	if info.Name != "ecr" {
		t.Errorf("expected name 'ecr', got '%s'", info.Name)
	}

	if info.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(info.Hooks) == 0 {
		t.Error("expected at least one hook")
	}

	// Should have PostPublish hook
	hasPostPublish := false
	for _, hook := range info.Hooks {
		if hook == plugin.HookPostPublish {
			hasPostPublish = true
			break
		}
	}
	if !hasPostPublish {
		t.Error("expected PostPublish hook")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		config     map[string]any
		wantErrors int
	}{
		{
			name:       "empty config",
			config:     map[string]any{},
			wantErrors: 3, // region, repository, source_image
		},
		{
			name: "missing region",
			config: map[string]any{
				"repository":   "my-repo",
				"source_image": "myapp:latest",
				"tags":         []string{"v1.0.0"},
			},
			wantErrors: 1,
		},
		{
			name: "missing repository",
			config: map[string]any{
				"region":       "us-east-1",
				"source_image": "myapp:latest",
				"tags":         []string{"v1.0.0"},
			},
			wantErrors: 1,
		},
		{
			name: "missing source image",
			config: map[string]any{
				"region":     "us-east-1",
				"repository": "my-repo",
				"tags":       []string{"v1.0.0"},
			},
			wantErrors: 1,
		},
		{
			name: "empty tags with default",
			config: map[string]any{
				"region":       "us-east-1",
				"repository":   "my-repo",
				"source_image": "myapp:latest",
				"tags":         []string{},
			},
			wantErrors: 0, // Empty tags get default {{.Version}}
		},
		{
			name: "empty tag in list",
			config: map[string]any{
				"region":       "us-east-1",
				"repository":   "my-repo",
				"source_image": "myapp:latest",
				"tags":         []string{"v1.0.0", ""},
			},
			wantErrors: 1,
		},
		{
			name: "valid config",
			config: map[string]any{
				"region":       "us-east-1",
				"repository":   "my-repo",
				"source_image": "myapp:latest",
				"tags":         []string{"v1.0.0", "latest"},
			},
			wantErrors: 0,
		},
		{
			name: "valid config with credentials",
			config: map[string]any{
				"region":            "us-west-2",
				"repository":        "my-app",
				"source_image":      "myapp:v1.0.0",
				"tags":              []string{"{{.Version}}", "latest"},
				"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
				"secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErrors: 0,
		},
		{
			name: "valid public ECR config",
			config: map[string]any{
				"region":       "us-east-1",
				"repository":   "my-public-repo",
				"source_image": "myapp:latest",
				"tags":         []string{"v1.0.0"},
				"public":       true,
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ECRPlugin{}
			resp, err := p.Validate(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.Errors) != tt.wantErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrors, len(resp.Errors), resp.Errors)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	p := &ECRPlugin{}

	config := map[string]any{
		"region":            "us-west-2",
		"repository":        "my-app",
		"source_image":      "myapp:v1.0.0",
		"tags":              []string{"v1.0.0", "latest"},
		"registry_id":       "123456789012",
		"public":            false,
		"create_repository": true,
		"dry_run":           true,
	}

	cfg := p.parseConfig(config)

	if cfg.Region != "us-west-2" {
		t.Errorf("expected region 'us-west-2', got '%s'", cfg.Region)
	}

	if cfg.Repository != "my-app" {
		t.Errorf("expected repository 'my-app', got '%s'", cfg.Repository)
	}

	if cfg.SourceImage != "myapp:v1.0.0" {
		t.Errorf("expected source_image 'myapp:v1.0.0', got '%s'", cfg.SourceImage)
	}

	if len(cfg.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(cfg.Tags))
	}

	if cfg.RegistryID != "123456789012" {
		t.Errorf("expected registry_id '123456789012', got '%s'", cfg.RegistryID)
	}

	if cfg.Public {
		t.Error("expected public to be false")
	}

	if !cfg.CreateRepo {
		t.Error("expected create_repository to be true")
	}

	if !cfg.DryRun {
		t.Error("expected dry_run to be true")
	}
}

func TestParseConfigDefaults(t *testing.T) {
	p := &ECRPlugin{}

	cfg := p.parseConfig(map[string]any{})

	if cfg.Region != "" {
		t.Errorf("expected empty region, got '%s'", cfg.Region)
	}

	if cfg.Public {
		t.Error("expected public to default to false")
	}

	if cfg.CreateRepo {
		t.Error("expected create_repository to default to false")
	}

	if cfg.DryRun {
		t.Error("expected dry_run to default to false")
	}

	// Default tag should be version template
	if len(cfg.Tags) != 1 || cfg.Tags[0] != "{{.Version}}" {
		t.Errorf("expected default tag ['{{.Version}}'], got %v", cfg.Tags)
	}
}

func TestProcessTags(t *testing.T) {
	p := &ECRPlugin{}

	ctx := &plugin.ReleaseContext{
		Version:         "1.2.3",
		PreviousVersion: "1.2.2",
		TagName:         "v1.2.3",
		Branch:          "feature/new-feature",
		ReleaseType:     "minor",
	}

	tests := []struct {
		name     string
		tags     []string
		expected []string
	}{
		{
			name:     "version tag",
			tags:     []string{"{{.Version}}"},
			expected: []string{"1.2.3"},
		},
		{
			name:     "multiple tags",
			tags:     []string{"{{.Version}}", "latest", "{{.TagName}}"},
			expected: []string{"1.2.3", "latest", "v1.2.3"},
		},
		{
			name:     "release type tag",
			tags:     []string{"{{.ReleaseType}}-{{.Version}}"},
			expected: []string{"minor-1.2.3"},
		},
		{
			name:     "branch tag",
			tags:     []string{"{{.Branch}}"},
			expected: []string{"feature-new-feature"}, // / replaced with -
		},
		{
			name:     "previous version",
			tags:     []string{"prev-{{.PreviousVersion}}"},
			expected: []string{"prev-1.2.2"},
		},
		{
			name:     "literal tag",
			tags:     []string{"stable"},
			expected: []string{"stable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.processTags(tt.tags, ctx)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d", len(tt.expected), len(result))
				return
			}

			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("tag[%d]: expected '%s', got '%s'", i, tt.expected[i], tag)
				}
			}
		})
	}
}

func TestProcessTemplate(t *testing.T) {
	p := &ECRPlugin{}

	tests := []struct {
		name     string
		template string
		ctx      *plugin.ReleaseContext
		expected string
	}{
		{
			name:     "version only",
			template: "{{.Version}}",
			ctx:      &plugin.ReleaseContext{Version: "2.0.0"},
			expected: "2.0.0",
		},
		{
			name:     "mixed template",
			template: "v{{.Version}}-{{.TagName}}",
			ctx:      &plugin.ReleaseContext{Version: "1.0.0", TagName: "v1.0.0"},
			expected: "v1.0.0-v1.0.0",
		},
		{
			name:     "no placeholders",
			template: "latest",
			ctx:      &plugin.ReleaseContext{Version: "1.0.0"},
			expected: "latest",
		},
		{
			name:     "release type",
			template: "type-{{.ReleaseType}}",
			ctx:      &plugin.ReleaseContext{ReleaseType: "patch"},
			expected: "type-patch",
		},
		{
			name:     "branch with slashes",
			template: "{{.Branch}}",
			ctx:      &plugin.ReleaseContext{Branch: "feature/foo/bar"},
			expected: "feature-foo-bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.processTemplate(tt.template, tt.ctx)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
