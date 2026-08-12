package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"tfviz-agent/pkg/auth"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate your machine with TFViz Cloud",
	Run: func(cmd *cobra.Command, args []string) {
		var apiKey string

		// If the user accidentally passes it as an argument, warn them but accept it.
		// Otherwise, use a secure, hidden prompt.
		if len(args) > 0 {
			apiKey = strings.TrimSpace(args[0])
			fmt.Println("⚠️  Warning: Passing secrets as arguments saves them to your shell history.")
			fmt.Println("   In the future, just run `tfviz login` for a secure prompt.")
		} else {
			prompt := &survey.Password{
				Message: "Paste your TFViz API Key:",
			}
			err := survey.AskOne(prompt, &apiKey)
			if err != nil {
				fmt.Println("❌ Login cancelled.")
				os.Exit(1)
			}
		}

		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			fmt.Println("❌ API Key cannot be empty.")
			os.Exit(1)
		}

		baseURL := os.Getenv("TFVIZ_BACKEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}

		req, _ := http.NewRequest("GET", baseURL+"/api/cli/whoami/", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("❌ Network error: Could not reach TFViz Cloud.")
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Println("❌ Authentication failed. Invalid API Key.")
			os.Exit(1)
		}

		var result struct {
			Authenticated bool   `json:"authenticated"`
			KeyName       string `json:"key_name"`
			UserID        string `json:"user_id"`
			UserName      string `json:"user_name"` // <-- NEW
			Organization  struct {
				ClerkID string `json:"clerk_id"`
				Name    string `json:"name"`      // <-- NEW
			} `json:"organization"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Println("❌ Failed to parse server response.")
			os.Exit(1)
		}

		// Save GLOBAL credentials
		creds := &auth.Credentials{
			Token:    apiKey,
			UserID:   result.UserID,
			UserName: result.UserName,
			OrgID:    result.Organization.ClerkID,
			OrgName:  result.Organization.Name,
			Email:    result.KeyName,
		}

		if err := auth.Save(creds); err != nil {
			fmt.Printf("❌ Failed to save credentials: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully authenticated as: %s\n", result.KeyName)
		fmt.Println("Next, navigate to your Terraform directory and run `tfviz init`.")
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}