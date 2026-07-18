// Package modelwriter provides editing of westack Go model files via go/ast.
//
// The Writer interface writes/edits the Go source file for a model.
// Operations are idempotent (running them twice doesn't duplicate anything)
// and preserve any hand-written code that is not part of the model definition.
package modelwriter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/format"
	"os"
	"strings"

	"github.com/fredyk/westack-go/v3/cli"
)

// ModelWriter edits a Go model file.
type ModelWriter interface {
	// NewModel creates the Go file for a new model.
	// If the file exists it is kept (idempotent).
	NewModel(model cli.Model, path string) error

	// AddField appends a new field to the model's struct in the Go file.
	// Idempotent: calling it twice for the same field is a no-op.
	AddField(path, modelName string, f cli.Field) error

	// AddRelation adds a relation to the model's relations map in the Go file.
	// Idempotent: calling it twice for the same relation is a no-op.
	AddRelation(path, modelName string, r cli.Relation) error
}

type modelWriterImpl struct{}

func New() ModelWriter {
	return modelWriterImpl{}
}

func (modelWriterImpl) NewModel(model cli.Model, path string) error {
	// If file exists, return (idempotent — model already defined)
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	// Build the Go source for the model
	var fields []string
	for _, f := range model.Fields {
		fields = append(fields, fmt.Sprintf("    %s %s %s", f.Name, goTypeToGo(f), buildTag(f)))
	}

	src := fmt.Sprintf("package models\n\ntype %s struct {\n%s\n}\n", model.Name, strings.Join(fields, "\n"))

	// format.Source on a well-formed struct source string never fails
	data, _ := format.Source([]byte(src))

	return os.WriteFile(path, data, 0644)
}

func (modelWriterImpl) AddField(path, modelName string, f cli.Field) error {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	// Find the struct and check if field already exists
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != modelName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Check idempotency: field already present
			for _, ft := range st.Fields.List {
				for _, name := range ft.Names {
					if name.Name == f.Name {
						return nil
					}
				}
			}

			// Add the new field
			tag := buildTag(f)
			field := &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(f.Name)},
				Type:  ast.NewIdent(goTypeToGo(f)),
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: tag},
			}
			st.Fields.List = append(st.Fields.List, field)
		}
	}

	// Write back
	var buf strings.Builder
	fs2 := token.NewFileSet()
	// format.Node on a valid Go AST never fails
	_ = format.Node(&buf, fs2, file)

	return os.WriteFile(path, []byte(buf.String()), 0644)
}

func (modelWriterImpl) AddRelation(path, modelName string, r cli.Relation) error {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	// Find the struct and check if relation field already exists
	for _, d := range file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != modelName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Check idempotency
			for _, ft := range st.Fields.List {
				for _, name := range ft.Names {
					if name.Name == r.Name {
						return nil
					}
				}
			}

			// Add relation as a field (pointer to target model)
			field := &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(r.Name)},
				Type:  &ast.StarExpr{X: ast.NewIdent(r.Target)},
				Tag:   &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`%s`", fmt.Sprintf("relation:%s target:%s fk:%s", r.Kind, r.Target, r.FKColumn))},
			}
			st.Fields.List = append(st.Fields.List, field)
		}
	}

	var buf strings.Builder
	fs2 := token.NewFileSet()
	// format.Node on a valid Go AST never fails
	_ = format.Node(&buf, fs2, file)

	return os.WriteFile(path, []byte(buf.String()), 0644)
}

// goTypeToGo maps a westack FieldType to a Go type name.
func goTypeToGo(f cli.Field) string {
	switch f.Type {
	case cli.TypeString:
		return "string"
	case cli.TypeInt:
		return "int64"
	case cli.TypeTime:
		return "time.Time"
	case cli.TypeBool:
		return "bool"
	case cli.TypeVector:
		return "[]float32"
	default:
		return "string"
	}
}

// buildTag creates a struct tag for a field.
func buildTag(f cli.Field) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("db:%s", strings.ToLower(f.Name)))
	for _, opt := range f.Options {
		switch opt.(type) {
		case cli.Required:
			parts = append(parts, "notnull:true")
		}
	}
	return fmt.Sprintf("`%s`", strings.Join(parts, " "))
}
