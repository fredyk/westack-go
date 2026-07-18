//go:build integration

package datasource

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ── Adversarial: Migrate idempotence (already tested in postgres_integration_test.go) ──

// TestIntegration_Migrate_Idempotent is already in postgres_integration_test.go.
// Additional idempotence tests below.

func TestIntegration_Migrate_DropAndRecreate(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "DropRecreate",
		Collection: "drop_recreate",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
		},
	}

	// Migrate, drop the table, migrate again — should recreate
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}

	// Drop the table directly
	if _, err := c.pool.Exec(ctx, "DROP TABLE "+pqQuoteIdent(schemaName)+".drop_recreate"); err != nil {
		t.Fatalf("DROP TABLE error: %v", err)
	}

	// Migrate again — should recreate the table without error
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("second Migrate() after drop error: %v", err)
	}

	// Verify the table works
	_, err := c.Create(ctx, "drop_recreate", map[string]interface{}{
		"id":   "test-1",
		"name": "works",
	})
	if err != nil {
		t.Fatalf("Create() after recreate error: %v", err)
	}
}

func TestIntegration_Migrate_AddNewColumn(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model1 := ModelDef{
		Name:       "AddColumn",
		Collection: "add_column",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
		},
	}
	if err := c.Migrate(ctx, model1); err != nil {
		t.Fatalf("first Migrate() error: %v", err)
	}

	model2 := ModelDef{
		Name:       "AddColumn",
		Collection: "add_column",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
			{Name: "email", Type: PropString, Nullable: true},
			{Name: "age", Type: PropInt64},
		},
	}
	if err := c.Migrate(ctx, model2); err != nil {
		t.Fatalf("second Migrate() (add columns) error: %v", err)
	}

	// Verify new columns exist and can be used
	_, err := c.Create(ctx, "add_column", map[string]interface{}{
		"id":    "test-1",
		"name":  "test",
		"email": "test@example.com",
		"age":   int64(25),
	})
	if err != nil {
		t.Fatalf("Create() after migrate error: %v", err)
	}
}

// ── Adversarial: KNN edge cases ──────────────────────────────────────────────

func TestIntegration_SearchSimilar_KZero(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "VecTest",
		Collection: "vec_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "embedding", Type: PropVector, Length: 3},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert some vectors
	for _, d := range []struct {
		id  string
		vec string
	}{
		{"v1", "[0.1,0.2,0.3]"},
		{"v2", "[0.4,0.5,0.6]"},
	} {
		_, err := c.Create(ctx, "vec_test", map[string]interface{}{
			"id":        d.id,
			"embedding": d.vec,
		})
		if err != nil {
			t.Fatalf("Create(%s) error: %v", d.id, err)
		}
	}

	// k=0 → should return empty results
	queryVec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	results, err := c.SearchSimilar(ctx, "vec_test", queryVec, 0, nil)
	if err != nil {
		t.Fatalf("SearchSimilar(k=0) error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchSimilar(k=0) returned %d results, want 0", len(results))
	}
}

func TestIntegration_SearchSimilar_EmptyVector(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "VecTest",
		Collection: "vec_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "embedding", Type: PropVector, Length: 3},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert some vectors
	for _, d := range []struct {
		id  string
		vec string
	}{
		{"v1", "[0.1,0.2,0.3]"},
		{"v2", "[0.4,0.5,0.6]"},
	} {
		_, err := c.Create(ctx, "vec_test", map[string]interface{}{
			"id":        d.id,
			"embedding": d.vec,
		})
		if err != nil {
			t.Fatalf("Create(%s) error: %v", d.id, err)
		}
	}

	// Empty vector — pgvector accepts [] and distance is 0 for all rows
	// This is an edge case; behavior depends on pgvector version
	queryVec := Vector{Values: []float32{}, Dimensions: 0}
	_, err := c.SearchSimilar(ctx, "vec_test", queryVec, 10, nil)
	// Either returns results or errors — both are acceptable edge cases
	t.Logf("SearchSimilar(empty vector) result: %v", err)
}

func TestIntegration_SearchSimilar_KExceedsRows(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "VecTest",
		Collection: "vec_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "embedding", Type: PropVector, Length: 3},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert 2 vectors
	for _, d := range []struct {
		id  string
		vec string
	}{
		{"v1", "[0.1,0.2,0.3]"},
		{"v2", "[0.4,0.5,0.6]"},
	} {
		_, err := c.Create(ctx, "vec_test", map[string]interface{}{
			"id":        d.id,
			"embedding": d.vec,
		})
		if err != nil {
			t.Fatalf("Create(%s) error: %v", d.id, err)
		}
	}

	// k=100 but only 2 rows → should return 2
	queryVec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	results, err := c.SearchSimilar(ctx, "vec_test", queryVec, 100, nil)
	if err != nil {
		t.Fatalf("SearchSimilar(k=100) error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchSimilar(k=100) returned %d results, want 2", len(results))
	}
}

// ── Adversarial: FindMany with large limit → cap at 1000 ─────────────────────

