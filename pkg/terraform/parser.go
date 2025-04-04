package terraform

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"

	"encoding/json"

	tfjson "github.com/hashicorp/terraform-json"
)

func ParseStateFile(filepath, instanceID string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filepath)

	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state tfjson.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if state.Values == nil || state.Values.RootModule == nil {
		return nil, fmt.Errorf("invalid state file: no root module found")
	}

	// find the resource for the given instance ID
	for _, resource := range state.Values.RootModule.Resources {
		if resource.Type == "aws_instance" {
			if resource.AttributeValues != nil {
				if idVal, ok := resource.AttributeValues["id"].(string); ok && idVal == instanceID {
					return resource.AttributeValues, nil
				}
			}
		}
	}

	// check in child mods if not in root
	for _, module := range state.Values.RootModule.ChildModules {
		config := findResourceInModule(module, instanceID)
		if config != nil {
			return config, nil
		}
	}

	return nil, fmt.Errorf("instance %s not found in Terraform state", instanceID)

}

func findResourceInModule(module *tfjson.StateModule, instanceID string) map[string]interface{} {
	for _, resource := range module.Resources {
		if resource.Type == "aws_instance" {
			if resource.AttributeValues != nil {
				if idVal, ok := resource.AttributeValues["id"].(string); ok && idVal == instanceID {
					return resource.AttributeValues
				}
			}
		}
	}

	for _, childModule := range module.ChildModules {
		config := findResourceInModule(childModule, instanceID)
		if config != nil {
			return config
		}
	}

	return nil
}

// ParseHCLConfig parses Terraform HCL configuration files
func ParseHCLConfig(configPath, instanceID string) (map[string]interface{}, error) {
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config path: %w", err)
	}

	var configFiles []string
	if fileInfo.IsDir() {
		// Find all .tf files in the directory
		err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
				configFiles = append(configFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk config directory: %w", err)
		}
	} else {
		// Single file
		configFiles = []string{configPath}
	}

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no .tf files found in %s", configPath)
	}

	parser := hclparse.NewParser()
	for _, file := range configFiles {
		_, diags := parser.ParseHCLFile(file)
		if diags.HasErrors() {
			return nil, fmt.Errorf("failed to parse HCL file %s: %v", file, diags)
		}
	}

	// Extract instance configuration from parsed files
	config, err := extractInstanceConfig(parser, instanceID)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// extractInstanceConfig extracts configuration for a specific instance from parsed HCL
func extractInstanceConfig(parser *hclparse.Parser, instanceID string) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Get all parsed files
	files := parser.Files()
	if len(files) == 0 {
		return nil, fmt.Errorf("no parsed files available")
	}

	// Iterate through all parsed files
	for _, file := range files {
		body := file.Body
		if body == nil {
			continue
		}

		// Get content blocks
		content, diags := body.Content(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{
					Type:       "resource",
					LabelNames: []string{"type", "name"},
				},
			},
		})

		if diags.HasErrors() {
			return nil, fmt.Errorf("error getting content: %v", diags)
		}

		// Look for aws_instance resources
		for _, block := range content.Blocks {
			if block.Type == "resource" && len(block.Labels) >= 2 && block.Labels[0] == "aws_instance" {
				// Get the instance attributes
				attrs, diags := block.Body.JustAttributes()
				if diags.HasErrors() {
					continue
				}

				// Look for tags to match instance ID
				if tagsAttr, exists := attrs["tags"]; exists {
					tagsVal, diags := tagsAttr.Expr.Value(nil)
					if !diags.HasErrors() && tagsVal.Type().IsMapType() {
						tagsMap := tagsVal.AsValueMap()
						if idTag, ok := tagsMap["Name"]; ok && idTag.AsString() == instanceID {
							// Found matching instance, extract all attributes
							for name, attr := range attrs {
								val, diags := attr.Expr.Value(nil)
								if !diags.HasErrors() {
									config[name] = val.AsString()
								}
							}
							return config, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("instance %s not found in Terraform configuration", instanceID)
}
