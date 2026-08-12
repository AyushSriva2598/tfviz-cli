package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tfviz",
	Short: "Real-time Terraform visualization and syncing",
	Long: `TFViz is a production-grade CLI tool that parses Terraform (HCL) locally 
and securely streams the architecture graph to your cloud dashboard in real-time.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}