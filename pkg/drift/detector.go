package drift

import (
	"fmt"
	"reflect"
	"strings"
)

// DetectDrift compares AWS and Terraform configurations and returns differences
func DetectDrift(awsConfig, tfConfig map[string]any, attributesToCheck []string) (map[string]DriftDetail, error) {
	drifts := make(map[string]DriftDetail)

	for _, attr := range attributesToCheck {
		awsValue, awsExists := getNestedValue(awsConfig, attr)
		tfValue, tfExists := getNestedValue(tfConfig, attr)

		if !awsExists && !tfExists {
			continue
		}

		if !awsExists {
			drifts[attr] = DriftDetail{
				Attribute:      attr,
				InAWS:          false,
				InTerraform:    true,
				TerraformValue: tfValue,
			}
			continue
		}

		if !tfExists {
			drifts[attr] = DriftDetail{
				Attribute:   attr,
				InAWS:       true,
				InTerraform: false,
				AWSValue:    awsValue,
			}
			continue
		}

		if !compareValues(awsValue, tfValue) {
			drifts[attr] = DriftDetail{
				Attribute:      attr,
				InAWS:          true,
				InTerraform:    true,
				AWSValue:       awsValue,
				TerraformValue: tfValue,
			}
		}
	}

	return drifts, nil
}

type DriftDetail struct {
	Attribute      string
	InAWS          bool
	InTerraform    bool
	AWSValue       any
	TerraformValue any
}

func getNestedValue(data map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := data

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		val, ok := current[part]
		if !ok {
			return nil, false
		}

		next, ok := val.(map[string]any)
		if !ok {
			if strMap, isStrMap := val.(map[string]string); isStrMap {
				next = make(map[string]any)
				for k, v := range strMap {
					next[k] = v
				}
			} else {
				return nil, false
			}
		}

		current = next
	}

	lastPart := parts[len(parts)-1]
	val, ok := current[lastPart]
	return val, ok
}

func compareValues(v1, v2 any) bool {
	if v1 == nil && v2 == nil {
		return true
	}
	if v1 == nil || v2 == nil {
		return false
	}

	v1 = normalizeValue(v1)
	v2 = normalizeValue(v2)

	m1, isMap1 := v1.(map[string]any)
	m2, isMap2 := v2.(map[string]any)

	if isMap1 && isMap2 {
		return compareMaps(m1, m2)
	}

	s1, isSlice1 := v1.([]any)
	s2, isSlice2 := v2.([]any)

	if isSlice1 && isSlice2 {
		return compareSlices(s1, s2)
	}

	return reflect.DeepEqual(v1, v2)
}

func normalizeValue(v any) any {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case uint:
		return float64(val)
	case uint32:
		return float64(val)
	case uint64:
		return float64(val)
	case map[string]string:

		result := make(map[string]any)
		for k, v := range val {
			result[k] = v
		}
		return result
	case []string:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = v
		}
		return result
	default:
		return val
	}
}

func compareMaps(m1, m2 map[string]any) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok {
			return false
		}

		if !compareValues(v1, v2) {
			return false
		}
	}

	return true
}

func compareSlices(s1, s2 []any) bool {
	if len(s1) != len(s2) {
		return false
	}

	s2Copy := make([]any, len(s2))
	copy(s2Copy, s2)

	for _, v1 := range s1 {
		found := false
		for i, v2 := range s2Copy {
			if compareValues(v1, v2) {
				s2Copy[i] = nil
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func (d DriftDetail) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Attribute: %s\n", d.Attribute))

	if d.InAWS && d.InTerraform {
		sb.WriteString("Status: Values differ between AWS and Terraform\n")
		sb.WriteString(fmt.Sprintf("AWS value: %v\n", d.AWSValue))
		sb.WriteString(fmt.Sprintf("Terraform value: %v\n", d.TerraformValue))
	} else if d.InAWS {
		sb.WriteString("Status: Exists in AWS but not in Terraform\n")
		sb.WriteString(fmt.Sprintf("AWS value: %v\n", d.AWSValue))
	} else {
		sb.WriteString("Status: Exists in Terraform but not in AWS\n")
		sb.WriteString(fmt.Sprintf("Terraform value: %v\n", d.TerraformValue))
	}

	return sb.String()
}
