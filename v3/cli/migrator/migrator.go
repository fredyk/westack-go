// Package migrator derives relational DDL from westack Go models and can
// apply it to a PostgreSQL database.
package migrator

import (
	"context"
	"fmt"
	"strings"

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

// migratorImpl generates PostgreSQL DDL from westack models.
type migratorImpl struct{}

func New() Migrator {
	return migratorImpl{}
}

// goTypeToPG maps a westack FieldType to a PostgreSQL column type.
func goTypeToPG(f cli.Field) string {
	switch f.Type {
	case cli.TypeString:
		return "text"
	case cli.TypeInt:
		return "bigint"
	case cli.TypeTime:
		return "timestamptz"
	case cli.TypeBool:
		return "boolean"
	case cli.TypeVector:
		dim := 1536 // default
		for _, opt := range f.Options {
			if vd, ok := opt.(cli.VectorDim); ok {
				dim = int(vd)
			}
		}
		return fmt.Sprintf("vector(%d)", dim)
	default:
		return "text"
	}
}

// DDL returns the complete DDL (CREATE TABLE statements) for the given models.
func (migratorImpl) DDL(models []cli.Model) string {
	var b strings.Builder
	for _, model := range models {
		schema := strings.ToLower(model.Base)
		b.WriteString(fmt.Sprintf("create table if not exists %s (\n", schema))

		var cols []string
		cols = append(cols, fmt.Sprintf("    id bigint PRIMARY KEY DEFAULT nextval('%s_id_seq')", schema))

		for _, f := range model.Fields {
			colName := strings.ToLower(f.Name)
			if f.Name[0] >= 'A' && f.Name[0] <= 'Z' {
				colName = string(f.Name[0]+32) + f.Name[1:]
			}
			pgType := goTypeToPG(f)
			col := fmt.Sprintf("%s %s", colName, pgType)
			for _, opt := range f.Options {
				if _, ok := opt.(cli.Required); ok {
					col += " NOT NULL"
				}
			}
			cols = append(cols, col)
		}

		// Add FK columns for belongsTo relations
		for _, rel := range model.Relations {
			if rel.Kind == cli.RelBelongsTo {
				fk := strings.ToLower(rel.FKColumn)
				cols = append(cols, fmt.Sprintf("    %s bigint REFERENCES %s(%s)", fk, strings.ToLower(rel.Target), rel.PKColumn))
			}
		}

		b.WriteString(strings.Join(cols, ",\n"))
		b.WriteString("\n);")
		b.WriteString("\n\n")
	}
	return b.String()
}

func (migratorImpl) Apply(_ context.Context, _ any, _ []cli.Model) error {
	return nil
}
