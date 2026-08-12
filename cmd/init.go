package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	// "io"
	"net/http"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"tfviz-agent/pkg/auth"
	"tfviz-agent/pkg/config"
)

type ProjectItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Link this directory to a TFViz Cloud project",
	Run: func(cmd *cobra.Command, args []string) {

		creds, err := auth.Load()
		if err != nil {
			fmt.Println("❌ Not logged in. Run `tfviz login <api_key>` first.")
			os.Exit(1)
		}

		if existing, err := config.Load(); err == nil {
			fmt.Printf("⚠️  This directory is already linked to project: %s\n", existing.ProjectName)
			fmt.Println("To link a different project, delete .tfviz.json and run init again.")
			os.Exit(0)
		}

		baseURL := os.Getenv("TFVIZ_BACKEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8000"
		}
		apiURL := baseURL + "/api/projects/"

		// 1. Fetch available projects
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Set("Authorization", "Bearer "+creds.Token)
		client := &http.Client{}
		resp, err := client.Do(req)
		
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Println("❌ Failed to fetch projects from TFViz Cloud.")
			os.Exit(1)
		}
		
		var projects []ProjectItem
		json.NewDecoder(resp.Body).Decode(&projects)
		resp.Body.Close()

		// 2. Build the selection menu
		options := []string{"+ Create a new project"}
		projectMap := make(map[string]ProjectItem)
		
		for _, p := range projects {
			options = append(options, p.Name)
			projectMap[p.Name] = p
		}

		var selectedOption string
		prompt := &survey.Select{
			Message: "Choose a project to link this directory to:",
			Options: options,
		}
		survey.AskOne(prompt, &selectedOption)

		var targetProject ProjectItem

		// 3. Handle Project Creation
		if selectedOption == "+ Create a new project" {
			var newProjectName string
			namePrompt := &survey.Input{Message: "Enter new project name:"}
			survey.AskOne(namePrompt, &newProjectName)

			payload := map[string]string{"name": newProjectName}
			jsonPayload, _ := json.Marshal(payload)
			
			createReq, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
			createReq.Header.Set("Authorization", "Bearer "+creds.Token)
			createReq.Header.Set("Content-Type", "application/json")
			
			createResp, err := client.Do(createReq)
			if err != nil || createResp.StatusCode != 201 {
				fmt.Println("❌ Failed to create project.")
				os.Exit(1)
			}
			
			json.NewDecoder(createResp.Body).Decode(&targetProject)
			createResp.Body.Close()
			fmt.Printf("✨ Created new project: %s\n", targetProject.Name)
			
		} else {
			targetProject = projectMap[selectedOption]
		}

		// 4. Save local configuration
		selectedConfig := &config.WorkspaceConfig{
			ProjectID:   targetProject.ID,
			ProjectName: targetProject.Name,
			OrgID:       creds.OrgID,
		}

		if err := config.Save(selectedConfig); err != nil {
            fmt.Printf("❌ Failed to save workspace config: %v\n", err)
            os.Exit(1)
        }

        // --- AUTOMATIC GITIGNORE APPEND ---
        ensureGitIgnore(".tfviz.json")

        fmt.Printf("✅ Success! Directory linked to: %s\n", targetProject.Name)

		fmt.Println("You can now run `tfviz watch` to start streaming architecture changes.")
	},
}

func ensureGitIgnore(filename string) {
	ignoreFile := ".gitignore"
	
	// Read existing .gitignore if it exists
	content, err := os.ReadFile(ignoreFile)
	if err == nil {
		// Check if it's already ignored
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == filename {
				return // Already ignored, do nothing
			}
		}
	}

	// Append to .gitignore
	f, err := os.OpenFile(ignoreFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Ensure newline spacing if file existed and didn't end with newline
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		f.WriteString("\n")
	}

	f.WriteString(filename + "\n")
	fmt.Println("🛡️  Added .tfviz.json to local .gitignore")
}

func init() {
	rootCmd.AddCommand(initCmd)
}