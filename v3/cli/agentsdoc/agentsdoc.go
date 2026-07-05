// Package agentsdoc generates markdown documentation for LLMs about the
// westack CLI (commands, type mapping, example model).
package agentsdoc

import (
	"fmt"
	"strings"

	"github.com/fredyk/westack-go/v3/cli"
)

// Generate returns a markdown guide for LLMs covering:
//  - the CLI commands (model new, add-field, add-relation, generate, migrate)
//  - the Go-type-to-SQL column mapping
//  - an example model
func Generate(models []cli.Model) string {
	var b strings.Builder

	b.WriteString("# Westack CLI Reference\n\n")
	b.WriteString("## CLI Commands\n\n")
	b.WriteString("- `model new <name>` — creates a new model file\n")
	b.WriteString("- `add-field <model> <field>` — adds a field to a model\n")
	b.WriteString("- `add-relation <model> <relation>` — adds a relation to a model\n")
	b.WriteString("- `generate` — generates Go code from models\n")
	b.WriteString("- `migrate` — applies DDL to the database\n\n")

	b.WriteString("## Type Mapping\n\n")
	b.WriteString("Go type to SQL column mapping:\n\n")
	b.WriteString("- `string` → `text`\n")
	b.WriteString("- `int` → `bigint`\n")
	b.WriteString("- `time.Time` → `timestamptz`\n")
	b.WriteString("- `bool` → `boolean`\n")
	b.WriteString("- `vector(N)` → `vector(N)` (pgvector)\n\n")

	b.WriteString("## Example Models\n\n")
	for _, m := range models {
		b.WriteString(fmt.Sprintf("### %s\n\n", m.Name))
		for _, f := range m.Fields {
			b.WriteString(fmt.Sprintf("- `%s` (%s)\n", f.Name, f.Type))
		}
		b.WriteString("\n")
	}

	return b.String()
}
