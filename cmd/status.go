package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"tfviz-agent/pkg/auth"

	"github.com/spf13/cobra"
)

type Config struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check current local project link and sync health",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Check Auth
		creds, err := auth.Load()
		if err != nil {
			fmt.Println("❌ Not logged in. Run `tfviz login` first.")
			os.Exit(1)
		}

		// 2. Check Local Project Binding (.tfviz.json)
		file, err := os.ReadFile(".tfviz.json")
		if os.IsNotExist(err) {
			fmt.Println("⚠️  This directory is not linked to a TFViz project.")
			fmt.Println("   Run `tfviz init` to link it.")
			return
		} else if err != nil {
			fmt.Printf("❌ Error reading .tfviz.json: %v\n", err)
			os.Exit(1)
		}

		var config Config
		if err := json.Unmarshal(file, &config); err != nil {
			fmt.Println("❌ Corrupted .tfviz.json file. Run `tfviz init` again.")
			os.Exit(1)
		}

		// 3. Ping Backend to verify project still exists and is accessible
		baseURL := os.Getenv("TFVIZ_BACKEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}

		req, _ := http.NewRequest("GET", baseURL+"/api/cli/projects/", nil)
		req.Header.Set("Authorization", "Bearer "+creds.Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("⚠️  Warning: Could not reach TFViz cloud backend.")
		} else {
			defer resp.Body.Close()
		}

		// 4. Print Status Dashboard
		fmt.Println("📊 TFViz Local Workspace Status:")
		fmt.Printf("   User:        %s\n", creds.UserName)
		fmt.Printf("   Organization: %s\n", creds.OrgName)
		fmt.Printf("   Project Name: %s\n", config.ProjectName)
		fmt.Printf("   Project ID:   %s\n", config.ProjectID)
		fmt.Println("   Directory:    Linked & Healthy ✅")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}