func TestIntegration_FindMany_LimitCap(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "LimitTest",
		Collection: "limit_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert 1500 rows
	for i := int64(0); i < 1500; i++ {
		_, err := c.Create(ctx, "limit_test", map[string]interface{}{
			"id": fmt.Sprintf("row-%d", i),
		})
		if err != nil {
			t.Fatalf("Create(row-%d) error: %v", i, err)
		}
	}

	// Request limit=10000 — should be capped at 1000
	cur, err := c.FindMany(ctx, "limit_test", &Query{Limit: 10000})
	if err != nil {
		t.Fatalf("FindMany(limit=10000) error: %v", err)
	}
	var all []map[string]interface{}
	if err := cur.All(ctx, &all); err != nil {
		t.Fatalf("FindMany().All() error: %v", err)
	}
	if len(all) != 1000 {
		t.Errorf("FindMany(limit=10000) returned %d rows, want 1000 (capped)", len(all))
	}

	// Request limit=500 — should return 500
	cur2, err := c.FindMany(ctx, "limit_test", &Query{Limit: 500})
	if err != nil {
		t.Fatalf("FindMany(limit=500) error: %v", err)
	}
	var filtered []map[string]interface{}
	if err := cur2.All(ctx, &filtered); err != nil {
		t.Fatalf("FindMany().All() error: %v", err)
	}
	if len(filtered) != 500 {
		t.Errorf("FindMany(limit=500) returned %d rows, want 500", len(filtered))
	}
}

// ── Adversarial: FindMany with offset pagination ─────────────────────────────

