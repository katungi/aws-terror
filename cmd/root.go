package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logLevel         string
	awsRegion        string
	instanceID       string
	tfStatePath      string
	tfConfigPath     string
	outputFormat     string
	attributesToCheck []string
	logger           *logrus.Logger
)

var rootCmd = &cobra.Command{
	Use:   "aws-terror",
	Short: "AWS-Terror - Detect drift between AWS resources and Terraform state",
	Long: `AWS-Terror is a CLI tool that helps you detect drift between your
AWS resources and your Terraform state files. This helps ensure your
infrastructure is in the expected state defined in your IaC.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "region", "", "AWS region (defaults to AWS_REGION env var)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "Output format (text, json, yaml)")
	
	if len(attributesToCheck) == 0 {
		attributesToCheck = []string{
			"instance_type",
			"ami",
			"tags",
			"ebs_block_device",
			"subnet_id",
			"vpc_security_group_ids",
			"associate_public_ip_address",
		}
	}
}