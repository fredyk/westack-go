package cli

// FieldType describes the Go type of a model field, which maps to a SQL column type.
type FieldType string

const (
	TypeString FieldType = "string"
	TypeInt    FieldType = "int"
	TypeTime   FieldType = "time.Time"
	TypeBool   FieldType = "bool"
	TypeVector FieldType = "vector"
)

// Field represents a typed field in a westack model.
type Field struct {
	Name     string
	Type     FieldType
	Options  []FieldOption
}

// FieldOption is a modifier for a field.
type FieldOption interface {
	applyTo(f *Field)
}

// Required marks a field as NOT NULL.
type Required struct{}

func (r Required) applyTo(f *Field) {
	f.Options = append(f.Options, r)
}

// PrimaryKey marks a field as the primary key of the model.
type PrimaryKey struct{}

func (pk PrimaryKey) applyTo(f *Field) {
	f.Options = append(f.Options, pk)
}

// VectorDim sets the dimension for a Vector type field.
type VectorDim int

func (vd VectorDim) applyTo(f *Field) {
	f.Options = append(f.Options, vd)
}

// FieldOpts applies common field options.
var (
	FieldRequired  = Required{}
	FieldPrimaryKey = PrimaryKey{}
)

// RelationKind describes the type of relation between two models.
type RelationKind string

const (
	RelBelongsTo RelationKind = "belongsTo"
	RelHasOne    RelationKind = "hasOne"
	RelHasMany   RelationKind = "hasMany"
)

// Relation represents a link from one model to another.
type Relation struct {
	Name   string
	Kind   RelationKind
	Target string
	// FKColumn is the foreign key column on the source model (belongsTo has FK here; hasMany/hasOne target the other side).
	FKColumn string
	// PKColumn is the referenced column on the target model.
	PKColumn string
}

// Model represents a westack domain model to be written as Go code and migrated to SQL.
type Model struct {
	Name    string
	Base    string
	Fields  []Field
	Relations []Relation
}