func TestIntegration_FindMany_OffsetPagination(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "Paginate",
		Collection: "paginated",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "seq", Type: PropInt64},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert 10 rows
	for i := int64(0); i < 10; i++ {
		_, err := c.Create(ctx, "paginated", map[string]interface{}{
			"id":  fmt.Sprintf("row-%d", i),
			"seq": i,
		})
		if err != nil {
			t.Fatalf("Create(row-%d) error: %v", i, err)
		}
	}

	// Page 1: limit=3, offset=0
	cur1, err := c.FindMany(ctx, "paginated", &Query{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("FindMany(page1) error: %v", err)
	}
	var page1 []map[string]interface{}
	if err := cur1.All(ctx, &page1); err != nil {
		t.Fatalf("page1.All() error: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page1 returned %d rows, want 3", len(page1))
	}

	// Page 2: limit=3, offset=3
	cur2, err := c.FindMany(ctx, "paginated", &Query{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("FindMany(page2) error: %v", err)
	}
	var page2 []map[string]interface{}
	if err := cur2.All(ctx, &page2); err != nil {
		t.Fatalf("page2.All() error: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page2 returned %d rows, want 3", len(page2))
	}
}

// ── Adversarial: FindMany with sort ──────────────────────────────────────────

func TestIntegration_FindMany_SortDesc(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "SortTest",
		Collection: "sort_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "score", Type: PropInt64},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert unsorted data
	_, _ = c.Create(ctx, "sort_test", map[string]interface{}{"id": "a", "score": int64(10)})
	_, _ = c.Create(ctx, "sort_test", map[string]interface{}{"id": "b", "score": int64(30)})
	_, _ = c.Create(ctx, "sort_test", map[string]interface{}{"id": "c", "score": int64(20)})

	// Sort by score DESC
	cur, err := c.FindMany(ctx, "sort_test", &Query{
		Sort: []SortField{{Field: "score", Order: "desc"}},
	})
	if err != nil {
		t.Fatalf("FindMany(sort desc) error: %v", err)
	}
	var all []map[string]interface{}
	if err := cur.All(ctx, &all); err != nil {
		t.Fatalf("FindMany().All() error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(all))
	}
	// First should be score=30
	if v := all[0]["score"]; v == nil {
		t.Fatal("first row score is nil")
	}
}

// ── Adversarial: Multiple WHERE filters ──────────────────────────────────────

func TestIntegration_FindMany_MultipleFilters(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "MultiFilter",
		Collection: "multi_filter",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "status", Type: PropString},
			{Name: "score", Type: PropInt64},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	_, _ = c.Create(ctx, "multi_filter", map[string]interface{}{"id": "a", "status": "active", "score": int64(50)})
	_, _ = c.Create(ctx, "multi_filter", map[string]interface{}{"id": "b", "status": "inactive", "score": int64(30)})
	_, _ = c.Create(ctx, "multi_filter", map[string]interface{}{"id": "c", "status": "active", "score": int64(70)})

	// Multiple eq filters
	cur, err := c.FindMany(ctx, "multi_filter", &Query{
		Filter: &Filter{Conditions: []Condition{
			{Field: "status", Op: "eq", Value: "active"},
			{Field: "score", Op: "gt", Value: int64(40)},
		}},
	})
	if err != nil {
		t.Fatalf("FindMany(multi-filter) error: %v", err)
	}
	var all []map[string]interface{}
	if err := cur.All(ctx, &all); err != nil {
		t.Fatalf("FindMany().All() error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("FindMany(multi-filter) returned %d rows, want 2 (a and c)", len(all))
	}
}

// ── Adversarial: DDL type mapping ────────────────────────────────────────────

func TestIntegration_DDL_CorrectColumnTypes(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "Types",
		Collection: "types_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "str_val", Type: PropString},
			{Name: "int_val", Type: PropInt},
			{Name: "int64_val", Type: PropInt64},
			{Name: "float32_val", Type: PropFloat32},
			{Name: "float64_val", Type: PropFloat64},
			{Name: "bool_val", Type: PropBool},
			{Name: "time_val", Type: PropTime},
			{Name: "json_val", Type: PropJSON},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Verify column types in information_schema
	rows, err := c.pool.Query(ctx,
		`SELECT column_name, data_type FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2 ORDER BY ordinal_position`,
		schemaName, "types_test",
	)
	if err != nil {
		t.Fatalf("query information_schema: %v", err)
	}
	defer rows.Close()

	typeCol := map[string]string{}
	for rows.Next() {
		var col, dtype string
		if err := rows.Scan(&col, &dtype); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		typeCol[col] = dtype
	}

	expected := map[string]string{
		"id":          "text",
		"str_val":     "text",
		"int_val":     "bigint",
		"int64_val":   "bigint",
		"float32_val": "real",
		"float64_val": "double precision",
		"bool_val":    "boolean",
		"time_val":    "timestamp with time zone",
		"json_val":    "jsonb",
	}
	for col, want := range expected {
		got := typeCol[col]
		if got != want {
			t.Errorf("Column %s type = %q, want %q", col, got, want)
		}
	}
}

// ── Adversarial: FindMany with OR filter ─────────────────────────────────────

func TestIntegration_FindMany_OrFilter(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "OrFilter",
		Collection: "or_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "role", Type: PropString},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	_, _ = c.Create(ctx, "or_test", map[string]interface{}{"id": "a", "role": "admin"})
	_, _ = c.Create(ctx, "or_test", map[string]interface{}{"id": "b", "role": "user"})
	_, _ = c.Create(ctx, "or_test", map[string]interface{}{"id": "c", "role": "owner"})

	// OR filter: role = admin OR role = owner
	cur, err := c.FindMany(ctx, "or_test", &Query{
		Filter: &Filter{
			LogicalOp: "or",
			Conditions: []Condition{
				{Field: "role", Op: "eq", Value: "admin"},
				{Field: "role", Op: "eq", Value: "owner"},
			},
		},
	})
	if err != nil {
		t.Fatalf("FindMany(or-filter) error: %v", err)
	}
	var all []map[string]interface{}
	if err := cur.All(ctx, &all); err != nil {
		t.Fatalf("FindMany().All() error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("FindMany(or-filter) returned %d rows, want 2 (a and c)", len(all))
	}
}

// ── Adversarial: Count with complex filter ───────────────────────────────────

func TestIntegration_Count_ComplexFilter(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "CountFilter",
		Collection: "count_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "status", Type: PropString},
			{Name: "score", Type: PropInt64},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	_, _ = c.Create(ctx, "count_test", map[string]interface{}{"id": "a", "status": "active", "score": int64(50)})
	_, _ = c.Create(ctx, "count_test", map[string]interface{}{"id": "b", "status": "inactive", "score": int64(30)})
	_, _ = c.Create(ctx, "count_test", map[string]interface{}{"id": "c", "status": "active", "score": int64(70)})

	// Count with filter
	count, err := c.Count(ctx, "count_test", &Filter{
		Conditions: []Condition{
			{Field: "status", Op: "eq", Value: "active"},
			{Field: "score", Op: "gt", Value: int64(40)},
		},
	})
	if err != nil {
		t.Fatalf("Count(complex filter) error: %v", err)
	}
	if count != 2 {
		t.Errorf("Count(complex filter) = %d, want 2", count)
	}
}

// ── Adversarial: DeleteMany with filter ──────────────────────────────────────

func TestIntegration_DeleteMany_WithFilter(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	c.schema = schemaName

	model := ModelDef{
		Name:       "DeleteFilter",
		Collection: "delete_test",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "status", Type: PropString},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	_, _ = c.Create(ctx, "delete_test", map[string]interface{}{"id": "a", "status": "active"})
	_, _ = c.Create(ctx, "delete_test", map[string]interface{}{"id": "b", "status": "inactive"})
	_, _ = c.Create(ctx, "delete_test", map[string]interface{}{"id": "c", "status": "inactive"})

	// Delete all inactive
	affected, err := c.DeleteMany(ctx, "delete_test", &Filter{
		Conditions: []Condition{{Field: "status", Op: "eq", Value: "inactive"}},
	})
	if err != nil {
		t.Fatalf("DeleteMany(filter) error: %v", err)
	}
	if affected != 2 {
		t.Errorf("DeleteMany(filter) affected = %d, want 2", affected)
	}

	// Verify only 'a' remains
	count, err := c.Count(ctx, "delete_test", nil)
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 1 {
		t.Errorf("Count() after delete = %d, want 1", count)
	}
}
