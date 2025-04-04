package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/katungi/aws-terror/pkg/drift"
)

func FormatDriftResults(drifts map[string]drift.DriftDetail, instanceID, format string) string {
	switch strings.ToLower(format) {
	case "json":
		return formatJSON(drifts, instanceID)
	case "yaml":
		return formatYAML(drifts, instanceID)
	default:
		return formatText(drifts, instanceID)
	}
}

func formatText(drifts map[string]drift.DriftDetail, instanceID string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Drift Detection Results for EC2 Instance: %s\n\n", instanceID))

	if len(drifts) == 0 {
		sb.WriteString("No configuration drift detected! AWS and Terraform configurations are in sync.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found %d attributes with configuration drift:\n\n", len(drifts)))

	for _, detail := range drifts {
		sb.WriteString(fmt.Sprintf("--- %s ---\n", detail.Attribute))

		if detail.InAWS && detail.InTerraform {
			sb.WriteString("Status: Values differ between AWS and Terraform\n")
			sb.WriteString(fmt.Sprintf("AWS value: %v\n", detail.AWSValue))
			sb.WriteString(fmt.Sprintf("Terraform value: %v\n", detail.TerraformValue))
		} else if detail.InAWS {
			sb.WriteString("Status: Exists in AWS but not in Terraform\n")
			sb.WriteString(fmt.Sprintf("AWS value: %v\n", detail.AWSValue))
		} else {
			sb.WriteString("Status: Exists in Terraform but not in AWS\n")
			sb.WriteString(fmt.Sprintf("Terraform value: %v\n", detail.TerraformValue))
		}

		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\nDetection completed at: %s\n", time.Now().Format(time.RFC1123)))
	return sb.String()
}

func formatJSON(drifts map[string]drift.DriftDetail, instanceID string) string {
	type jsonResult struct {
		InstanceID   string                       `json:"instance_id"`
		DriftFound   bool                         `json:"drift_found"`
		DriftCount   int                          `json:"drift_count"`
		Drifts       map[string]drift.DriftDetail `json:"drifts"`
		TimeDetected string                       `json:"time_detected"`
	}

	result := jsonResult{
		InstanceID:   instanceID,
		DriftFound:   len(drifts) > 0,
		DriftCount:   len(drifts),
		Drifts:       drifts,
		TimeDetected: time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}

	return string(jsonData)
}

func formatYAML(drifts map[string]drift.DriftDetail, instanceID string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("instance_id: %s\n", instanceID))
	sb.WriteString(fmt.Sprintf("drift_found: %t\n", len(drifts) > 0))
	sb.WriteString(fmt.Sprintf("drift_count: %d\n", len(drifts)))
	sb.WriteString(fmt.Sprintf("time_detected: %s\n", time.Now().Format(time.RFC3339)))

	if len(drifts) > 0 {
		sb.WriteString("drifts:\n")

		for attr, detail := range drifts {
			sb.WriteString(fmt.Sprintf("  %s:\n", attr))
			sb.WriteString(fmt.Sprintf("    in_aws: %t\n", detail.InAWS))
			sb.WriteString(fmt.Sprintf("    in_terraform: %t\n", detail.InTerraform))

			if detail.InAWS {
				sb.WriteString(fmt.Sprintf("    aws_value: \"%v\"\n", detail.AWSValue))
			}

			if detail.InTerraform {
				sb.WriteString(fmt.Sprintf("    terraform_value: \"%v\"\n", detail.TerraformValue))
			}
		}
	}

	return sb.String()
}
