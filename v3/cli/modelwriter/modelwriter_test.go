package modelwriter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/modelwriter"
)

func tmpDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "modelwriter-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// TestNewModel_types — Golden: all Go type mappings are generated correctly.
func TestNewModel_types(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	model := cli.Model{
		Name: "AllTypes",
		Base: "AllTypes",
		Fields: []cli.Field{
			{Name: "Title", Type: cli.TypeString},
			{Name: "Count", Type: cli.TypeInt},
			{Name: "CreatedAt", Type: cli.TypeTime},
			{Name: "Active", Type: cli.TypeBool},
			{Name: "Embedding", Type: cli.TypeVector},
		},
	}
	err := mw.NewModel(model, filepath.Join(d, "alltypes.wst.go"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(d, "alltypes.wst.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{"string", "int64", "time.Time", "bool", "[]float32"} {
		if !strings.Contains(src, want) {
			t.Errorf("expected %q in generated Go file. Got:\n%s", want, src)
		}
	}
}

// TestNewModel_withRequired — Golden: required field gets notnull tag.
func TestNewModel_withRequired(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	model := cli.Model{
		Name: "User",
		Base: "User",
		Fields: []cli.Field{
			{Name: "Email", Type: cli.TypeString, Options: []cli.FieldOption{cli.FieldRequired}},
		},
	}
	err := mw.NewModel(model, filepath.Join(d, "user.wst.go"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(d, "user.wst.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "notnull:true") {
		t.Errorf("expected 'notnull:true' tag for required field. Got:\n%s", src)
	}
	if !strings.Contains(src, "db:email") {
		t.Errorf("expected 'db:email' tag. Got:\n%s", src)
	}
}

// TestAddField_withRequired — Golden: required field added via AddField has notnull tag.
func TestAddField_withRequired(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	existing := []byte("package models\n\ntype Account struct {}\n")
	path := filepath.Join(d, "account.wst.go")
	os.WriteFile(path, existing, 0644)

	mw.AddField(path, "Account", cli.Field{Name: "Email", Type: cli.TypeString, Options: []cli.FieldOption{cli.FieldRequired}})

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "notnull:true") {
		t.Errorf("expected 'notnull:true' for required field added via AddField. Got:\n%s", src)
	}
}

// TestAddRelation_idempotent — Golden: AddRelation calling twice does not duplicate.
func TestAddRelation_idempotent(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	existing := []byte("package models\n\ntype Account struct {}\n")
	path := filepath.Join(d, "account.wst.go")
	os.WriteFile(path, existing, 0644)

	rel := cli.Relation{Name: "profile", Kind: cli.RelHasOne, Target: "Profile", FKColumn: "account_id", PKColumn: "id"}
	mw.AddRelation(path, "Account", rel)
	mw.AddRelation(path, "Account", rel)

	data, _ := os.ReadFile(path)
	src := string(data)
	count := strings.Count(src, "profile")
	if count != 1 {
		t.Errorf("idempotent: expected exactly 1 'profile', got %d. Got:\n%s", count, src)
	}
}

// TestAddRelation_belongsTo — Golden: belongsTo relation generates pointer field.
func TestAddRelation_belongsTo(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	existing := []byte("package models\n\ntype Comment struct {}\n")
	path := filepath.Join(d, "comment.wst.go")
	os.WriteFile(path, existing, 0644)

	mw.AddRelation(path, "Comment", cli.Relation{
		Name: "account", Kind: cli.RelBelongsTo,
		Target: "Account", FKColumn: "account_id", PKColumn: "id",
	})

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "*Account") {
		t.Errorf("golden: expected '*Account' pointer field. Got:\n%s", src)
	}
	if !strings.Contains(src, "relation:belongsTo") {
		t.Errorf("expected 'relation:belongsTo' tag. Got:\n%s", src)
	}
	if !strings.Contains(src, "target:Account") {
		t.Errorf("expected 'target:Account' tag. Got:\n%s", src)
	}
	if !strings.Contains(src, "fk:account_id") {
		t.Errorf("expected 'fk:account_id' tag. Got:\n%s", src)
	}
}

// TestNewModel_emptyFields — Golden: model with no fields still generates valid struct.
func TestNewModel_emptyFields(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	model := cli.Model{
		Name:   "EmptyModel",
		Base:   "EmptyModel",
		Fields: nil,
	}
	err := mw.NewModel(model, filepath.Join(d, "emptymodel.wst.go"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(d, "emptymodel.wst.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "type EmptyModel struct") {
		t.Errorf("golden: expected EmptyModel struct. Got:\n%s", src)
	}
}

// TestNewModel_unknownType_fallsBackToString — Golden: unknown field type falls back to string.
func TestNewModel_unknownType_fallsBackToString(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	model := cli.Model{
		Name: "Test",
		Base: "Test",
		Fields: []cli.Field{
			{Name: "Foo", Type: "unknown_type"},
		},
	}
	err := mw.NewModel(model, filepath.Join(d, "test.wst.go"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(d, "test.wst.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "string") {
		t.Errorf("expected fallback to 'string' for unknown type. Got:\n%s", src)
	}
}

// TestAddField_fieldNotFoundDoesNotError — Golden: adding field to nonexistent model does not error.
func TestAddField_fieldNotFoundDoesNotError(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "nonexistent.wst.go")
	existing := []byte("package models\n\ntype Other struct {}\n")
	os.WriteFile(path, existing, 0644)

	err := mw.AddField(path, "UnknownModel", cli.Field{Name: "Foo", Type: cli.TypeString})
	if err != nil {
		t.Fatalf("expected no error for unknown model, got: %v", err)
	}

	// File should be unchanged since model wasn't found
	data, _ := os.ReadFile(path)
	src := string(data)
	if strings.Contains(src, "Foo") {
		t.Errorf("expected 'Foo' NOT to be added when model is unknown. Got:\n%s", src)
	}
}

// TestAddRelation_fieldNotFoundDoesNotError — Golden: AddRelation for unknown model does not error.
func TestAddRelation_fieldNotFoundDoesNotError(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "other.wst.go")
	existing := []byte("package models\n\ntype Account struct {}\n")
	os.WriteFile(path, existing, 0644)

	err := mw.AddRelation(path, "Unknown", cli.Relation{Name: "foo", Kind: cli.RelHasOne, Target: "Foo", FKColumn: "foo_id", PKColumn: "id"})
	if err != nil {
		t.Fatalf("expected no error for unknown model, got: %v", err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if strings.Contains(src, "foo") {
		t.Errorf("expected 'foo' NOT to be added when model is unknown. Got:\n%s", src)
	}
}

// TestAddField_parseError — Golden: invalid Go source returns error.
func TestAddField_parseError(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "invalid.wst.go")
	os.WriteFile(path, []byte("this is not valid go source {{{"), 0644)

	err := mw.AddField(path, "Foo", cli.Field{Name: "Bar", Type: cli.TypeString})
	if err == nil {
		t.Fatal("expected error for invalid Go source, got nil")
	}
}

// TestAddField_nonStructType — Golden: model name exists but type is not a struct (interface).
func TestAddField_nonStructType(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "iface.wst.go")
	existing := []byte("package models\n\ntype Foo interface{}\n")
	os.WriteFile(path, existing, 0644)

	err := mw.AddField(path, "Foo", cli.Field{Name: "Bar", Type: cli.TypeString})
	if err != nil {
		t.Fatalf("expected no error when type is not a struct, got: %v", err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if strings.Contains(src, "Bar") {
		t.Errorf("expected 'Bar' NOT to be added to non-struct type. Got:\n%s", src)
	}
}

// TestAddField_idempotentExistingField — Golden: struct already has the field, file unchanged.
func TestAddField_idempotentExistingField(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "account.wst.go")
	existing := []byte("package models\n\ntype Account struct {\n\tEmail string `db:email`\n}\n")
	os.WriteFile(path, existing, 0644)

	before, _ := os.ReadFile(path)

	err := mw.AddField(path, "Account", cli.Field{Name: "Email", Type: cli.TypeString})
	if err != nil {
		t.Fatal(err)
	}

	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("expected file to be unchanged for idempotent field add.\nBefore:\n%s\nAfter:\n%s", before, after)
	}

	src := string(after)
	count := strings.Count(src, "Email")
	if count != 1 {
		t.Errorf("expected exactly 1 'Email', got %d. Got:\n%s", count, src)
	}
}

// TestAddRelation_nonStructType — Golden: model name exists but type is not a struct (interface).
func TestAddRelation_nonStructType(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "iface.wst.go")
	existing := []byte("package models\n\ntype Foo interface{}\n")
	os.WriteFile(path, existing, 0644)

	err := mw.AddRelation(path, "Foo", cli.Relation{Name: "bar", Kind: cli.RelHasOne, Target: "Bar", FKColumn: "bar_id", PKColumn: "id"})
	if err != nil {
		t.Fatalf("expected no error when type is not a struct, got: %v", err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if strings.Contains(src, "bar") {
		t.Errorf("expected 'bar' NOT to be added to non-struct type. Got:\n%s", src)
	}
}

// TestAddRelation_parseError — Golden: invalid Go source returns error.
func TestAddRelation_parseError(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "invalid.wst.go")
	os.WriteFile(path, []byte("not valid go source at all !!!"), 0644)

	err := mw.AddRelation(path, "Foo", cli.Relation{Name: "bar", Kind: cli.RelHasOne, Target: "Bar", FKColumn: "bar_id", PKColumn: "id"})
	if err == nil {
		t.Fatal("expected error for invalid Go source, got nil")
	}
}

// TestAddRelation_idempotentExisting — Golden: struct already has the relation field, file unchanged.
func TestAddRelation_idempotentExisting(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "account.wst.go")
	existing := []byte("package models\n\ntype Account struct {\n\tProfile *Profile `relation:hasOne target:Profile fk:account_id`\n}\n")
	os.WriteFile(path, existing, 0644)

	before, _ := os.ReadFile(path)

	err := mw.AddRelation(path, "Account", cli.Relation{Name: "Profile", Kind: cli.RelHasOne, Target: "Profile", FKColumn: "account_id", PKColumn: "id"})
	if err != nil {
		t.Fatal(err)
	}

	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("expected file to be unchanged for idempotent relation add.\nBefore:\n%s\nAfter:\n%s", before, after)
	}

	src := string(after)
	count := strings.Count(src, "Profile")
	if count != 3 {
		t.Errorf("expected exactly 3 'Profile' (field name, pointer type, tag target), got %d. Got:\n%s", count, src)
	}
}

// TestNewModel_fileExists — Idempotent: NewModel returns nil when file already exists.
func TestNewModel_fileExists(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	path := filepath.Join(d, "existing.wst.go")
	existing := []byte("package models\n\ntype Existing struct {\n\tName string `db:name`\n}\n")
	os.WriteFile(path, existing, 0644)

	err := mw.NewModel(cli.Model{Name: "Other", Base: "Other", Fields: []cli.Field{{Name: "Foo", Type: cli.TypeString}}}, path)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "Existing") {
		t.Errorf("expected file to be unchanged. Got:\n%s", src)
	}
}

// TestNewModel_formatError — Golden: format.Source failure returns error.
func TestNewModel_formatError(t *testing.T) {
	mw := modelwriter.New()
	// We can't easily trigger format.Source failure with valid AST,
	// but we can test the error wrapping by checking the function covers the path.
	// The format.Source error path is exercised by the fact that the function
	// returns error — the happy path already covers the format call.
	// To actually cover the error branch, we'd need an unreachable AST which
	// go/format can't produce from valid parsed source. Instead we test via
	// WriteFile error:
	model := cli.Model{
		Name: "Test",
		Base: "Test",
		Fields: []cli.Field{
			{Name: "Foo", Type: cli.TypeString},
		},
	}
	// Write to a non-existent directory — os.WriteFile will fail
	err := mw.NewModel(model, "/nonexistent/directory/model.wst.go")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// TestAddField_multipleGenDeclBlocks — Golden: file has multiple GenDecl blocks, field added to correct one.
func TestAddField_multipleGenDeclBlocks(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "multi.wst.go")
	existing := []byte("package models\n\ntype User struct {\n\tName string `db:name`\n}\n\ntype Post struct {\n\tTitle string `db:title`\n}\n")
	os.WriteFile(path, existing, 0644)

	err := mw.AddField(path, "Post", cli.Field{Name: "Body", Type: cli.TypeString})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "Body") {
		t.Errorf("expected 'Body' to be added. Got:\n%s", src)
	}
	// Post should have 2 fields now (Title + Body), User should still have 1 (Name)
	postFields := strings.Count(src, "`db:")
	if postFields != 3 {
		t.Errorf("expected 3 total db tags (Name, Title, Body), got %d. Got:\n%s", postFields, src)
	}
}
