package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/cobra"
	"tfviz-agent/pkg/auth"
	"tfviz-agent/pkg/config"
	"tfviz-agent/pkg/parser"
)

var pushCmd = &cobra.Command{
	Use:   "push [path]",
	Short: "Parse local Terraform files and push architecture state to TFViz Cloud",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dirPath := "."
		if len(args) > 0 {
			dirPath = args[0]
		}

		// 1. Verify Authentication
		creds, err := auth.Load()
		if err != nil {
			fmt.Println("❌ Not logged in. Run `tfviz login` first.")
			os.Exit(1)
		}

		// 2. Verify Project Link
		workspace, err := config.Load()
		if err != nil {
			fmt.Println("❌ Directory not linked. Run `tfviz init` first.")
			os.Exit(1)
		}

		baseURL := os.Getenv("TFVIZ_BACKEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}
		apiURL := baseURL + "/api/cli/stream/"

		fmt.Printf("🚀 Scanning '%s' for project: %s...\n", dirPath, workspace.ProjectName)

		// 3. Collect and parse all .tf files
		p := hclparse.NewParser()
		var allNodes []parser.ResourceNode
		scannedFiles := 0

		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if strings.Contains(path, ".terraform") || strings.Contains(path, ".git") {
					return filepath.SkipDir
				}
				return nil
			}

			if strings.HasSuffix(path, ".tf") {
				scannedFiles++
				nodes, err := parser.ExtractTerraformNodes(p, path)
				if err != nil {
					log.Printf("⚠️  Skipping %s (syntax incomplete/error: %v)", path, err)
					return nil
				}
				allNodes = append(allNodes, nodes...)
			}
			return nil
		})

		if err != nil {
			log.Fatalf("❌ Failed to read directory: %v", err)
		}

		if scannedFiles == 0 {
			fmt.Println("⚠️  No .tf files found to push.")
			return
		}

		// 4. Build single push payload
		payload := struct {
			Event     string                `json:"event"`
			ProjectID string                `json:"project_id"`
			FilePath  string                `json:"file_path"`
			Nodes     []parser.ResourceNode `json:"nodes"`
		}{
			Event:     "batch_push",
			ProjectID: workspace.ProjectID,
			FilePath:  dirPath,
			Nodes:     allNodes,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("❌ Failed to format payload: %v", err)
		}

		// 5. Send payload to backend
		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Fatalf("❌ Failed to create request: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+creds.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("❌ Network push failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			log.Fatalf("❌ Server rejected push (Status %d): %s", resp.StatusCode, string(body))
		}

		fmt.Printf("✅ Pushed successfully! (%d files scanned, %d resources synced)\n", scannedFiles, len(allNodes))
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
