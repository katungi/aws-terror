/*
Copyright © 2025 Daniel Denis <dankatdennis@gmail.com>
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

You can also use simulation mode to compare two Terraform state files without AWS access:
  aws-terror drift -i INSTANCE_ID -s SOURCE_STATE -t TARGET_STATE --simulate

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if instanceID == "" {
			globalSpinner.Error("Instance ID is required")
			logger.Fatal("Instance ID is required")
		}

		// Check if simulation mode is enabled
		simulate, _ := cmd.Flags().GetBool("simulate")
		targetState, _ := cmd.Flags().GetString("target-state")

		if simulate {
			if tfStatePath == "" || targetState == "" {
				globalSpinner.Error("Both source and target state files are required for simulation mode")
				logger.Fatal("Both source and target state files are required for simulation mode")
			}
			globalSpinner.UpdateMessage("Starting drift simulation")
			
			drifts, err := terraform.SimulateDrift(tfStatePath, targetState, instanceID)
			if err != nil {
				logger.Fatalf("Simulation failed: %v", err)
			}

			// Format and output results
			formattedOutput := output.FormatDriftResults(drifts, instanceID, outputFormat)
			fmt.Println(formattedOutput)
			return
		}

		if tfStatePath == "" && tfConfigPath == "" {
			globalSpinner.Error("Either Terraform state file or HCL configuration path is required")
			logger.Fatal("Either Terraform state file or HCL configuration path is required")
		}
		globalSpinner.UpdateMessage("Initializing drift detection")

		// Initialize AWS client
		globalSpinner.UpdateMessage("Initializing AWS client")
		awsClient, err := aws.NewClient(awsRegion, logger)
		if err != nil {
			globalSpinner.Error(fmt.Sprintf("Failed to initialize AWS client: %v", err))
			logger.Fatalf("Failed to initialize AWS client: %v", err)
		}

		// Create channels for results and errors
		// Get instance IDs from flag
		globalSpinner.UpdateMessage("Processing instance IDs")
		instanceIDs, err := cmd.Flags().GetStringSlice("instances")
		if err != nil {
			globalSpinner.Error(fmt.Sprintf("Failed to get instance IDs: %v", err))
			logger.Fatalf("Failed to get instance IDs: %v", err)
		}

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
awsConfig, err := awsClient.GetEC2InstanceConfig(cmd.Context(), instanceID)
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
			globalSpinner.Error("One or more instances failed to process")
			logger.Fatal("One or more instances failed to process")
		}
		globalSpinner.Success("Drift detection completed successfully")
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
	driftCmd.Flags().BoolP("simulate", "", false, "Enable simulation mode to compare two state files")
	driftCmd.Flags().StringP("target-state", "t", "", "Path to target Terraform state file for simulation mode")

	driftCmd.MarkFlagRequired("instances")
}
