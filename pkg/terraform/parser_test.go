package terraform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseStateFile(t *testing.T) {
	// Create a temporary state file
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "terraform.tfstate")

	tests := []struct {
		name           string
		stateContent   map[string]any
		instanceID     string
		expectedConfig map[string]any
		expectError    bool
	}{
		{
			name: "Successfully parse state file",
			stateContent: map[string]any{
				"version": 4,
				"terraform_version": "1.0.0",
				"serial": 1,
				"lineage": "example",
				"outputs": map[string]any{},
				"resources": []any{
					map[string]any{
						"mode": "managed",
						"type": "aws_instance",
						"name": "test_instance",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": []any{
							map[string]any{
								"attributes": map[string]any{
									"id":            "i-1234567890abcdef0",
									"instance_type": "t2.micro",
									"ami":           "ami-123",
								},
							},
						},
					},
				},
			},
			instanceID: "i-1234567890abcdef0",
			expectedConfig: map[string]any{
				"id":            "i-1234567890abcdef0",
				"instance_type": "t2.micro",
				"ami":           "ami-123",
			},
		},
		{
			name: "Instance not found in state",
			stateContent: map[string]any{
				"version": 4,
				"terraform_version": "1.0.0",
				"serial": 1,
				"lineage": "example",
				"outputs": map[string]any{},
				"resources": []any{},
			},
			instanceID:  "i-nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write state file
			stateData, err := json.Marshal(tt.stateContent)
			if err != nil {
				t.Fatalf("failed to marshal state: %v", err)
			}
			if err := os.WriteFile(statePath, stateData, 0644); err != nil {
				t.Fatalf("failed to write state file: %v", err)
			}

			// Test ParseStateFile
			config, err := ParseStateFile(statePath, tt.instanceID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Compare configurations
			for key, expectedValue := range tt.expectedConfig {
				actualValue, ok := config[key]
				if !ok {
					t.Errorf("expected key %s not found in config", key)
					continue
				}

				if !reflect.DeepEqual(expectedValue, actualValue) {
					t.Errorf("for key %s, expected %v but got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestParseHCLConfig(t *testing.T) {
	// Create a temporary directory for HCL files
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		hclContent     string
		instanceID     string
		expectedConfig map[string]any
		expectError    bool
	}{
		{
			name: "Successfully parse HCL config",
			hclContent: `
			resource "aws_instance" "test" {
				instance_type = "t2.micro"
				ami           = "ami-123"
				id            = "test-instance"
				tags = {
					Name = "test-instance"
				}
			}
			`,
			instanceID: "test-instance",
			expectedConfig: map[string]any{
				"instance_type": any("t2.micro"),
				"ami":           any("ami-123"),
				"id":            any("test-instance"),
				"tags":          map[string]any{"Name": any("test-instance")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write HCL file
			hclPath := filepath.Join(tmpDir, "main.tf")
			if err := os.WriteFile(hclPath, []byte(tt.hclContent), 0644); err != nil {
				t.Fatalf("failed to write HCL file: %v", err)
			}

			// Test ParseHCLConfig
			config, err := ParseHCLConfig(tmpDir, tt.instanceID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Compare configurations
			for key, expectedValue := range tt.expectedConfig {
				actualValue, ok := config[key]
				if !ok {
					t.Errorf("expected key %s not found in config", key)
					continue
				}

				if !reflect.DeepEqual(expectedValue, actualValue) {
					t.Errorf("for key %s, expected %v but got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}