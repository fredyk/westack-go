package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/modelwriter"
)

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
			f, err := ParseField(tt.input)
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

func TestParseField_vectorDim(t *testing.T) {
	f, err := ParseField("Embedding:vector(768)")
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

func TestParseField_noColon(t *testing.T) {
	_, err := ParseField("Email")
	if err == nil {
		t.Fatal("expected error for input without colon")
	}
}

func TestParseField_emptyName(t *testing.T) {
	_, err := ParseField(":string")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestParseField_emptyType(t *testing.T) {
	_, err := ParseField("Email:")
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestParseField_unknownType(t *testing.T) {
	_, err := ParseField("Foo:float32")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestParseField_invalidVectorDim(t *testing.T) {
	_, err := ParseField("Embedding:vector(abc)")
	if err == nil {
		t.Fatal("expected error for non-numeric vector dimension")
	}
}

func TestParseField_negativeVectorDim(t *testing.T) {
	_, err := ParseField("Embedding:vector(-1)")
	if err == nil {
		t.Fatal("expected error for negative vector dimension")
	}
}

func TestParseFields_parsesMultiple(t *testing.T) {
	inputs := []string{"Email:string", "Count:int", "Active:bool"}
	fields, err := ParseFields(inputs)
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

func TestParseFields_emptyList(t *testing.T) {
	fields, err := ParseFields(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Errorf("expected empty slice, got %d fields", len(fields))
	}
}

func TestParseFields_firstFails(t *testing.T) {
	_, err := ParseFields([]string{"Email:string", "Invalid"})
	if err == nil {
		t.Fatal("expected error when second field fails")
	}
}

func TestParseModelName_validName(t *testing.T) {
	name, err := ParseModelName("Account")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Account" {
		t.Errorf("name = %q, want %q", name, "Account")
	}
}

func TestParseModelName_invalidName(t *testing.T) {
	_, err := ParseModelName("123invalid")
	if err == nil {
		t.Fatal("expected error for invalid model name starting with digit")
	}
}

func TestParseModelName_emptyName(t *testing.T) {
	_, err := ParseModelName("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestParseModelName_lowercase(t *testing.T) {
	_, err := ParseModelName("account")
	if err == nil {
		t.Fatal("expected error for lowercase name")
	}
}

func TestParseModelName_hyphen(t *testing.T) {
	_, err := ParseModelName("my-model")
	if err == nil {
		t.Fatal("expected error for name with hyphen")
	}
}

func TestParseModelName_underscoreOk(t *testing.T) {
	name, err := ParseModelName("My_Model")
	if err != nil {
		t.Fatal(err)
	}
	if name != "My_Model" {
		t.Errorf("name = %q, want %q", name, "My_Model")
	}
}

func TestParseModelName_veryLong(t *testing.T) {
	long := "A"
	for i := 0; i < 128; i++ {
		long += "a"
	}
	_, err := ParseModelName(long)
	if err == nil {
		t.Fatal("expected error for very long name (129 chars)")
	}
}

func TestParseModelName_maxLength(t *testing.T) {
	long := "A"
	for i := 0; i < 127; i++ {
		long += "a"
	}
	_, err := ParseModelName(long)
	if err != nil {
		t.Fatalf("expected valid name for 128 chars, got: %v", err)
	}
}

func TestParseModelNew_valid(t *testing.T) {
	mw := modelwriter.New()
	d := "/tmp/test"
	model, fields, err := ParseModelNew("Account", []string{"Email:string"}, mw, d)
	if err != nil {
		t.Fatal(err)
	}
	if model.Name != "Account" {
		t.Errorf("model.Name = %q, want %q", model.Name, "Account")
	}
	if model.Base != "account" {
		t.Errorf("model.Base = %q, want %q", model.Base, "account")
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
}

func TestParseModelNew_invalidName(t *testing.T) {
	mw := modelwriter.New()
	_, _, err := ParseModelNew("123", nil, mw, "/tmp/test")
	if err == nil {
		t.Fatal("expected error for invalid model name")
	}
}

func TestParseModelNew_unknownType(t *testing.T) {
	mw := modelwriter.New()
	_, _, err := ParseModelNew("Test", []string{"Foo:float32"}, mw, "/tmp/test")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestParseModelNew_emptyFields(t *testing.T) {
	mw := modelwriter.New()
	model, fields, err := ParseModelNew("Empty", nil, mw, "/tmp/test")
	if err != nil {
		t.Fatal(err)
	}
	if model.Name != "Empty" {
		t.Errorf("model.Name = %q, want %q", model.Name, "Empty")
	}
	if len(fields) != 0 {
		t.Fatalf("expected 0 fields, got %d", len(fields))
	}
}

func TestRunMigrate_generatesDDL(t *testing.T) {
	models := []cli.Model{
		{Name: "Account", Base: "account", Fields: []cli.Field{{Name: "Email", Type: cli.TypeString}}},
	}
	got := RunMigrate(models)
	if got == "" {
		t.Fatal("expected DDL output, got empty string")
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty DDL")
	}
}

func TestRunModelNew_createsFile(t *testing.T) {
	mw := modelwriter.New()
	d, err := os.MkdirTemp("", "cmd-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)

	model := cli.Model{Name: "Account", Base: "account", Fields: []cli.Field{{Name: "Email", Type: cli.TypeString}}}
	err = RunModelNew(model, model.Fields, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(d, "account.wst.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("file should exist:", err)
	}
	src := string(data)
	if !containsStr(src, "type Account struct") {
		t.Errorf("expected 'type Account struct' in generated file. Got:\n%s", src)
	}
}

func TestRunModelAddField_appends(t *testing.T) {
	mw := modelwriter.New()
	d, err := os.MkdirTemp("", "cmd-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(d)

	path := filepath.Join(d, "account.wst.go")
	existing := []byte("package models\n\ntype Account struct {}\n")
	os.WriteFile(path, existing, 0644)

	field := cli.Field{Name: "Email", Type: cli.TypeString}
	err = RunModelAddField(path, "Account", field, mw)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if !containsStr(src, "Email") {
		t.Errorf("expected 'Email' in struct. Got:\n%s", src)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
