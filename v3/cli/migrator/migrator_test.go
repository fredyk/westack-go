package migrator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/migrator"
)

func TestDDL_singleModel(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{
			Name: "Account",
			Base: "Account",
			Fields: []cli.Field{
				{Name: "Email", Type: cli.TypeString},
				{Name: "Password", Type: cli.TypeString},
				{Name: "Created", Type: cli.TypeTime},
			},
		},
	}
	got := m.DDL(models)
	if got == "" {
		t.Fatal("golden: expected DDL for Account model, got empty string (stub)")
	}
	if !strings.Contains(got, "create table") {
		t.Errorf("golden: expected 'Create table' in DDL. Got:\n%s", got)
	}
	if !strings.Contains(got, "email text") {
		t.Errorf("golden: expected 'email text' column mapping. Got:\n%s", got)
	}
	if !strings.Contains(got, "created timestamptz") {
		t.Errorf("golden: expected 'created timestamptz' column mapping. Got:\n%s", got)
	}
	// Regresión: las columnas deben ir separadas por COMA (un CREATE TABLE con
	// columnas unidas solo por '\n' es SQL inválido — "syntax error"). Bug real
	// cazado por el e2e de integración; aquí queda protegido en unit.
	if !strings.Contains(got, "email text,") {
		t.Errorf("columnas sin coma separadora (DDL inválido). Got:\n%s", got)
	}
}

func TestDDL_withBelongsToFK(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{
			Name: "Account", Base: "Account",
			Fields: []cli.Field{{Name: "Email", Type: cli.TypeString}},
		},
		{
			Name: "Profile", Base: "Profile",
			Fields: []cli.Field{{Name: "DisplayName", Type: cli.TypeString}},
			Relations: []cli.Relation{
				{Name: "account", Kind: cli.RelBelongsTo, Target: "Account", FKColumn: "account_id", PKColumn: "id"},
			},
		},
	}
	got := m.DDL(models)
	if got == "" {
		t.Fatal("golden: expected DDL with FK, got empty string (stub)")
	}
	if !strings.Contains(got, "account_id") {
		t.Errorf("golden: expected 'account_id' column for belongsTo FK. Got:\n%s", got)
	}
}

func TestDDL_withVector(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{
			Name: "Document", Base: "Document",
			Fields: []cli.Field{
				{Name: "Title", Type: cli.TypeString},
				{Name: "Embedding", Type: cli.TypeVector, Options: []cli.FieldOption{cli.VectorDim(4096)}},
			},
		},
	}
	got := m.DDL(models)
	if got == "" {
		t.Fatal("golden: expected DDL with vector(4096), got empty string (stub)")
	}
	if !strings.Contains(got, "vector(4096)") {
		t.Errorf("golden: expected 'vector(4096)'. Got:\n%s", got)
	}
}

func TestDDL_intToBigint(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Counter", Base: "Counter", Fields: []cli.Field{{Name: "Count", Type: cli.TypeInt}}},
	}
	got := m.DDL(models)
	if got == "" {
		t.Fatal("golden: expected DDL, got empty string (stub)")
	}
	if !strings.Contains(got, "count bigint") {
		t.Errorf("golden: expected 'count bigint'. Got:\n%s", got)
	}
}

func TestDDL_boolToBoolean(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Flag", Base: "Flag", Fields: []cli.Field{{Name: "Enabled", Type: cli.TypeBool}}},
	}
	got := m.DDL(models)
	if got == "" {
		t.Fatal("golden: expected DDL, got empty string (stub)")
	}
	if !strings.Contains(got, "enabled boolean") {
		t.Errorf("golden: expected 'enabled boolean'. Got:\n%s", got)
	}
}

func TestApply_doesNothing(t *testing.T) {
	m := migrator.New()
	err := m.Apply(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("stub Apply returned error: %v", err)
	}
}
