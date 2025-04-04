/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/katungi/aws-terror/aws"
	"github.com/katungi/aws-terror/pkg/drift"
	"github.com/katungi/aws-terror/pkg/output"
	"github.com/katungi/aws-terror/pkg/terraform"
	"github.com/spf13/cobra"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect configuration drift between AWS and Terraform",
	Long: `Detect configuration drift between AWS EC2 instances and Terraform configurations.

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if instanceID == "" {
			logger.Fatal("Instance ID is required")
		}

		if tfStatePath == "" && tfConfigPath == "" {
			logger.Fatal("Either Terraform state file or HCL configuration path is required")
		}

		// Initialize AWS client
		awsClient, err := aws.NewClient(awsRegion, logger)
		if err != nil {
			logger.Fatalf("Failed to initialize AWS client: %v", err)
		}

		// Fetch EC2 instance configuration from AWS
		logger.Infof("Fetching EC2 instance %s configuration from AWS...", instanceID)
		awsConfig, err := awsClient.GetEC2InstanceConfig(instanceID)
		if err != nil {
			logger.Fatalf("Failed to get EC2 instance config: %v", err)
		}

		// Parse Terraform configuration
		var tfConfig map[string]interface{}
		if tfStatePath != "" {
			logger.Infof("Parsing Terraform state file: %s", tfStatePath)
			tfConfig, err = terraform.ParseStateFile(tfStatePath, instanceID)
		} else {
			logger.Infof("Parsing Terraform HCL configuration: %s", tfConfigPath)
			tfConfig, err = terraform.ParseHCLConfig(tfConfigPath, instanceID)
		}

		if err != nil {
			logger.Fatalf("Failed to parse Terraform configuration: %v", err)
		}

		// Detect drift
		logger.Info("Detecting configuration drift...")
		drifts, err := drift.DetectDrift(awsConfig, tfConfig, attributesToCheck)
		if err != nil {
			logger.Fatalf("Error detecting drift: %v", err)
		}

		// Output results
		result := output.FormatDriftResults(drifts, instanceID, outputFormat)
		fmt.Println(result)

		if len(drifts) > 0 {
			attributes := make([]string, 0, len(drifts))
			for attr := range drifts {
				attributes = append(attributes, attr)
			}
			logger.Warnf("Drift detected in %d attributes: %s", len(drifts), strings.Join(attributes, ", "))
		} else {
			logger.Info("No drift detected! AWS and Terraform configurations are in sync.")
		}
	},
	}

func init() {
	rootCmd.AddCommand(driftCmd)
	driftCmd.Flags().StringVarP(&instanceID, "instance", "i", "", "EC2 instance ID to check (required)")
	driftCmd.Flags().StringVarP(&tfStatePath, "state", "s", "", "Path to Terraform state file")
	driftCmd.Flags().StringVarP(&tfConfigPath, "config", "c", "", "Path to Terraform HCL configuration directory")
	driftCmd.Flags().StringSliceVarP(&attributesToCheck, "attributes", "a", attributesToCheck, "Attributes to check for drift (comma-separated)")

	driftCmd.MarkFlagRequired("instance")
}
