// Package agentsdoc generates markdown documentation for LLMs about the
// westack CLI (commands, type mapping, example model).
package agentsdoc

import "github.com/fredyk/westack-go/v3/cli"

// Generate returns a markdown guide for LLMs covering:
//  - the CLI commands (model new, add-field, add-relation, generate, migrate)
//  - the Go-type-to-SQL column mapping
//  - an example model
func Generate(models []cli.Model) string {
	return ""
}
