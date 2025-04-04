package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/katungi/aws-terror/pkg/progress"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	logLevel          string
	awsRegion         string
	instanceID        string
	tfStatePath       string
	tfConfigPath      string
	outputFormat      string
	attributesToCheck []string
	logger            *logrus.Logger
	globalSpinner     *progress.Spinner
)

var rootCmd = &cobra.Command{
	Use:   "aws-terror",
	Short: "AWS-Terror - Detect drift between AWS resources and Terraform state",
	Long: `AWS-Terror is a CLI tool that helps you detect drift between your
AWS resources and your Terraform state files. This helps ensure your
infrastructure is in the expected state defined in your IaC.`,
}

func Execute() error {
	// Create a context that will be canceled on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		cancel()
	}()

	// Initialize global spinner with prettier steps
	globalSpinner = progress.NewSpinner("AWS-Terror Progress")
	globalSpinner.Start()
	defer globalSpinner.Stop()

	globalSpinner.UpdateMessage("⚡ Initializing CLI")
	globalSpinner.UpdateMessage("🔧 Setting up AWS client")
	globalSpinner.UpdateMessage("📝 Parsing Terraform configuration")
	globalSpinner.UpdateMessage("🔍 Detecting configuration drift")
	globalSpinner.UpdateMessage("📊 Formatting results")
	globalSpinner.UpdateMessage("✅ Execution complete")

	return rootCmd.ExecuteContext(ctx)
}

func ExecuteContext(ctx context.Context) error {
	// Initialize global spinner
	globalSpinner = progress.NewSpinner("Initializing AWS-Terror")
	globalSpinner.Start()
	defer globalSpinner.Stop()

	// Pass context to all commands
	rootCmd.SetContext(ctx)

	globalSpinner.UpdateMessage("1. Initializing AWS-Terror CLI")
	globalSpinner.UpdateMessage("2. Setting up AWS client configuration")
	globalSpinner.UpdateMessage("3. Parsing Terraform state/config")
	globalSpinner.UpdateMessage("4. Detecting configuration drift")
	globalSpinner.UpdateMessage("5. Formatting results")
	globalSpinner.UpdateMessage("Executing AWS-Terror commands")
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	// Initialize logger
	logger = logrus.New()

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "region", "", "AWS region (defaults to AWS_REGION env var)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "Output format (text, json, yaml)")

	// Set log level from flag
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

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
