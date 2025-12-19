package main

import (
	"context"
	"testing"
)

func TestNewECRClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *AWSConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AWSConfig{
				Region: "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "missing region",
			config: &AWSConfig{
				Region: "",
			},
			wantErr: true,
		},
		{
			name: "full config",
			config: &AWSConfig{
				Region:          "us-west-2",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "token",
				Profile:         "default",
				RoleARN:         "arn:aws:iam::123456789012:role/MyRole",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewECRClient(context.Background(), tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("expected client, got nil")
			}
		})
	}
}

func TestGetRegistryURL(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		registryID string
		public     bool
		expected   string
		wantErr    bool
	}{
		{
			name:       "public ECR",
			region:     "us-east-1",
			registryID: "",
			public:     true,
			expected:   "public.ecr.aws",
			wantErr:    false,
		},
		{
			name:       "private ECR with registry ID",
			region:     "us-west-2",
			registryID: "123456789012",
			public:     false,
			expected:   "123456789012.dkr.ecr.us-west-2.amazonaws.com",
			wantErr:    false,
		},
		{
			name:       "private ECR without registry ID",
			region:     "eu-west-1",
			registryID: "",
			public:     false,
			expected:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &ECRClient{
				config: &AWSConfig{
					Region: tt.region,
				},
			}

			url, err := client.GetRegistryURL(context.Background(), tt.registryID, tt.public)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if url != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, url)
			}
		})
	}
}

func TestGetAuthorizationToken(t *testing.T) {
	client := &ECRClient{
		config: &AWSConfig{
			Region: "us-east-1",
		},
	}

	token, err := client.GetAuthorizationToken(context.Background(), "123456789012", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == nil {
		t.Fatal("expected token, got nil")
	}

	if token.Username != "AWS" {
		t.Errorf("expected username 'AWS', got '%s'", token.Username)
	}
}

func TestDecodeAuthToken(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "valid token",
			token:        "QVdTOnBhc3N3b3JkMTIz", // base64("AWS:password123")
			wantUsername: "AWS",
			wantPassword: "password123",
			wantErr:      false,
		},
		{
			name:    "invalid base64",
			token:   "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "missing colon",
			token:   "QVdTcGFzc3dvcmQ=", // base64("AWSpassword")
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, password, err := DecodeAuthToken(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if username != tt.wantUsername {
				t.Errorf("expected username '%s', got '%s'", tt.wantUsername, username)
			}

			if password != tt.wantPassword {
				t.Errorf("expected password '%s', got '%s'", tt.wantPassword, password)
			}
		})
	}
}

func TestCreateRepositoryIfNotExists(t *testing.T) {
	client := &ECRClient{
		config: &AWSConfig{
			Region: "us-east-1",
		},
	}

	// Should not error
	err := client.CreateRepositoryIfNotExists(context.Background(), "test-repo", false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = client.CreateRepositoryIfNotExists(context.Background(), "test-public-repo", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
