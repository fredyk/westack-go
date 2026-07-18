package datasource

import (
	"context"
	"fmt"
	"testing"
)

func TestBuildWhereClause_NilFilter(t *testing.T) {
	clause, args := buildWhereClause(nil, 1)
	if clause != "" || args != nil {
		t.Errorf("nil filter should return empty clause, got %q / %v", clause, args)
	}
}

func TestBuildWhereClause_EmptyConditions(t *testing.T) {
	f := &Filter{}
	clause, args := buildWhereClause(f, 1)
	if clause != "" || args != nil {
		t.Errorf("empty filter should return empty clause, got %q / %v", clause, args)
	}
}

func TestBuildWhereClause_Eq(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "name", Op: "eq", Value: "Alice"}}}
	clause, args := buildWhereClause(f, 1)
	if clause != `"name" = $1` {
		t.Errorf("clause = %q, want `\"name\" = $1`", clause)
	}
	if len(args) != 1 || args[0] != "Alice" {
		t.Errorf("args = %v, want [Alice]", args)
	}
}

func TestBuildWhereClause_Neq(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "status", Op: "neq", Value: "archived"}}}
	clause, _ := buildWhereClause(f, 1)
	if clause != `"status" != $1` {
		t.Errorf("clause = %q", clause)
	}
}

func TestBuildWhereClause_Gt(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "age", Op: "gt", Value: 18}}}
	clause, _ := buildWhereClause(f, 1)
	if clause != `"age" > $1` {
		t.Errorf("clause = %q", clause)
	}
}

func TestBuildWhereClause_Gte(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "age", Op: "gte", Value: 18}}}
	clause, _ := buildWhereClause(f, 1)
	if clause != `"age" >= $1` {
		t.Errorf("clause = %q", clause)
	}
}

func TestBuildWhereClause_Lt(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "age", Op: "lt", Value: 65}}}
	clause, _ := buildWhereClause(f, 1)
	if clause != `"age" < $1` {
		t.Errorf("clause = %q", clause)
	}
}

func TestBuildWhereClause_Lte(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "age", Op: "lte", Value: 65}}}
	clause, _ := buildWhereClause(f, 1)
	if clause != `"age" <= $1` {
		t.Errorf("clause = %q", clause)
	}
}

func TestBuildWhereClause_IsNull(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "email", Op: "is_null"}}}
	clause, args := buildWhereClause(f, 1)
	if clause != `"email" IS NULL` {
		t.Errorf("clause = %q", clause)
	}
	if len(args) != 0 {
		t.Errorf("args should be empty for IS NULL, got %v", args)
	}
}

func TestBuildWhereClause_Like(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "name", Op: "like", Value: "A%"}}}
	clause, args := buildWhereClause(f, 1)
	if clause != `"name" LIKE $1` {
		t.Errorf("clause = %q", clause)
	}
	if args[0] != "A%" {
		t.Errorf("args[0] = %v, want A%%", args[0])
	}
}

func TestBuildWhereClause_DefaultLogicalAnd(t *testing.T) {
	f := &Filter{
		Conditions: []Condition{
			{Field: "name", Op: "eq", Value: "Alice"},
			{Field: "active", Op: "eq", Value: true},
		},
	}
	clause, args := buildWhereClause(f, 1)
	want := `"name" = $1 AND "active" = $2`
	if clause != want {
		t.Errorf("clause = %q, want %q", clause, want)
	}
	if len(args) != 2 {
		t.Errorf("len(args) = %d, want 2", len(args))
	}
}

func TestBuildWhereClause_LogicalOr(t *testing.T) {
	f := &Filter{
		LogicalOp: "or",
		Conditions: []Condition{
			{Field: "role", Op: "eq", Value: "admin"},
			{Field: "role", Op: "eq", Value: "owner"},
		},
	}
	clause, _ := buildWhereClause(f, 1)
	want := `"role" = $1 OR "role" = $2`
	if clause != want {
		t.Errorf("clause = %q, want %q", clause, want)
	}
}

func TestBuildWhereClause_StartIdxRespected(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "x", Op: "eq", Value: 1}}}
	clause, args := buildWhereClause(f, 5)
	if clause != `"x" = $5` {
		t.Errorf("clause = %q, want `\"x\" = $5`", clause)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Errorf("args = %v, want [1]", args)
	}
}

func TestBuildWhereClause_DefaultOpOnUnknown(t *testing.T) {
	f := &Filter{Conditions: []Condition{{Field: "data", Op: "unknown_op", Value: "x"}}}
	clause, args := buildWhereClause(f, 1)
	if clause != `"data" = $1` {
		t.Errorf("clause = %q, want `\"data\" = $1`", clause)
	}
	if len(args) != 1 || args[0] != "x" {
		t.Errorf("args = %v, want [x]", args)
	}
}

func TestVectorToLiteral(t *testing.T) {
	vec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	lit := vectorToLiteral(vec)
	want := "[0.1,0.2,0.3]"
	if lit != want {
		t.Errorf("vectorToLiteral = %q, want %q", lit, want)
	}
}

func TestVectorToLiteral_SingleElement(t *testing.T) {
	vec := Vector{Values: []float32{1.5}, Dimensions: 1}
	lit := vectorToLiteral(vec)
	want := "[1.5]"
	if lit != want {
		t.Errorf("vectorToLiteral = %q, want %q", lit, want)
	}
}

