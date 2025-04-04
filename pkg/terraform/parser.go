// Package terraform provides functionality for parsing and extracting information from
// Terraform state files and HCL configuration files.
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

// ParseStateFile reads and parses a Terraform state file to extract configuration
// for a specific EC2 instance identified by its instance ID.
// It returns the instance's configuration as a map or an error if the instance
// is not found or if there are any parsing issues.
func ParseStateFile(filepath, instanceID string) (map[string]interface{}, error) {
	if filepath == "" || instanceID == "" {
		return nil, fmt.Errorf("filepath and instanceID must not be empty")
	}

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

// findResourceInModule recursively searches for an EC2 instance resource in a Terraform module
// and its child modules. It returns the instance's configuration if found, or nil if not found.
func findResourceInModule(module *tfjson.StateModule, instanceID string) map[string]interface{} {
	if module == nil || instanceID == "" {
		return nil
	}

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

// ParseHCLConfig parses Terraform HCL configuration files and extracts configuration
// for a specific EC2 instance identified by its instance ID.
// It can handle both single .tf files and directories containing multiple .tf files.
func ParseHCLConfig(configPath, instanceID string) (map[string]interface{}, error) {
	if configPath == "" || instanceID == "" {
		return nil, fmt.Errorf("configPath and instanceID must not be empty")
	}

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
		if !strings.HasSuffix(configPath, ".tf") {
			return nil, fmt.Errorf("config file must have .tf extension")
		}
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
// by looking for aws_instance resources with matching tags.
func extractInstanceConfig(parser *hclparse.Parser, instanceID string) (map[string]interface{}, error) {
	if parser == nil || instanceID == "" {
		return nil, fmt.Errorf("parser and instanceID must not be nil")
	}

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
