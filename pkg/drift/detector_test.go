package drift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectDrift_NoDrift(t *testing.T) {
	awsConfig := map[string]any{
		"instance_type": "t2.micro",
		"ami":           "ami-12345",
		"tags": map[string]string{
			"Name":        "test-instance",
			"Environment": "dev",
		},
	}

	tfConfig := map[string]interface{}{
		"instance_type": "t2.micro",
		"ami":           "ami-12345",
		"tags": map[string]any{
			"Name":        "test-instance",
			"Environment": "dev",
		},
	}

	attributesToCheck := []string{"instance_type", "ami", "tags"}

	drifts, err := DetectDrift(awsConfig, tfConfig, attributesToCheck)
	
	assert.NoError(t, err)
	assert.Empty(t, drifts, "Expected no drift")
}

func TestDetectDrift_WithDrift(t *testing.T) {
	awsConfig := map[string]any{
		"instance_type": "t2.micro",
		"ami":           "ami-12345",
		"tags": map[string]string{
			"Name":        "test-instance",
			"Environment": "dev",
		},
	}

	tfConfig := map[string]any{
		"instance_type": "t2.small", 
		"ami":           "ami-12345",
		"tags": map[string]any{
			"Name":        "test-instance",
			"Environment": "production", 
		},
	}

	attributesToCheck := []string{"instance_type", "ami", "tags"}

	drifts, err := DetectDrift(awsConfig, tfConfig, attributesToCheck)
	
	assert.NoError(t, err)
	assert.Len(t, drifts, 2, "Expected drift in 2 attributes")
	assert.Contains(t, drifts, "instance_type")
	assert.Contains(t, drifts, "tags")
	assert.NotContains(t, drifts, "ami")
}

func TestDetectDrift_AttributeInAWSOnly(t *testing.T) {
	awsConfig := map[string]interface{}{
		"instance_type": "t2.micro",
		"ami":           "ami-12345",
	}

	tfConfig := map[string]interface{}{
		"instance_type": "t2.micro",
		// ami is missing
	}

	attributesToCheck := []string{"instance_type", "ami"}

	drifts, err := DetectDrift(awsConfig, tfConfig, attributesToCheck)
	
	assert.NoError(t, err)
	assert.Len(t, drifts, 1, "Expected drift in 1 attribute")
	assert.Contains(t, drifts, "ami")
	
	driftDetail := drifts["ami"]
	assert.True(t, driftDetail.InAWS)
	assert.False(t, driftDetail.InTerraform)
	assert.Equal(t, "ami-12345", driftDetail.AWSValue)
}

func TestDetectDrift_AttributeInTerraformOnly(t *testing.T) {
	awsConfig := map[string]any{
		"instance_type": "t2.micro",
		// ami is missing
	}

	tfConfig := map[string]any{
		"instance_type": "t2.micro",
		"ami":           "ami-12345",
	}

	attributesToCheck := []string{"instance_type", "ami"}

	drifts, err := DetectDrift(awsConfig, tfConfig, attributesToCheck)
	
	assert.NoError(t, err)
	assert.Len(t, drifts, 1, "Expected drift in 1 attribute")
	assert.Contains(t, drifts, "ami")
	
	driftDetail := drifts["ami"]
	assert.False(t, driftDetail.InAWS)
	assert.True(t, driftDetail.InTerraform)
	assert.Equal(t, "ami-12345", driftDetail.TerraformValue)
}

func TestGetNestedValue(t *testing.T) {
	data := map[string]any{
		"instance_type": "t2.micro",
		"tags": map[string]any{
			"Name": "test-instance",
			"Nested": map[string]any{
				"Key": "Value",
			},
		},
		"block_device": []any{
			map[string]any{
				"device_name": "/dev/sda1",
				"volume_size": 8,
			},
		},
	}

	// Test simple value
	val, exists := getNestedValue(data, "instance_type")
	assert.True(t, exists)
	assert.Equal(t, "t2.micro", val)

	// Test nested value
	val, exists = getNestedValue(data, "tags.Name")
	assert.True(t, exists)
	assert.Equal(t, "test-instance", val)

	// Test deeply nested value
	val, exists = getNestedValue(data, "tags.Nested.Key")
	assert.True(t, exists)
	assert.Equal(t, "Value", val)

	// Test non-existent value
	val, exists = getNestedValue(data, "non_existent")
	assert.False(t, exists)
	assert.Nil(t, val)

	// Test non-existent nested value
	val, exists = getNestedValue(data, "tags.non_existent")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestCompareValues(t *testing.T) {
	// Test simple values
	assert.True(t, compareValues("value", "value"))
	assert.False(t, compareValues("value1", "value2"))
	
	// Test numeric values
	assert.True(t, compareValues(10, 10))
	assert.True(t, compareValues(10, 10.0)) // Different types but same value
	assert.False(t, compareValues(10, 20))
	
	// Test nil values
	assert.True(t, compareValues(nil, nil))
	assert.False(t, compareValues(nil, "value"))
	assert.False(t, compareValues("value", nil))
	
	// Test maps
	map1 := map[string]any{"key1": "value1", "key2": "value2"}
	map2 := map[string]any{"key1": "value1", "key2": "value2"}
	map3 := map[string]any{"key1": "value1", "key2": "different"}
	
	assert.True(t, compareValues(map1, map2))
	assert.False(t, compareValues(map1, map3))
	
	// Test slices
	slice1 := []any{"a", "b", "c"}
	slice2 := []any{"c", "b", "a"} // Different order but same elements
	slice3 := []any{"a", "b", "d"}
	
	assert.True(t, compareValues(slice1, slice2))
	assert.False(t, compareValues(slice1, slice3))
}