func TestBuildReturnCols(t *testing.T) {
	data := map[string]interface{}{"id": "x", "name": "y", "embedding": "[1,2]"}
	cols := buildReturnCols(data)
	if cols == "" {
		t.Fatal("buildReturnCols returned empty")
	}
	if len(cols) < 3 {
		t.Errorf("buildReturnCols too short: %q", cols)
	}
}

func TestPqQuoteIdent_Simple(t *testing.T) {
	got := pqQuoteIdent("name")
	if got != `"name"` {
		t.Errorf("pqQuoteIdent(name) = %q, want `\"name\"`", got)
	}
}

func TestPqQuoteIdent_WithDoubleQuote(t *testing.T) {
	got := pqQuoteIdent(`bad"name`)
	if got != `"bad""name"` {
		t.Errorf("pqQuoteIdent with quotes = %q", got)
	}
}

func TestNewPostgresConnector_DefaultSchema(t *testing.T) {
	c := NewPostgresConnector("postgres://u:p@localhost/db", "")
	if c.schema != "public" {
		t.Errorf("default schema = %q, want public", c.schema)
	}
}

func TestNewPostgresConnector_CustomSchema(t *testing.T) {
	c := NewPostgresConnector("postgres://u:p@localhost/db", "custom")
	if c.schema != "custom" {
		t.Errorf("schema = %q, want custom", c.schema)
	}
}

func TestGetPrimaryKey_Found(t *testing.T) {
	c := &PostgresConnector{}
	model := &ModelDef{
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
		},
	}
	pk := c.getPrimaryKey(model)
	if pk != "id" {
		t.Errorf("getPrimaryKey = %q, want id", pk)
	}
}

func TestGetPrimaryKey_Fallback(t *testing.T) {
	c := &PostgresConnector{}
	model := &ModelDef{
		Properties: []PropertyDef{
			{Name: "name", Type: PropString},
		},
	}
	pk := c.getPrimaryKey(model)
	if pk != "id" {
		t.Errorf("getPrimaryKey fallback = %q, want id", pk)
	}
}

func TestGetVectorColumn_Found(t *testing.T) {
	c := &PostgresConnector{}
	model := &ModelDef{
		Properties: []PropertyDef{
			{Name: "id", Type: PropString},
			{Name: "embedding", Type: PropVector, Length: 3},
		},
	}
	col := c.getVectorColumn(model)
	if col != "embedding" {
		t.Errorf("getVectorColumn = %q, want embedding", col)
	}
}

func TestGetVectorColumn_NotFound(t *testing.T) {
	c := &PostgresConnector{}
	model := &ModelDef{
		Properties: []PropertyDef{
			{Name: "id", Type: PropString},
		},
	}
	col := c.getVectorColumn(model)
	if col != "" {
		t.Errorf("getVectorColumn with no vector = %q, want empty", col)
	}
}

func TestModelDef_GetByName_NotFound(t *testing.T) {
	reg := &ModelRegistry{}
	m := reg.GetByName("nonexistent")
	if m != nil {
		t.Errorf("GetByName(nonexistent) = %v, want nil", m)
	}
}

func TestMapPropertyType_VectorZeroDims(t *testing.T) {
	_, err := MapPropertyType(PropVector, 0)
	if err == nil {
		t.Fatal("MapPropertyType(vector, 0) should error")
	}
}

func TestMapPropertyType_Unknown(t *testing.T) {
	_, err := MapPropertyType("unknown_type", 0)
	if err == nil {
		t.Fatal("MapPropertyType(unknown) should error")
	}
}

func TestPostgresConnector_GetName(t *testing.T) {
	c := NewPostgresConnector("postgres://u:p@localhost/db", "public")
	if c.GetName() != "postgres" {
		t.Errorf("GetName() = %q, want %q", c.GetName(), "postgres")
	}
}

func TestPostgresConnector_Connect_BadDSN(t *testing.T) {
	c := NewPostgresConnector("not-a-valid-dsn!!!", "public")
	ctx := context.Background()
	err := c.Connect(ctx)
	if err == nil {
		t.Fatal("Connect() with bad DSN should error")
	}
}

func TestPostgresConnector_Ping_NotConnected(t *testing.T) {
	c := NewPostgresConnector("postgres://u:p@localhost/db", "public")
	ctx := context.Background()
	err := c.Ping(ctx)
	if err == nil {
		t.Fatal("Ping() on unconnected should error")
	}
}

func TestPostgresConnector_GetModel_NotFound(t *testing.T) {
	c := NewPostgresConnector("postgres://u:p@localhost/db", "public")
	_, err := c.getModel("nonexistent")
	if err == nil {
		t.Fatal("getModel() on unregistered should error")
	}
}

func TestPgxCursor_Err_ReturnsStoredError(t *testing.T) {
	want := fmt.Errorf("stored cursor error")
	cur := &pgxCursor{err: want}
	if cur.Err() != want {
		t.Errorf("Err() = %v, want %v", cur.Err(), want)
	}
}

func TestPgxCursor_Decode_WithError(t *testing.T) {
	want := fmt.Errorf("decode error")
	cur := &pgxCursor{err: want}
	var doc map[string]interface{}
	err := cur.Decode(&doc)
	if err != want {
		t.Errorf("Decode() error = %v, want %v", err, want)
	}
}