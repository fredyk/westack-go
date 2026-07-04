package datasource

// SchemaBuilder generates DDL (SQL or otherwise) from model definitions.
type SchemaBuilder interface {
	// BuildCreateTable generates the DDL CREATE TABLE statement for the given model.
	BuildCreateTable(model ModelDef) (string, error)

	// MapPropertyType maps a Go PropertyType to the SQL column type string.
	// Rules:
	//   string  → text
	//   int     → bigint
	//   int64   → bigint
	//   float32 → real
	//   float64 → double precision
	//   bool    → boolean
	//   time    → timestamptz
	//   bytes   → bytea
	//   vector  → vector(N) where N = dims
	//   json    → jsonb
	MapPropertyType(prop PropertyType, dims int) (string, error)

	// BuildAlterTable generates ALTER statements for adding columns that don't exist yet.
	BuildAlterTable(model ModelDef, existingColumns []string) ([]string, error)

	// BuildCreateIndex generates CREATE INDEX statements for the model.
	BuildCreateIndex(model ModelDef) ([]string, error)

	// BuildAddForeignKey generates ALTER TABLE ... ADD CONSTRAINT ... FOREIGN KEY statements.
	BuildAddForeignKey(model ModelDef, rel RelationDef) (string, error)
}

// Migrator applies schema changes to a live database.
type Migrator interface {
	// EnsureSchema creates or updates the schema for the given model.
	EnsureSchema(model ModelDef) error

	// DropSchema drops the table for the given model (use with caution).
	DropSchema(model ModelDef) error

	// GetTableColumns returns the list of existing column names for a table.
	GetTableColumns(tableName string) ([]string, error)
}
