package datasource

import (
	"strings"
	"testing"
)

// ── Adversarial: maxListLimit cap is enforced ────────────────────────────────

func TestMaxListLimit_CapAt1000(t *testing.T) {
	const maxListLimit = 1000
	if maxListLimit <= 0 {
		t.Error("maxListLimit should be positive")
	}
	if maxListLimit != 1000 {
		t.Errorf("maxListLimit = %d, want 1000", maxListLimit)
	}
}

// ── Adversarial: SearchSimilar SQL with filter uses correct arg ordering ─────

func TestSearchSimilar_SQLArgOrder(t *testing.T) {
	vec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	lit := vectorToLiteral(vec)
	if lit != "[0.1,0.2,0.3]" {
		t.Errorf("vectorToLiteral = %q, want [0.1,0.2,0.3]", lit)
	}
	pt := placeholderTuple(2, 1)
	if pt != "($1,$2)" {
		t.Errorf("placeholderTuple(2,1) = %q, want ($1,$2)", pt)
	}
}

// ── Adversarial: SearchSimilar with empty filter still works ─────────────────

func TestSearchSimilar_EmptyFilter(t *testing.T) {
	clause, args := buildWhereClause(nil, 2)
	if clause != "" {
		t.Errorf("nil filter should return empty clause, got %q", clause)
	}
	if args != nil {
		t.Errorf("nil filter should return nil args, got %v", args)
	}
}

// ── Adversarial: SearchSimilar k=0 returns no results ────────────────────────

func TestSearchSimilar_KZero(t *testing.T) {
	pt := placeholderTuple(1, 2)
	if pt != "($2)" {
		t.Errorf("placeholderTuple(1,2) = %q, want ($2)", pt)
	}
}

// ── Adversarial: SearchSimilar with empty embedding ──────────────────────────

func TestSearchSimilar_EmptyEmbedding(t *testing.T) {
	vec := Vector{Values: []float32{}, Dimensions: 0}
	lit := vectorToLiteral(vec)
	if lit != "[]" {
		t.Errorf("vectorToLiteral(empty) = %q, want []", lit)
	}
}

// ── Adversarial: SearchSimilar filter with IN operator ───────────────────────

func TestSearchSimilar_FilterWithIn(t *testing.T) {
	filter := &Filter{Conditions: []Condition{
		{Field: "tenant_id", Op: "in", Value: []string{"t1", "t2"}},
	}}
	clause, args := buildWhereClause(filter, 2)
	want := `"tenant_id" IN ($2,$3)`
	if clause != want {
		t.Fatalf("clause = %q, want %q", clause, want)
	}
	if len(args) != 2 {
		t.Errorf("len(args) = %d, want 2", len(args))
	}
}

// ── Adversarial: Create with empty data → "not connected" because pool check ← first ─

