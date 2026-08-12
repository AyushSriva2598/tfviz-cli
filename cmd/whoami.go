package cmd

import (
	"fmt"
	"strings"
	"tfviz-agent/pkg/auth"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current authentication status",
	Run: func(cmd *cobra.Command, args []string) {
		creds, err := auth.Load()
		if err != nil {
			fmt.Println("❌ Not logged in. Run `tfviz login` to authenticate.")
			return
		}

		// Aggressively mask the token, leaving only the prefix
		maskedToken := creds.Token
		if strings.HasPrefix(maskedToken, "tfviz_") {
			maskedToken = "tfviz_************************"
		} else if len(maskedToken) > 6 {
			maskedToken = maskedToken[:6] + "..."
		}

		fmt.Println("🟢 Currently authenticated:")
		fmt.Printf("   Key Name: %s\n", creds.Email)
		fmt.Printf("   User:     %s (%s)\n", creds.UserName, creds.UserID)
		fmt.Printf("   Org:      %s (%s)\n", creds.OrgName, creds.OrgID)
		fmt.Printf("   Token:    %s\n", maskedToken)
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}