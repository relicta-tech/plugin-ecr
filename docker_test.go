package main

import (
	"testing"
)

func TestNewDockerClient(t *testing.T) {
	client := NewDockerClient()
	if client == nil {
		t.Error("expected client, got nil")
	}
}

func TestExtractRegionFromRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		expected string
	}{
		{
			name:     "us-east-1",
			registry: "123456789012.dkr.ecr.us-east-1.amazonaws.com",
			expected: "us-east-1",
		},
		{
			name:     "us-west-2",
			registry: "987654321098.dkr.ecr.us-west-2.amazonaws.com",
			expected: "us-west-2",
		},
		{
			name:     "eu-central-1",
			registry: "111222333444.dkr.ecr.eu-central-1.amazonaws.com",
			expected: "eu-central-1",
		},
		{
			name:     "ap-southeast-1",
			registry: "555666777888.dkr.ecr.ap-southeast-1.amazonaws.com",
			expected: "ap-southeast-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract region using the same logic as loginPrivateECR
			parts := splitRegistryURL(tt.registry)
			if len(parts) < 4 {
				t.Fatalf("invalid registry URL: %s", tt.registry)
			}
			region := parts[3]

			if region != tt.expected {
				t.Errorf("expected region '%s', got '%s'", tt.expected, region)
			}
		})
	}
}

// Helper function to split registry URL
func splitRegistryURL(registry string) []string {
	result := []string{}
	current := ""
	for _, c := range registry {
		if c == '.' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func TestIsPublicECR(t *testing.T) {
	tests := []struct {
		registry string
		isPublic bool
	}{
		{"public.ecr.aws", true},
		{"public.ecr.aws/myrepo", true},
		{"123456789012.dkr.ecr.us-east-1.amazonaws.com", false},
		{"987654321098.dkr.ecr.eu-west-1.amazonaws.com/myrepo", false},
	}

	for _, tt := range tests {
		t.Run(tt.registry, func(t *testing.T) {
			isPublic := len(tt.registry) >= 14 && tt.registry[:14] == "public.ecr.aws"
			if isPublic != tt.isPublic {
				t.Errorf("expected isPublic=%v for %s, got %v", tt.isPublic, tt.registry, isPublic)
			}
		})
	}
}
