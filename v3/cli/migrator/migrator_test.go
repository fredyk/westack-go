package migrator_test

import (
	"context"
	"testing"

	"github.com/fredyk/westack-go/v3/cli"
	"github.com/fredyk/westack-go/v3/cli/migrator"
)

func TestDDL_emptyModels(t *testing.T) {
	m := migrator.New()
	got := m.DDL(nil)
	if got != "" {
		t.Errorf("expected empty DDL for nil models, got:\n%s", got)
	}
}

func TestDDL_multipleModels(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Account", Base: "Account", Fields: []cli.Field{{Name: "Email", Type: cli.TypeString}}},
		{Name: "Profile", Base: "Profile", Fields: []cli.Field{{Name: "Name", Type: cli.TypeString}}},
	}
	got := m.DDL(models)
	if got == "" {
		t.Fatal("expected DDL for multiple models")
	}
	// Both tables should be present
	for _, table := range []string{"account", "profile"} {
		if !contains(got, "create table if not exists "+table) {
			t.Errorf("expected table %q in DDL. Got:\n%s", table, got)
		}
	}
}

func TestDDL_requiredFieldNotNullable(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "User", Base: "User", Fields: []cli.Field{
			{Name: "Email", Type: cli.TypeString, Options: []cli.FieldOption{cli.FieldRequired}},
		}},
	}
	got := m.DDL(models)
	if !contains(got, "email text NOT NULL") {
		t.Errorf("expected 'email text NOT NULL' in DDL. Got:\n%s", got)
	}
}

func TestDDL_defaultTypeIsText(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Test", Base: "Test", Fields: []cli.Field{
			{Name: "Foo", Type: "unknown_type"},
		}},
	}
	got := m.DDL(models)
	if !contains(got, "foo text") {
		t.Errorf("expected fallback to 'text' for unknown type. Got:\n%s", got)
	}
}

func TestDDL_idSequence(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Widget", Base: "Widget", Fields: []cli.Field{{Name: "Name", Type: cli.TypeString}}},
	}
	got := m.DDL(models)
	if !contains(got, "widget_id_seq") {
		t.Errorf("expected 'widget_id_seq' sequence. Got:\n%s", got)
	}
	if !contains(got, "id bigint PRIMARY KEY") {
		t.Errorf("expected 'id bigint PRIMARY KEY DEFAULT nextval'. Got:\n%s", got)
	}
}

func TestDDL_hasNoTrailingComma(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Simple", Base: "Simple", Fields: []cli.Field{
			{Name: "A", Type: cli.TypeString},
			{Name: "B", Type: cli.TypeInt},
		}},
	}
	got := m.DDL(models)
	// The last column before the closing paren should not end with comma
	if contains(got, "b bigint,") {
		t.Errorf("expected last column without trailing comma. Got:\n%s", got)
	}
}

func TestDDL_hasOneRelation(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Profile", Base: "Profile", Fields: []cli.Field{{Name: "Bio", Type: cli.TypeString}},
			Relations: []cli.Relation{
				{Name: "account", Kind: cli.RelHasOne, Target: "Account", FKColumn: "account_id", PKColumn: "id"},
			},
		},
	}
	got := m.DDL(models)
	// hasOne should NOT generate FK column (only belongsTo does)
	if contains(got, "account_id") {
		t.Errorf("hasOne should not generate FK column. Got:\n%s", got)
	}
}

func TestDDL_hasManyRelation(t *testing.T) {
	m := migrator.New()
	models := []cli.Model{
		{Name: "Account", Base: "Account", Fields: []cli.Field{{Name: "Name", Type: cli.TypeString}},
			Relations: []cli.Relation{
				{Name: "profiles", Kind: cli.RelHasMany, Target: "Profile", FKColumn: "account_id", PKColumn: "id"},
			},
		},
	}
	got := m.DDL(models)
	// hasMany should NOT generate a REFERENCES FK column
	// (the sequence name 'account_id_seq' is fine, but no 'REFERENCES' should appear)
	if contains(got, "REFERENCES") {
		t.Errorf("hasMany should not generate FK REFERENCES. Got:\n%s", got)
	}
}

func TestApply_doesNothing(t *testing.T) {
	m := migrator.New()
	err := m.Apply(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("stub Apply returned error: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
