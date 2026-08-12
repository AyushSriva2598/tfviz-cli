package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// WorkspaceConfig represents the local directory's link to the cloud
type WorkspaceConfig struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	OrgID       string `json:"org_id"`
}

const configFileName = ".tfviz.json"

// Save writes the project configuration to the current directory
func Save(config *WorkspaceConfig) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.Join(cwd, configFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Load reads the local project configuration
func Load() (*WorkspaceConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(cwd, configFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("workspace not initialized. Run `tfviz init` first")
		}
		return nil, err
	}

	var config WorkspaceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.New("corrupted .tfviz.json file")
	}

	return &config, nil
}