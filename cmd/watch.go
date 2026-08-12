package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/cobra"
	"tfviz-agent/pkg/auth"
	"tfviz-agent/pkg/config"
	"tfviz-agent/pkg/parser"
)

var watchCmd = &cobra.Command{
	Use:   "watch [path]",
	Short: "Watch a directory and stream changes to TFViz Cloud via HTTP POST",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		
		// Determine directory to watch (default to current directory)
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

		// Set up API Endpoint
		baseURL := os.Getenv("TFVIZ_BACKEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}
		apiURL := baseURL + "/api/cli/stream/"

		// 3. Setup File Watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("❌ Failed to create file watcher: %v", err)
		}
		defer watcher.Close()

		// Recursively add directory and subdirectories (skipping .git and .terraform)
		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if strings.Contains(path, ".terraform") || strings.Contains(path, ".git") {
					return filepath.SkipDir
				}
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			log.Fatalf("❌ Failed to walk directory: %v", err)
		}

		// 4. Setup Outbound Network Queue & Parser
		outboundQueue := make(chan []byte, 100)
		p := hclparse.NewParser()

		// Network Sender Goroutine (HTTP POST)
		go func() {
			client := &http.Client{Timeout: 10 * time.Second}
			for payload := range outboundQueue {
				req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
				if err != nil {
					log.Printf("❌ Failed to create request: %v", err)
					continue
				}

				req.Header.Set("Authorization", "Bearer "+creds.Token)
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					log.Printf("❌ Network write failed: %v", err)
					continue
				}
				
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
					body, _ := io.ReadAll(resp.Body)
					log.Printf("❌ Server rejected payload (Status %d): %s", resp.StatusCode, string(body))
				}
				resp.Body.Close()
			}
		}()

		// Channel to catch Ctrl+C gracefully
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		fmt.Printf("🔍 Watching directory '%s' for project: %s\n", dirPath, workspace.ProjectName)
		fmt.Println("✅ Connected. Listening for .tf file changes (Press Ctrl+C to exit)...")

		// 5. Debounce Loop & Event Listener
		debounceTimer := time.NewTimer(150 * time.Millisecond)
		debounceTimer.Stop()
		pendingFiles := make(map[string]bool)

		for {
			select {
			case sig := <-sigChan:
				fmt.Printf("\n🛑 Received signal (%v). Stopping file watcher. Goodbye!\n", sig)
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				
				if strings.HasSuffix(event.Name, ".tf") {
					// Handle Deletions instantly
					if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
						payload := struct {
							Event     string `json:"event"`
							ProjectID string `json:"project_id"`
							FilePath  string `json:"file_path"`
							Nodes     []any  `json:"nodes"`
						}{
							Event:     "file_deleted",
							ProjectID: workspace.ProjectID,
							FilePath:  event.Name,
							Nodes:     []any{},
						}
						jsonData, _ := json.Marshal(payload)
						outboundQueue <- jsonData
						log.Printf("🗑️  File deleted: %s\n", event.Name)
						continue
					}

					// Handle Writes/Creates (debounced)
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
						pendingFiles[event.Name] = true
						debounceTimer.Reset(150 * time.Millisecond)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("❌ Watcher error: %v\n", err)

			case <-debounceTimer.C:
				for file := range pendingFiles {
					nodes, err := parser.ExtractTerraformNodes(p, file)
					if err != nil {
						log.Printf("⚠️  Skipping %s (syntax incomplete)", file)
						continue
					}

					payload := struct {
						Event     string                `json:"event"`
						ProjectID string                `json:"project_id"`
						FilePath  string                `json:"file_path"`
						Nodes     []parser.ResourceNode `json:"nodes"`
					}{
						Event:     "file_update",
						ProjectID: workspace.ProjectID,
						FilePath:  file,
						Nodes:     nodes,
					}

					jsonData, _ := json.Marshal(payload)
					outboundQueue <- jsonData
					log.Printf("⬆️  Synced %s (%d resources)\n", file, len(nodes))
				}
				pendingFiles = make(map[string]bool)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
}