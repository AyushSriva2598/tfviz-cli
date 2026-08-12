package auth

import (
	"encoding/json"
	"github.com/zalando/go-keyring"
)

// We use these constants to define the "Vault" name in your OS Keychain
const serviceName = "tfviz-cli"
const accountName = "default-user" 

type Credentials struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	OrgID    string `json:"org_id"`
	OrgName  string `json:"org_name"`
	Email    string `json:"email"`
}

// Save marshals the credentials and locks them in the OS Keychain
func Save(creds *Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	// This securely writes to macOS Keychain / Windows Credential Manager
	return keyring.Set(serviceName, accountName, string(data))
}

// Load retrieves the locked credentials from the OS Keychain
func Load() (*Credentials, error) {
	data, err := keyring.Get(serviceName, accountName)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	err = json.Unmarshal([]byte(data), &creds)
	return &creds, err
}

// Delete removes the credentials from the OS Keychain (useful for a logout command)
func Delete() error {
	return keyring.Delete(serviceName, accountName)
}