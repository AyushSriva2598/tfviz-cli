package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/cobra"

	"tfviz-agent/pkg/parser"
)

var parseCmd = &cobra.Command{
	Use:   "parse [file_or_directory]",
	Short: "Parse local Terraform files and display the raw JSON AST",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		
		target := "."
		if len(args) > 0 {
			target = args[0]
		}

		p := hclparse.NewParser()
		var allNodes []parser.ResourceNode

		fileInfo, err := os.Stat(target)
		if err != nil {
			fmt.Printf("❌ Cannot access path: %v\n", err)
			os.Exit(1)
		}

		if fileInfo.IsDir() {
			// Walk directory and parse all .tf files
			err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
					nodes, err := parser.ExtractTerraformNodes(p, path)
					if err == nil {
						allNodes = append(allNodes, nodes...)
					}
				}
				return nil
			})
			if err != nil {
				log.Fatalf("❌ Failed to walk directory: %v", err)
			}
		} else {
			// Parse a single file
			if strings.HasSuffix(fileInfo.Name(), ".tf") {
				nodes, err := parser.ExtractTerraformNodes(p, target)
				if err == nil {
					allNodes = append(allNodes, nodes...)
				}
			}
		}

		if len(allNodes) == 0 {
			fmt.Println("⚠️  No valid Terraform resources found.")
			return
		}

		// Convert to pretty JSON and print
		jsonData, err := json.MarshalIndent(allNodes, "", "  ")
		if err != nil {
			log.Fatalf("❌ Failed to generate JSON: %v", err)
		}

		fmt.Println(string(jsonData))
		fmt.Printf("\n✅ Successfully parsed %d resources from %s.\n", len(allNodes), target)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
}