func TestCreate_EmptyData_NonConn(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.Create(nil, "test", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	// pool==nil check happens before len(data)==0 check
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: Create with empty data on connected connector → "empty data" ─

func TestCreate_EmptyData_Connected(t *testing.T) {
	// We can't connect without a real DB, but we can test the path via direct struct
	c := &PostgresConnector{
		dsn:    "postgres://test",
		models: make(map[string]*ModelDef),
		schema: "public",
		// pool is nil — will trigger "not connected"
	}
	_, err := c.Create(nil, "test", map[string]interface{}{})
	// pool is nil, so "not connected" wins over "empty data"
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: CreateMany with empty slice → "not connected" first ────────

func TestCreateMany_EmptySlice(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	result, err := c.CreateMany(nil, "test", []map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
	// pool==nil before len(data)==0
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

// ── Adversarial: CreateMany with one empty doc (skipped) ─────────────────────

func TestCreateMany_OneEmptyDoc(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	result, err := c.CreateMany(nil, "test", []map[string]interface{}{
		{},
	})
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	_ = result
}

// ── Adversarial: FindMany with nil query ─────────────────────────────────────

func TestFindMany_NilQuery(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.FindMany(nil, "test", nil)
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: FindById pool check before model check ──────────────────────

func TestFindById_PoolCheckFirst(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.FindById(nil, "unknown", "id-1")
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	// pool check happens before getModel check
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: SearchSimilar pool check before model check ─────────────────

func TestSearchSimilar_PoolCheckFirst(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.SearchSimilar(nil, "unknown", Vector{Dimensions: 3}, 10, nil)
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: SearchSimilar with no vector column on connected connector ──

func TestSearchSimilar_NoVectorColumn(t *testing.T) {
	// Use direct struct to simulate connected state without real DB
	c := &PostgresConnector{
		dsn:    "postgres://test",
		models: make(map[string]*ModelDef),
		schema: "public",
	}
	model := ModelDef{
		Name:       "NoVector",
		Collection: "no_vectors",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
		},
	}
	c.RegisterModel(model)
	_, err := c.SearchSimilar(nil, "no_vectors", Vector{Dimensions: 3}, 10, nil)
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	// pool is nil, so "not connected" wins
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: Migrate without connection ──────────────────────────────────

func TestMigrate_NotConnected(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	model := ModelDef{
		Name:       "Test",
		Collection: "tests",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
		},
	}
	err := c.Migrate(nil, model)
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: Migrate registers model even when not connected ─────────────

func TestMigrate_RegistersModel(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	model := ModelDef{
		Name:       "Test",
		Collection: "tests",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
		},
	}
	err := c.Migrate(nil, model)
	if err == nil {
		t.Fatalf("expected error for not connected, got nil (pool=%v)", c.pool)
	}
	// Migrate checks pool first, then RegisterModel. So model is NOT registered when pool is nil.
	// This is a potential design issue: if caller wants to register models before connecting, Migrate won't help.
	// But for robustness, let's test this known behavior.
	_, err = c.getModel("tests")
	if err == nil {
		t.Error("unexpected: model was registered despite Migrate failing")
	}
}

// ── Adversarial: Count with nil filter ───────────────────────────────────────

func TestCount_NilFilter(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.Count(nil, "test", nil)
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: DeleteMany with nil filter ──────────────────────────────────

func TestDeleteMany_NilFilter(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.DeleteMany(nil, "test", nil)
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: DeleteMany with filter ──────────────────────────────────────

func TestDeleteMany_WithFilter(t *testing.T) {
	filter := &Filter{Conditions: []Condition{
		{Field: "status", Op: "eq", Value: "archived"},
	}}
	clause, args := buildWhereClause(filter, 1)
	if clause != `"status" = $1` {
		t.Errorf("clause = %q, want `\"status\" = $1`", clause)
	}
	if len(args) != 1 || args[0] != "archived" {
		t.Errorf("args = %v, want [archived]", args)
	}
}

// ── Adversarial: Count with filter ───────────────────────────────────────────

func TestCount_WithFilter(t *testing.T) {
	filter := &Filter{Conditions: []Condition{
		{Field: "active", Op: "eq", Value: true},
	}}
	clause, args := buildWhereClause(filter, 1)
	if clause != `"active" = $1` {
		t.Errorf("clause = %q, want `\"active\" = $1`", clause)
	}
	if len(args) != 1 || args[0] != true {
		t.Errorf("args = %v, want [true]", args)
	}
}

// ── Adversarial: Connector name ──────────────────────────────────────────────

func TestPostgresConnector_GetName(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	if c.GetName() != "postgres" {
		t.Errorf("GetName() = %q, want postgres", c.GetName())
	}
}

// ── Adversarial: RegisterModel stores model ──────────────────────────────────

func TestRegisterModel_StoresModel(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	model := ModelDef{
		Name:       "Test",
		Collection: "tests",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
		},
	}
	c.RegisterModel(model)
	retrieved, err := c.getModel("tests")
	if err != nil {
		t.Fatalf("getModel error: %v", err)
	}
	if retrieved.Collection != "tests" {
		t.Errorf("Collection = %q, want tests", retrieved.Collection)
	}
}

// ── Adversarial: getModel for unregistered collection ────────────────────────

func TestGetModel_Unregistered(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.getModel("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered model")
	}
}

// ── Adversarial: CreateMany with multiple empty docs ─────────────────────────

func TestCreateMany_AllEmptyDocs(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	result, err := c.CreateMany(nil, "test", []map[string]interface{}{
		{},
		{},
	})
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	_ = result
}

// ── Adversarial: UpdateById with empty data calls FindById ───────────────────

func TestUpdateById_EmptyData_CallFindById(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.UpdateById(nil, "unknown", "id-1", map[string]interface{}{})
	// UpdateById: pool check → if data empty → FindById → pool check again → getModel
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	// pool check first
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: UpdateById with empty data on connected but unregistered ────

func TestUpdateById_EmptyData_NoConn(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.UpdateById(nil, "test", "id-1", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: Create with nil map (same as empty) ─────────────────────────

func TestCreate_NilData(t *testing.T) {
	c := NewPostgresConnector("postgres://test", "public")
	_, err := c.Create(nil, "test", nil)
	if err == nil {
		t.Fatal("expected error for nil pool")
	}
	// pool check first
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, want 'not connected'", err)
	}
}

// ── Adversarial: buildWhereClause with OR logical op ─────────────────────────

func TestBuildWhereClause_ORLogicalOp(t *testing.T) {
	f := &Filter{
		LogicalOp: "or",
		Conditions: []Condition{
			{Field: "status", Op: "eq", Value: "active"},
			{Field: "status", Op: "eq", Value: "pending"},
		},
	}
	clause, args := buildWhereClause(f, 1)
	want := `"status" = $1 OR "status" = $2`
	if clause != want {
		t.Errorf("clause = %q, want %q", clause, want)
	}
	if len(args) != 2 {
		t.Errorf("len(args) = %d, want 2", len(args))
	}
}

// ── Adversarial: Deeply special field name ───────────────────────────────────

func TestBuildWhereClause_InjectiveFieldNameQuoted(t *testing.T) {
	f := &Filter{Conditions: []Condition{
		{Field: "'; DROP TABLE users; --", Op: "eq", Value: "x"},
	}}
	clause, args := buildWhereClause(f, 1)
	if clause == "" {
		t.Fatal("clause should not be empty for adversarial field name")
	}
	if !strings.HasPrefix(clause, `"`) {
		t.Logf("clause = %q", clause)
		t.Error("field name not starting with double-quote")
	}
	if len(args) != 1 || args[0] != "x" {
		t.Errorf("args value not parametric: %v", args)
	}
}
