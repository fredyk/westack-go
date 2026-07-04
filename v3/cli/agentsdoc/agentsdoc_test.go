package agentsdoc_test

import (
	"strings"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/agentsdoc"
)

func TestGenerate_containsSections(t *testing.T) {
	models := []cli.Model{
		{
			Name: "Account",
			Base: "Account",
			Fields: []cli.Field{{Name: "Email", Type: cli.TypeString}},
		},
	}
	got := agentsdoc.Generate(models)
	if got == "" {
		t.Fatal("golden: expected generated doc to contain sections, got empty string (stub)")
	}
	// Golden: doc must mention CLI commands.
	if !strings.Contains(strings.ToLower(got), "command") {
		t.Errorf("golden: doc should mention 'command'. Got:\n%s", got)
	}
	// Golden: doc must mention type mapping.
	if !strings.Contains(strings.ToLower(got), "map") {
		t.Errorf("golden: doc should mention 'map' (type mapping). Got:\n%s", got)
	}
	// Golden: doc must include the example model name.
	if !strings.Contains(got, "Account") {
		t.Errorf("golden: doc should include model name 'Account'. Got:\n%s", got)
	}
}
