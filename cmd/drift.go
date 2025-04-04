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

		// Create channels for results and errors
		resultsChan := make(chan struct {
			instanceID string
			drifts     map[string]drift.DriftDetail
			err        error
		}, len(instanceIDs))

		// Process instances concurrently with worker pool
		workerPool := make(chan struct{}, maxConcurrency)
		for _, id := range instanceIDs {
			workerPool <- struct{}{} // Acquire worker
			go func(instanceID string) {
				defer func() { <-workerPool }() // Release worker

				// Fetch EC2 instance configuration from AWS
				logger.Infof("Fetching EC2 instance %s configuration from AWS...", instanceID)
				awsConfig, err := awsClient.GetEC2InstanceConfig(instanceID)
				if err != nil {
					resultsChan <- struct {
						instanceID string
						drifts     map[string]drift.DriftDetail
						err        error
					}{instanceID: instanceID, err: fmt.Errorf("failed to get EC2 instance config: %v", err)}
					return
				}

				// Parse Terraform configuration
				var tfConfig map[string]interface{}
				if tfStatePath != "" {
					tfConfig, err = terraform.ParseStateFile(tfStatePath, instanceID)
				} else {
					tfConfig, err = terraform.ParseHCLConfig(tfConfigPath, instanceID)
				}

				if err != nil {
					resultsChan <- struct {
						instanceID string
						drifts     map[string]drift.DriftDetail
						err        error
					}{instanceID: instanceID, err: fmt.Errorf("failed to parse Terraform configuration: %v", err)}
					return
				}

				// Detect drift
				drifts, err := drift.DetectDrift(awsConfig, tfConfig, attributesToCheck)
				resultsChan <- struct {
					instanceID string
					drifts     map[string]drift.DriftDetail
					err        error
				}{instanceID: instanceID, drifts: drifts, err: err}
			}(id)
		}

		// Collect and process results
		var hasErrors bool
		for i := 0; i < len(instanceIDs); i++ {
			result := <-resultsChan
			if result.err != nil {
				logger.Errorf("Error processing instance %s: %v", result.instanceID, result.err)
				hasErrors = true
				continue
			}

			// Output results for each instance
			fmt.Printf("\nResults for instance %s:\n", result.instanceID)
			output := output.FormatDriftResults(result.drifts, result.instanceID, outputFormat)
			fmt.Println(output)

			if len(result.drifts) > 0 {
				attributes := make([]string, 0, len(result.drifts))
				for attr := range result.drifts {
					attributes = append(attributes, attr)
				}
				logger.Warnf("Instance %s: Drift detected in %d attributes: %s",
					result.instanceID, len(result.drifts), strings.Join(attributes, ", "))
			} else {
				logger.Infof("Instance %s: No drift detected", result.instanceID)
			}
		}

		if hasErrors {
			logger.Fatal("One or more instances failed to process")
		}
	},
}

var (
	instanceIDs       []string
	maxConcurrency    int
	defaultAttributes = []string{
		"instance_type",
		"ami",
		"subnet_id",
		"vpc_security_group_ids",
		"associate_public_ip_address",
		"tags",
		"root_block_device",
		"ebs_block_device",
	}
)

func init() {
	rootCmd.AddCommand(driftCmd)
	driftCmd.Flags().StringSliceVarP(&instanceIDs, "instances", "i", nil, "EC2 instance IDs to check (required, comma-separated)")
	driftCmd.Flags().StringVarP(&tfStatePath, "state", "s", "", "Path to Terraform state file")
	driftCmd.Flags().StringVarP(&tfConfigPath, "config", "c", "", "Path to Terraform HCL configuration directory")
	driftCmd.Flags().StringSliceVarP(&attributesToCheck, "attributes", "a", defaultAttributes, "Attributes to check for drift (comma-separated)")
	driftCmd.Flags().IntVarP(&maxConcurrency, "concurrency", "n", 5, "Maximum number of concurrent instance checks")

	driftCmd.MarkFlagRequired("instances")
}
