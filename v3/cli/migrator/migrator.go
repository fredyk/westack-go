// Package migrator derives relational DDL from westack Go models and can
// apply it to a PostgreSQL database.
package migrator

import (
	"context"

	"github.com/fredyk/westack-go/v3/cli"
)

// Migrator derives SQL DDL from a set of westack models and applies it.
type Migrator interface {
	// DDL returns the complete DDL for the given models.
	DDL(models []cli.Model) string

	// Apply executes the DDL against the given database connection.
	// This is a stub: it does nothing until a real implementation lands.
	Apply(ctx context.Context, conn any, models []cli.Model) error
}

// New returns a stub implementation that compiles but does not produce
// real DDL or apply anything.
type stubMigrator struct{}

func New() Migrator {
	return stubMigrator{}
}

func (stubMigrator) DDL(_ []cli.Model) string {
	return ""
}

func (stubMigrator) Apply(_ context.Context, _ any, _ []cli.Model) error {
	return nil
}
