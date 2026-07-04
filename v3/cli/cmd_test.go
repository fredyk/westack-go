package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/commands"
	"github.com/fredyk/westack-go/v3/cli/modelwriter"
)

// TestModelNew_createsFile — Golden: model new creates a Go file with the struct.
func TestModelNew_createsFile(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	model, fields, err := commands.ParseModelNew("Account", []string{"Email:string", "Password:string"}, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	err = commands.RunModelNew(model, fields, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(d, "account.wst.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("file should exist after RunModelNew:", err)
	}
	src := string(data)
	for _, want := range []string{"type Account struct", "Email", "Password"} {
		if !strings.Contains(src, want) {
			t.Errorf("golden: expected %q in generated file. Got:\n%s", want, src)
		}
	}
	_ = model
}

// TestModelNew_withFields — Golden: fields passed via --field appear in struct.
func TestModelNew_withFields(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	fields := []string{"Email:string", "Count:int", "Active:bool"}
	model, parsedFields, err := commands.ParseModelNew("User", fields, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	err = commands.RunModelNew(model, parsedFields, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(d, "user.wst.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{"type User struct", "Email", "Count", "Active"} {
		if !strings.Contains(src, want) {
			t.Errorf("golden: expected %q in generated file. Got:\n%s", want, src)
		}
	}
	_ = model
}

// TestModelNew_idempotent — Golden: calling twice does not duplicate struct.
func TestModelNew_idempotent(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	model, fields, err := commands.ParseModelNew("Account", nil, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	commands.RunModelNew(model, fields, mw, d)
	commands.RunModelNew(model, fields, mw, d)

	path := filepath.Join(d, "account.wst.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	count := strings.Count(src, "type Account struct")
	if count != 1 {
		t.Errorf("idempotent: expected exactly 1 struct, found %d. Got:\n%s", count, src)
	}
}

// TestModelNew_existingFile_noOverwrite — Golden: existing file is preserved.
func TestModelNew_existingFile_noOverwrite(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	path := filepath.Join(d, "account.wst.go")
	existing := []byte("package models\n\ntype Account struct {\n    CustomField string\n}\n")
	os.WriteFile(path, existing, 0644)

	model, fields, err := commands.ParseModelNew("Account", nil, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	err = commands.RunModelNew(model, fields, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "CustomField") {
		t.Errorf("golden: expected CustomField to be preserved. Got:\n%s", src)
	}
	_ = model
	_ = fields
}

// TestModelNew_invalidName_returnsError.
func TestModelNew_invalidName(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()
	_, _, err := commands.ParseModelNew("123", nil, mw, d)
	if err == nil {
		t.Fatal("expected error for invalid model name")
	}
}

// TestModelNew_unknownType_returnsError.
func TestModelNew_unknownType(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()
	_, _, err := commands.ParseModelNew("Test", []string{"Foo:float32"}, mw, d)
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

// TestModelAddField_appends — Golden: add-field adds field to existing struct.
func TestModelAddField_appends(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	model, _, err := commands.ParseModelNew("Account", nil, mw, d)
	if err != nil {
		t.Fatal(err)
	}
	commands.RunModelNew(model, nil, mw, d)

	path := filepath.Join(d, "account.wst.go")
	field, err := commands.ParseField("Email:string")
	if err != nil {
		t.Fatal(err)
	}

	err = commands.RunModelAddField(path, "Account", field, mw)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, "Email") {
		t.Errorf("golden: expected 'Email' in struct. Got:\n%s", src)
	}
}

// TestModelAddField_idempotent — Golden: adding same field twice doesn't duplicate.
func TestModelAddField_idempotent(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	model, _, err := commands.ParseModelNew("Account", nil, mw, d)
	if err != nil {
		t.Fatal(err)
	}
	commands.RunModelNew(model, nil, mw, d)

	path := filepath.Join(d, "account.wst.go")
	field, err := commands.ParseField("Email:string")
	if err != nil {
		t.Fatal(err)
	}

	commands.RunModelAddField(path, "Account", field, mw)
	commands.RunModelAddField(path, "Account", field, mw)

	data, _ := os.ReadFile(path)
	src := string(data)
	count := strings.Count(src, "Email")
	if count != 1 {
		t.Errorf("idempotent: expected exactly 1 'Email', got %d. Got:\n%s", count, src)
	}
}

// TestModelAddField_vector — Golden: vector field with dimension.
func TestModelAddField_vector(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	model, _, err := commands.ParseModelNew("Document", nil, mw, d)
	if err != nil {
		t.Fatal(err)
	}
	commands.RunModelNew(model, nil, mw, d)

	path := filepath.Join(d, "document.wst.go")
	field, err := commands.ParseField("Embedding:vector(4096)")
	if err != nil {
		t.Fatal(err)
	}

	err = commands.RunModelAddField(path, "Document", field, mw)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "Embedding") {
		t.Errorf("golden: expected 'Embedding' in struct. Got:\n%s", src)
	}
}

// TestMigrateCommand_generatesDDL — Golden: migrate produces correct DDL.
func TestMigrateCommand_generatesDDL(t *testing.T) {
	d := tmpDirForCli(t)
	mw := modelwriter.New()

	model, fields, err := commands.ParseModelNew("Account", []string{"Email:string", "Password:string"}, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	err = commands.RunModelNew(model, fields, mw, d)
	if err != nil {
		t.Fatal(err)
	}

	models := []cli.Model{
		{Name: "Account", Base: "Account", Fields: fields},
	}

	got := commands.RunMigrate(models)
	if got == "" {
		t.Fatal("golden: expected DDL output, got empty string")
	}
	if !strings.Contains(got, "create table") {
		t.Errorf("golden: expected 'create table' in DDL. Got:\n%s", got)
	}
	if !strings.Contains(got, "email text") {
		t.Errorf("golden: expected 'email text' column. Got:\n%s", got)
	}
}

// TestMigrateCommand_withVector — Golden: vector(N) appears in DDL.
func TestMigrateCommand_withVector(t *testing.T) {
	fields := []cli.Field{
		{Name: "Title", Type: cli.TypeString},
		{Name: "Embedding", Type: cli.TypeVector, Options: []cli.FieldOption{cli.VectorDim(4096)}},
	}
	models := []cli.Model{{Name: "Document", Base: "Document", Fields: fields}}

	got := commands.RunMigrate(models)
	if !strings.Contains(got, "vector(4096)") {
		t.Errorf("golden: expected 'vector(4096)'. Got:\n%s", got)
	}
}

// TestMigrateCommand_withBelongsToFK — Golden: FK column appears.
func TestMigrateCommand_withBelongsToFK(t *testing.T) {
	models := []cli.Model{
		{Name: "Account", Base: "Account", Fields: []cli.Field{{Name: "Email", Type: cli.TypeString}}},
		{
			Name: "Profile", Base: "Profile",
			Fields: []cli.Field{{Name: "DisplayName", Type: cli.TypeString}},
			Relations: []cli.Relation{
				{Name: "account", Kind: cli.RelBelongsTo, Target: "Account", FKColumn: "account_id", PKColumn: "id"},
			},
		},
	}

	got := commands.RunMigrate(models)
	if !strings.Contains(got, "account_id") {
		t.Errorf("golden: expected 'account_id' FK column. Got:\n%s", got)
	}
}

// TestMigrateCommand_intToBigint — Golden: int maps to bigint.
func TestMigrateCommand_intToBigint(t *testing.T) {
	models := []cli.Model{{Name: "Counter", Base: "Counter", Fields: []cli.Field{{Name: "Count", Type: cli.TypeInt}}}}
	got := commands.RunMigrate(models)
	if !strings.Contains(got, "count bigint") {
		t.Errorf("golden: expected 'count bigint'. Got:\n%s", got)
	}
}

// TestMigrateCommand_boolToBoolean — Golden: bool maps to boolean.
func TestMigrateCommand_boolToBoolean(t *testing.T) {
	models := []cli.Model{{Name: "Flag", Base: "Flag", Fields: []cli.Field{{Name: "Enabled", Type: cli.TypeBool}}}}
	got := commands.RunMigrate(models)
	if !strings.Contains(got, "enabled boolean") {
		t.Errorf("golden: expected 'enabled boolean'. Got:\n%s", got)
	}
}

// TestMigrateCommand_timeToTimestamptz — Golden: time.Time maps to timestamptz.
func TestMigrateCommand_timeToTimestamptz(t *testing.T) {
	models := []cli.Model{{Name: "Event", Base: "Event", Fields: []cli.Field{{Name: "CreatedAt", Type: cli.TypeTime}}}}
	got := commands.RunMigrate(models)
	if !strings.Contains(got, "createdAt timestamptz") {
		t.Errorf("golden: expected 'createdAt timestamptz'. Got:\n%s", got)
	}
}

func tmpDirForCli(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "cli-cmd-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}
