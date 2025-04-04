/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// driftCmd represents the drift command
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect configuration drift between AWS and Terraform",
	Long: `Detect configuration drift between AWS EC2 instances and Terraform configurations.

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("drift called")
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
