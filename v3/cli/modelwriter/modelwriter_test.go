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

// TestNewModel_createsStructWithFields — Golden: NewModel genera un archivo Go
// que contiene un struct con todos los campos. El stub no escribe nada, así
// que este test FALLA (rojo). Cuando se implemente correctamente, PASA.
func TestNewModel_createsStructWithFields(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	model := cli.Model{
		Name: "Account",
		Base: "Account",
		Fields: []cli.Field{
			{Name: "Email", Type: cli.TypeString},
			{Name: "Password", Type: cli.TypeString},
		},
	}
	err := mw.NewModel(model, filepath.Join(d, "account.wst.go"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(d, "account.wst.go"))
	if err != nil {
		t.Fatal("NewModel should have created the file:", err)
	}
	src := string(data)
	for _, want := range []string{"type Account struct", "Email", "Password"} {
		if !strings.Contains(src, want) {
			t.Errorf("golden: Expected generated Go to contain %q, got:\n%s", want, src)
		}
	}
}

// TestNewModel_idempotent — Golden: llamar dos veces no duplica el struct.
func TestNewModel_idempotent(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	model := cli.Model{
		Name: "Account",
		Base: "Account",
		Fields: []cli.Field{
			{Name: "Name", Type: cli.TypeString},
		},
	}
	mw.NewModel(model, filepath.Join(d, "account.wst.go"))
	mw.NewModel(model, filepath.Join(d, "account.wst.go"))

	data, err := os.ReadFile(filepath.Join(d, "account.wst.go"))
	if err != nil {
		t.Fatal("file should exist after 2 calls:", err)
	}
	src := string(data)
	count := strings.Count(src, "type Account struct")
	if count != 1 {
		t.Errorf("idempotent: expected exactly 1 struct definition, found %d. Got:\n%s", count, src)
	}
}

// TestAddField_appendsField — Golden: tras AddField, el campo está en el struct.
func TestAddField_appendsField(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	existing := []byte("package models\n\ntype Account struct {}\n")
	path := filepath.Join(d, "account.wst.go")
	os.WriteFile(path, existing, 0644)

	mw.AddField(path, "Account", cli.Field{Name: "Email", Type: cli.TypeString})

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "Email") {
		t.Errorf("golden: expected field 'Email' to be in the struct. Got:\n%s", src)
	}
}

// TestAddField_idempotent — Golden: llamar dos veces no duplica el campo.
func TestAddField_idempotent(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	existing := []byte("package models\n\ntype Account struct {}\n")
	path := filepath.Join(d, "account.wst.go")
	os.WriteFile(path, existing, 0644)

	for i := 0; i < 2; i++ {
		mw.AddField(path, "Account", cli.Field{Name: "Email", Type: cli.TypeString})
	}

	data, _ := os.ReadFile(path)
	src := string(data)
	count := strings.Count(src, "Email")
	if count != 1 {
		t.Errorf("idempotent: expected exactly 1 'Email', got %d. Got:\n%s", count, src)
	}
}

// TestAddRelation_appendsRelation — Golden: tras AddRelation, la relación aparece.
func TestAddRelation_appendsRelation(t *testing.T) {
	d := tmpDir(t)
	mw := modelwriter.New()
	existing := []byte("package models\n\ntype Account struct {}\n")
	path := filepath.Join(d, "account.wst.go")
	os.WriteFile(path, existing, 0644)

	mw.AddRelation(path, "Account", cli.Relation{
		Name: "profile", Kind: cli.RelHasOne,
		Target: "Profile", FKColumn: "account_id", PKColumn: "id",
	})

	data, _ := os.ReadFile(path)
	src := string(data)
	if !strings.Contains(src, "profile") {
		t.Errorf("golden: expected relation 'profile' to be in the file. Got:\n%s", src)
	}
}
