package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/katungi/aws-terror/pkg/progress"
	"github.com/zclconf/go-cty/cty"

	"encoding/json"

	tfjson "github.com/hashicorp/terraform-json"
)

func ParseStateFile(filepath, instanceID string) (map[string]any, error) {
	if filepath == "" || instanceID == "" {
		return nil, fmt.Errorf("filepath and instanceID must not be empty")
	}

	// Initialize progress spinner
	s := progress.NewSpinner("Parsing Terraform state file")
	s.Start()
	defer s.Stop()

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	// Parse raw JSON first to handle the actual state file structure
	var rawState map[string]any
	if err := json.NewDecoder(file).Decode(&rawState); err != nil {
		s.Error(fmt.Sprintf("Failed to parse state file: %v", err))
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	s.UpdateMessage("Analyzing state file contents")

	// Navigate through the state structure
	resources, ok := rawState["resources"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid state file: resources not found or invalid format")
	}

	// Iterate through resources to find the matching instance
	for _, res := range resources {
		resource, ok := res.(map[string]any)
		if !ok {
			continue
		}

		// Check if this is an AWS instance resource
		if resourceType, ok := resource["type"].(string); ok && resourceType == "aws_instance" {
			// Check the instances array
			instances, ok := resource["instances"].([]any)
			if !ok || len(instances) == 0 {
				continue
			}

			// Look at the first instance (usually there's only one)
			instance, ok := instances[0].(map[string]any)
			if !ok {
				continue
			}

			// Get the attributes
			attributes, ok := instance["attributes"].(map[string]any)
			if !ok {
				continue
			}

			// Check if this is the instance we're looking for
			if id, ok := attributes["id"].(string); ok && id == instanceID {
				s.Success("Successfully parsed Terraform state file")
	return attributes, nil
			}
		}
	}

	s.Error(fmt.Sprintf("Instance %s not found in Terraform state", instanceID))
	return nil, fmt.Errorf("instance %s not found in Terraform state", instanceID)
}

func findResourceInModule(module *tfjson.StateModule, instanceID string) map[string]any {
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

func ParseHCLConfig(configPath, instanceID string) (map[string]any, error) {
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
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				configFiles = append(configFiles, absPath)
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
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		configFiles = []string{absPath}
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

func extractInstanceConfig(parser *hclparse.Parser, instanceID string) (map[string]any, error) {
	fmt.Println("....Parsing......")
	if parser == nil || instanceID == "" {
		return nil, fmt.Errorf("parser and instanceID must not be nil")
	}

	config := make(map[string]any)

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
			continue // Skip files with parsing errors instead of failing
		}

		// Look for aws_instance resources
		for _, block := range content.Blocks {
			if block.Type == "resource" && len(block.Labels) >= 2 && block.Labels[0] == "aws_instance" {
				// Get the instance attributes
				attrs, diags := block.Body.JustAttributes()
				if diags.HasErrors() {
					continue
				}

				// Check for instance ID in the id attribute
				if idAttr, exists := attrs["id"]; exists {
					idVal, diags := idAttr.Expr.Value(nil)
					if !diags.HasErrors() && idVal.Type() == cty.String && idVal.AsString() == instanceID {
						// Found matching instance, extract all attributes
						for name, attr := range attrs {
							val, diags := attr.Expr.Value(nil)
							if !diags.HasErrors() {
								switch {
								case val.Type() == cty.String:
									config[name] = val.AsString()
								case val.Type().IsMapType() || val.Type().IsObjectType():
									tagsMap := make(map[string]interface{})
									if val.Type().IsMapType() {
										for k, v := range val.AsValueMap() {
											tagsMap[k] = v.AsString()
										}
									} else {
										for k, v := range val.AsValueMap() {
											tagsMap[k] = v.AsString()
										}
									}
									config[name] = tagsMap
								case val.Type() == cty.Number:
									config[name] = val.AsBigFloat()
								case val.Type() == cty.Bool:
									config[name] = val.True()
								default:
									config[name] = val.AsString()
								}
							}
						}
						return config, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("instance %s not found in Terraform configuration", instanceID)
}
