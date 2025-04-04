package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/katungi/aws-terror/pkg/drift"
	"github.com/stretchr/testify/assert"
)

func TestFormatDriftResults_TextFormat(t *testing.T) {
	// Create test drift data
	drifts := map[string]drift.DriftDetail{
		"instance_type": {
			Attribute:      "instance_type",
			InAWS:          true,
			InTerraform:    true,
			AWSValue:       "t2.micro",
			TerraformValue: "t2.small",
		},
		"tags": {
			Attribute:      "tags",
			InAWS:          true,
			InTerraform:    true,
			AWSValue:       map[string]string{"Name": "test-instance", "Environment": "dev"},
			TerraformValue: map[string]any{"Name": "test-instance", "Environment": "prod"},
		},
	}

	result := FormatDriftResults(drifts, "i-12345", "text")
	
	// Verify text format output
	assert.Contains(t, result, "Drift Detection Results for EC2 Instance: i-12345")
	assert.Contains(t, result, "Found 2 attributes with configuration drift")
	assert.Contains(t, result, "--- instance_type ---")
	assert.Contains(t, result, "AWS value: t2.micro")
	assert.Contains(t, result, "Terraform value: t2.small")
	assert.Contains(t, result, "--- tags ---")
}

func TestFormatDriftResults_JsonFormat(t *testing.T) {
	// Create test drift data
	drifts := map[string]drift.DriftDetail{
		"instance_type": {
			Attribute:      "instance_type",
			InAWS:          true,
			InTerraform:    true,
			AWSValue:       "t2.micro",
			TerraformValue: "t2.small",
		},
	}

	result := FormatDriftResults(drifts, "i-12345", "json")
	
	// Verify JSON format output
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(result), &jsonData)
	
	assert.NoError(t, err, "Should be valid JSON")
	assert.Equal(t, "i-12345", jsonData["instance_id"])
	assert.Equal(t, true, jsonData["drift_found"])
	assert.Equal(t, float64(1), jsonData["drift_count"])
}

func TestFormatDriftResults_YamlFormat(t *testing.T) {
	// Create test drift data
	drifts := map[string]drift.DriftDetail{
		"instance_type": {
			Attribute:      "instance_type",
			InAWS:          true,
			InTerraform:    true,
			AWSValue:       "t2.micro",
			TerraformValue: "t2.small",
		},
	}

	result := FormatDriftResults(drifts, "i-12345", "yaml")
	
	// Verify YAML format output
	assert.True(t, strings.Contains(result, "instance_id: i-12345"))
	assert.True(t, strings.Contains(result, "drift_found: true"))
	assert.True(t, strings.Contains(result, "drift_count: 1"))
	assert.True(t, strings.Contains(result, "  instance_type:"))
	assert.True(t, strings.Contains(result, "    in_aws: true"))
	assert.True(t, strings.Contains(result, "    in_terraform: true"))
	assert.True(t, strings.Contains(result, "    aws_value: \"t2.micro\""))
	assert.True(t, strings.Contains(result, "    terraform_value: \"t2.small\""))
}

func TestFormatDriftResults_NoDrift(t *testing.T) {
	// Test with empty drifts map
	drifts := map[string]drift.DriftDetail{}

	result := FormatDriftResults(drifts, "i-12345", "text")
	
	assert.Contains(t, result, "No configuration drift detected")
}