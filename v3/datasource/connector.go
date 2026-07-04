package datasource

import "context"

// PropertyType represents the Go-level type of a model property.
type PropertyType string

const (
	PropString      PropertyType = "string"
	PropInt         PropertyType = "int"
	PropInt64       PropertyType = "int64"
	PropFloat32     PropertyType = "float32"
	PropFloat64     PropertyType = "float64"
	PropBool        PropertyType = "bool"
	PropTime        PropertyType = "time"
	PropBytes       PropertyType = "bytes"
	PropVector      PropertyType = "vector"
	PropJSON        PropertyType = "json"
)

// RelationType represents the type of relationship.
type RelationType string

const (
	BelongsTo RelationType = "belongsTo"
	HasMany   RelationType = "hasMany"
	HasOne    RelationType = "hasOne"
)

// PropertyDef defines a single field of a model.
type PropertyDef struct {
	Name       string
	Type       PropertyType
	PrimaryKey bool
	Nullable   bool
	Length     int // for string types (0 = unlimited); for Vector, the dimensions
	DefaultValue interface{}
}

// RelationDef defines a relationship between models.
type RelationDef struct {
	Type          RelationType
	Name          string // e.g. "cliente", "expedientes"
	TargetModel   string
	ForeignKey    string // column name on the FK side
	SourceField   string // field on this model that holds the reference
}

// ModelDef is the Go-level definition of a persistable model.
type ModelDef struct {
	Name       string
	Collection string // table/collection name
	Properties []PropertyDef
	Relations  []RelationDef
}

// ColumnMapping represents the SQL mapping of a property.
type ColumnMapping struct {
	ColumnName string
	SQLType   string // e.g. "text", "bigint", "timestamptz", "boolean", "double precision", "vector(N)"
	Nullable  bool
}

// PersistedConnector is the v3 generic interface for data persistence,
// replacing the v2 version that was coupled to MongoCursorI and BSON pipelines.
type PersistedConnector interface {
	// GetName returns the connector name (e.g. "postgres", "mongodb", "memorykv").
	GetName() string
	// Connect establishes a connection to the datasource.
	Connect(ctx context.Context) error
	// Disconnect closes the connection.
	Disconnect() error
	// Ping checks connectivity.
	Ping(ctx context.Context) error

	// FindMany executes a generic query and returns a Cursor.
	FindMany(ctx context.Context, collection string, query *Query) (Cursor, error)
	// FindById retrieves a single document by its primary key.
	FindById(ctx context.Context, collection string, id interface{}) (map[string]interface{}, error)
	// Count returns the number of documents matching the filter.
	Count(ctx context.Context, collection string, filter *Filter) (int64, error)
	// Create inserts a single document and returns it with generated fields.
	Create(ctx context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error)
	// CreateMany inserts multiple documents and returns them.
	CreateMany(ctx context.Context, collection string, data []map[string]interface{}) ([]map[string]interface{}, error)
	// UpdateById updates a document by primary key and returns the updated version.
	UpdateById(ctx context.Context, collection string, id interface{}, data map[string]interface{}) (map[string]interface{}, error)
	// DeleteById deletes a document by primary key.
	DeleteById(ctx context.Context, collection string, id interface{}) (int64, error)
	// DeleteMany deletes documents matching the filter.
	DeleteMany(ctx context.Context, collection string, filter *Filter) (int64, error)

	// Migrate creates or alters the schema for the given model definition.
	Migrate(ctx context.Context, model ModelDef) error
}

// ModelRegistry holds the set of model definitions for DDL generation.
type ModelRegistry struct {
	Models []ModelDef
}

// GetByName returns the ModelDef for the given name, or nil if not found.
func (r *ModelRegistry) GetByName(name string) *ModelDef {
	for i := range r.Models {
		if r.Models[i].Name == name {
			return &r.Models[i]
		}
	}
	return nil
}
