package cmd

var (
	cfgFile string
	verbose bool
	region string
	output string 
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.aws-terror.yaml)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&region, "region", "", "AWS region to use (overrides config/env)")
	rootCmd.PersistentFlags().StringVar(&output, "output", "text", "Output format: text, json, or yaml")
}