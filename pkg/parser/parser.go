package parser

import (
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ResourceNode represents the clean, scrubbed data sent to the cloud.
type ResourceNode struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	FilePath     string   `json:"file_path"`
	Line         int      `json:"line"`
	Dependencies []string `json:"dependencies"`
}

// ExtractTerraformNodes dives into HashiCorp's official native AST
func ExtractTerraformNodes(p *hclparse.Parser, filePath string) ([]ResourceNode, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	file, diags := p.ParseHCL(src, filePath)
	if diags.HasErrors() {
		// Suppressed diagnostics to keep logs clean on half-typed files
	}

	var nodes []ResourceNode
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nodes, nil
	}

	for _, block := range body.Blocks {
		var id, nodeType, name string

		if block.Type == "resource" && len(block.Labels) >= 2 {
			nodeType = block.Labels[0]
			name = block.Labels[1]
			id = nodeType + "." + name
		} else if block.Type == "module" && len(block.Labels) >= 1 {
			nodeType = "module"
			name = block.Labels[0]
			id = "module." + name
		} else {
			continue // Skip variables, providers, outputs
		}

		depMap := make(map[string]bool)

		for _, attr := range block.Body.Attributes {
			for _, traversal := range attr.Expr.Variables() {
				if len(traversal) >= 2 {
					root := traversal.RootName()

					// Filter out standard HCL keywords
					if root == "var" || root == "local" || root == "data" || root == "count" || root == "each" || root == "path" {
						continue
					}

					// Route module references correctly
					if root == "module" {
						if nameAttr, ok := traversal[1].(hcl.TraverseAttr); ok {
							depMap["module."+nameAttr.Name] = true
						}
						continue
					}

					// Standard resource dependency (e.g., aws_vpc.main)
					if nameAttr, ok := traversal[1].(hcl.TraverseAttr); ok {
						depMap[root+"."+nameAttr.Name] = true
					}
				}
			}
		}

		var dependencies []string
		for dep := range depMap {
			dependencies = append(dependencies, dep)
		}

		nodes = append(nodes, ResourceNode{
			ID:           id,
			Type:         nodeType,
			Name:         name,
			FilePath:     filePath,
			Line:         block.DefRange().Start.Line, // AST Line tracing
			Dependencies: dependencies,
		})
	}
	return nodes, nil
}