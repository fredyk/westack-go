package cli_test

import (
	"strings"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/commands"
)

// TestParseField_ok parses valid name:type pairs.
func TestParseField_ok(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantType cli.FieldType
	}{
		{"Email:string", "Email", cli.TypeString},
		{"Count:int", "Count", cli.TypeInt},
		{"CreatedAt:time.Time", "CreatedAt", cli.TypeTime},
		{"Active:bool", "Active", cli.TypeBool},
		{"Embedding:vector", "Embedding", cli.TypeVector},
		{"Embedding:vector(4096)", "Embedding", cli.TypeVector},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			f, err := commands.ParseField(tt.input)
			if err != nil {
				t.Fatalf("ParseField(%q) unexpected error: %v", tt.input, err)
			}
			if f.Name != tt.wantName {
				t.Errorf("name = %q, want %q", f.Name, tt.wantName)
			}
			if f.Type != tt.wantType {
				t.Errorf("type = %q, want %q", f.Type, tt.wantType)
			}
		})
	}
}

// TestParseField_vectorDim extracts dimension from vector(N).
func TestParseField_vectorDim(t *testing.T) {
	f, err := commands.ParseField("Embedding:vector(768)")
	if err != nil {
		t.Fatal(err)
	}
	if f.Type != cli.TypeVector {
		t.Errorf("type = %q, want vector", f.Type)
	}
	hasDim := false
	for _, opt := range f.Options {
		if vd, ok := opt.(cli.VectorDim); ok && int(vd) == 768 {
			hasDim = true
		}
	}
	if !hasDim {
		t.Errorf("expected VectorDim(768) option, got options: %v", f.Options)
	}
}

// TestParseField_noColon returns error.
func TestParseField_noColon(t *testing.T) {
	_, err := commands.ParseField("Email")
	if err == nil {
		t.Fatal("expected error for input without colon")
	}
}

// TestParseField_emptyName returns error.
func TestParseField_emptyName(t *testing.T) {
	_, err := commands.ParseField(":string")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestParseField_emptyType returns error.
func TestParseField_emptyType(t *testing.T) {
	_, err := commands.ParseField("Email:")
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

// TestParseField_unknownType returns error.
func TestParseField_unknownType(t *testing.T) {
	_, err := commands.ParseField("Foo:float32")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

// TestParseField_invalidVectorDim returns error.
func TestParseField_invalidVectorDim(t *testing.T) {
	_, err := commands.ParseField("Embedding:vector(abc)")
	if err == nil {
		t.Fatal("expected error for non-numeric vector dimension")
	}
}

// TestParseFields_parsesMultiple.
func TestParseFields_parsesMultiple(t *testing.T) {
	inputs := []string{"Email:string", "Count:int", "Active:bool"}
	fields, err := commands.ParseFields(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}
	if fields[0].Name != "Email" || fields[1].Name != "Count" || fields[2].Name != "Active" {
		t.Errorf("unexpected field names: %v", fields)
	}
}

// TestParseFields_emptyList_returnsEmpty.
func TestParseFields_emptyList(t *testing.T) {
	fields, err := commands.ParseFields(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Errorf("expected empty slice, got %d fields", len(fields))
	}
}

// TestParseModelName_invalidName returns error.
func TestParseModelName_invalidName(t *testing.T) {
	_, err := commands.ParseModelName("123invalid")
	if err == nil {
		t.Fatal("expected error for invalid model name starting with digit")
	}
}

// TestParseModelName_validName.
func TestParseModelName_validName(t *testing.T) {
	name, err := commands.ParseModelName("Account")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Account" {
		t.Errorf("name = %q, want %q", name, "Account")
	}
}

// TestParseModelName_emptyName returns error.
func TestParseModelName_emptyName(t *testing.T) {
	_, err := commands.ParseModelName("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestParseModelName_lowercase returns error.
func TestParseModelName_lowercase(t *testing.T) {
	_, err := commands.ParseModelName("account")
	if err == nil {
		t.Fatal("expected error for lowercase name")
	}
}

// TestParseModelName_hyphen returns error.
func TestParseModelName_hyphen(t *testing.T) {
	_, err := commands.ParseModelName("my-model")
	if err == nil {
		t.Fatal("expected error for name with hyphen")
	}
}

// TestParseModelName_underscoreOk.
func TestParseModelName_underscoreOk(t *testing.T) {
	name, err := commands.ParseModelName("My_Model")
	if err != nil {
		t.Fatal(err)
	}
	if name != "My_Model" {
		t.Errorf("name = %q, want %q", name, "My_Model")
	}
}

// TestParseModelName_veryLong returns error.
func TestParseModelName_veryLong(t *testing.T) {
	long := strings.Repeat("a", 129)
	_, err := commands.ParseModelName(long)
	if err == nil {
		t.Fatal("expected error for very long name")
	}
}
