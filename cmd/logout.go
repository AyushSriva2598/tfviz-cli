package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"tfviz-agent/pkg/auth"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local authentication credentials",
	Run: func(cmd *cobra.Command, args []string) {
		
		err := auth.Delete()
		if err != nil {
			fmt.Printf("❌ Failed to clear credentials: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println("✅ Successfully logged out. Credentials removed from device.")
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}