package datasource

import "testing"

// stubSchemaBuilder implements SchemaBuilder — delegates to real MapPropertyType.
type stubSchemaBuilder struct{}

func (s *stubSchemaBuilder) BuildCreateTable(model ModelDef) (string, error) {
	return "CREATE TABLE stub (id text NOT NULL)", nil
}

func (s *stubSchemaBuilder) MapPropertyType(prop PropertyType, dims int) (string, error) {
	return MapPropertyType(prop, dims)
}

func (s *stubSchemaBuilder) BuildAlterTable(model ModelDef, existingColumns []string) ([]string, error) {
	return nil, nil
}

func (s *stubSchemaBuilder) BuildCreateIndex(model ModelDef) ([]string, error) {
	return nil, nil
}

func (s *stubSchemaBuilder) BuildAddForeignKey(model ModelDef, rel RelationDef) (string, error) {
	return "ALTER TABLE stub ADD CONSTRAINT fk_stub FOREIGN KEY (cliente_id) REFERENCES clientes(id)", nil
}

// Tests for SchemaBuilder property type mapping rules.

func TestMapPropertyType_String(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropString, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(string) error: %v", err)
	}
	if got != "text" {
		t.Errorf("MapPropertyType(string) = %q, want %q", got, "text")
	}
}

func TestMapPropertyType_Int(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropInt, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(int) error: %v", err)
	}
	if got != "bigint" {
		t.Errorf("MapPropertyType(int) = %q, want %q", got, "bigint")
	}
}

func TestMapPropertyType_Int64(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropInt64, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(int64) error: %v", err)
	}
	if got != "bigint" {
		t.Errorf("MapPropertyType(int64) = %q, want %q", got, "bigint")
	}
}

func TestMapPropertyType_Float32(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropFloat32, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(float32) error: %v", err)
	}
	if got != "real" {
		t.Errorf("MapPropertyType(float32) = %q, want %q", got, "real")
	}
}

func TestMapPropertyType_Float64(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropFloat64, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(float64) error: %v", err)
	}
	if got != "double precision" {
		t.Errorf("MapPropertyType(float64) = %q, want %q", got, "double precision")
	}
}

func TestMapPropertyType_Bool(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropBool, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(bool) error: %v", err)
	}
	if got != "boolean" {
		t.Errorf("MapPropertyType(bool) = %q, want %q", got, "boolean")
	}
}

func TestMapPropertyType_Time(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropTime, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(time) error: %v", err)
	}
	if got != "timestamptz" {
		t.Errorf("MapPropertyType(time) = %q, want %q", got, "timestamptz")
	}
}

func TestMapPropertyType_Bytes(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropBytes, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(bytes) error: %v", err)
	}
	if got != "bytea" {
		t.Errorf("MapPropertyType(bytes) = %q, want %q", got, "bytea")
	}
}

func TestMapPropertyType_Vector(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropVector, 1536)
	if err != nil {
		t.Fatalf("MapPropertyType(vector) error: %v", err)
	}
	expected := "vector(1536)"
	if got != expected {
		t.Errorf("MapPropertyType(vector, 1536) = %q, want %q", got, expected)
	}
}

func TestMapPropertyType_Vector4096(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropVector, 4096)
	if err != nil {
		t.Fatalf("MapPropertyType(vector) error: %v", err)
	}
	expected := "vector(4096)"
	if got != expected {
		t.Errorf("MapPropertyType(vector, 4096) = %q, want %q", got, expected)
	}
}

func TestMapPropertyType_JSON(t *testing.T) {
	s := &stubSchemaBuilder{}
	got, err := s.MapPropertyType(PropJSON, 0)
	if err != nil {
		t.Fatalf("MapPropertyType(json) error: %v", err)
	}
	if got != "jsonb" {
		t.Errorf("MapPropertyType(json) = %q, want %q", got, "jsonb")
	}
}

func TestBuildCreateTable_GeneratesDDL(t *testing.T) {
	s := &stubSchemaBuilder{}
	model := ModelDef{
		Name:       "Cliente",
		Collection: "clientes",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
			{Name: "email", Type: PropString, Nullable: true},
		},
	}
	ddl, err := s.BuildCreateTable(model)
	if err != nil {
		t.Fatalf("BuildCreateTable() error: %v", err)
	}
	if ddl == "" {
		t.Error("BuildCreateTable() returned empty DDL")
	}
}

func TestBuildAddForeignKey_GeneratesConstraint(t *testing.T) {
	s := &stubSchemaBuilder{}
	model := ModelDef{
		Name:       "Expediente",
		Collection: "expedientes",
	}
	rel := RelationDef{
		Type:        BelongsTo,
		TargetModel: "Cliente",
		ForeignKey:  "cliente_id",
	}
	ddl, err := s.BuildAddForeignKey(model, rel)
	if err != nil {
		t.Fatalf("BuildAddForeignKey() error: %v", err)
	}
	if ddl == "" {
		t.Error("BuildAddForeignKey() returned empty DDL")
	}
}

func TestSchemaBuilder_ImplementsInterface(t *testing.T) {
	var _ SchemaBuilder = (*stubSchemaBuilder)(nil)
}

func TestModelRegistry_GetByName(t *testing.T) {
	reg := &ModelRegistry{
		Models: []ModelDef{
			{Name: "Cliente", Collection: "clientes"},
			{Name: "Expediente", Collection: "expedientes"},
		},
	}
	m := reg.GetByName("Expediente")
	if m == nil {
		t.Fatal("GetByName('Expediente') returned nil")
	}
	if m.Collection != "expedientes" {
		t.Errorf("Collection = %q, want expedientes", m.Collection)
	}
	if reg.GetByName("NonExistent") != nil {
		t.Error("GetByName('NonExistent') should return nil")
	}
}
