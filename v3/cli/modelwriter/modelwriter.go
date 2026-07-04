// Package modelwriter provides editing of westack Go model files via go/ast.
//
// The Writer interface writes/edits the Go source file for a model.
// Operations are idempotent (running them twice doesn't duplicate anything)
// and preserve any hand-written code that is not part of the model definition.
package modelwriter

import "github.com/fredyk/westack-go/v3/cli"

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

// New returns a stub implementation that compiles but does not actually
// edit files (the real implementation will use go/ast).
//
// This stub always returns nil without side-effects so that the caller
// can build successfully.  Tests for this package should see the stub
// behaviour fail when they assert on file contents.
type stubWriter struct{}

func New() ModelWriter {
	return stubWriter{}
}

func (stubWriter) NewModel(_ cli.Model, _ string) error {
	return nil
}

func (stubWriter) AddField(_, _ string, _ cli.Field) error {
	return nil
}

func (stubWriter) AddRelation(_ string, _ string, _ cli.Relation) error {
	return nil
}
