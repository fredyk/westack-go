//go:build integration

package datasource

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const defaultDSN = "postgres://ec:ecdev@127.0.0.1:15432/ec"

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("EC_PG_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}
	return dsn
}

func TestIntegration_Connect(t *testing.T) {
	c := NewPostgresConnector(testDSN(t), "public")
	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestIntegration_FullLifecycle(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	// Connect to the shared DB.
	c := NewPostgresConnector(dsn, "public")
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer c.Disconnect()

	// Create ephemeral schema for this test.
	schemaName := fmt.Sprintf("wsk_it_%d", time.Now().UnixNano())
	if _, err := c.pool.Exec(ctx, "CREATE SCHEMA "+pqQuoteIdent(schemaName)); err != nil {
		t.Fatalf("CREATE SCHEMA error: %v", err)
	}
	defer func() {
		c.pool.Exec(ctx, "DROP SCHEMA "+pqQuoteIdent(schemaName)+" CASCADE")
	}()

	// Reconnect targeting the ephemeral schema.
	c.schema = schemaName
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping after schema change error: %v", err)
	}

	// ── Migrate ───────────────────────────────────────────────────────
	model := ModelDef{
		Name:       "Cliente",
		Collection: "clientes",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
			{Name: "email", Type: PropString, Nullable: true},
			{Name: "embedding", Type: PropVector, Length: 3},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// ── Create ────────────────────────────────────────────────────────
	doc, err := c.Create(ctx, "clientes", map[string]interface{}{
		"id":    "c-001",
		"name":  "Alice",
		"email": "alice@example.com",
		"embedding": "[0.1,0.2,0.3]",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if doc["name"] != "Alice" {
		t.Errorf("Create() name = %v, want Alice", doc["name"])
	}

	doc2, err := c.Create(ctx, "clientes", map[string]interface{}{
		"id":    "c-002",
		"name":  "Bob",
		"email": "bob@example.com",
		"embedding": "[0.4,0.5,0.6]",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if doc2["name"] != "Bob" {
		t.Errorf("Create() name = %v, want Bob", doc2["name"])
	}

	doc3, err := c.Create(ctx, "clientes", map[string]interface{}{
		"id":    "c-003",
		"name":  "Charlie",
		"email": nil,
		"embedding": "[0.7,0.8,0.9]",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if doc3["name"] != "Charlie" {
		t.Errorf("Create() name = %v, want Charlie", doc3["name"])
	}

	// ── CreateMany ────────────────────────────────────────────────────
	docs, err := c.CreateMany(ctx, "clientes", []map[string]interface{}{
		{"id": "c-004", "name": "Diana", "embedding": "[0.15,0.25,0.35]"},
		{"id": "c-005", "name": "Eve", "embedding": "[0.45,0.55,0.65]"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("CreateMany() returned %d docs, want 2", len(docs))
	}

	// ── FindById ──────────────────────────────────────────────────────
	found, err := c.FindById(ctx, "clientes", "c-001")
	if err != nil {
		t.Fatalf("FindById() error: %v", err)
	}
	if found == nil {
		t.Fatal("FindById() returned nil")
	}
	if found["name"] != "Alice" {
		t.Errorf("FindById() name = %v, want Alice", found["name"])
	}

	// ── FindMany (all) ────────────────────────────────────────────────
	cur, err := c.FindMany(ctx, "clientes", &Query{Limit: 100})
	if err != nil {
		t.Fatalf("FindMany() error: %v", err)
	}
	var all []map[string]interface{}
	if err := cur.All(ctx, &all); err != nil {
		t.Fatalf("FindMany().All() error: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("FindMany() returned %d rows, want 5", len(all))
	}

	// ── FindMany (with filter) ────────────────────────────────────────
	cur2, err := c.FindMany(ctx, "clientes", &Query{
		Filter: &Filter{Conditions: []Condition{
			{Field: "name", Op: "eq", Value: "Alice"},
		}},
	})
	if err != nil {
		t.Fatalf("FindMany(filter) error: %v", err)
	}
	var filtered []map[string]interface{}
	if err := cur2.All(ctx, &filtered); err != nil {
		t.Fatalf("FindMany(filter).All() error: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("FindMany(filter) returned %d rows, want 1", len(filtered))
	}

	// ── Count ─────────────────────────────────────────────────────────
	count, err := c.Count(ctx, "clientes", nil)
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 5 {
		t.Errorf("Count() = %d, want 5", count)
	}

	countFiltered, err := c.Count(ctx, "clientes", &Filter{
		Conditions: []Condition{{Field: "name", Op: "eq", Value: "Alice"}},
	})
	if err != nil {
		t.Fatalf("Count(filter) error: %v", err)
	}
	if countFiltered != 1 {
		t.Errorf("Count(filter) = %d, want 1", countFiltered)
	}

	// ── UpdateById ────────────────────────────────────────────────────
	updated, err := c.UpdateById(ctx, "clientes", "c-001", map[string]interface{}{
		"name": "Alice Updated",
	})
	if err != nil {
		t.Fatalf("UpdateById() error: %v", err)
	}
	if updated["name"] != "Alice Updated" {
		t.Errorf("UpdateById() name = %v, want 'Alice Updated'", updated["name"])
	}

	// ── DeleteById ────────────────────────────────────────────────────
	affected, err := c.DeleteById(ctx, "clientes", "c-005")
	if err != nil {
		t.Fatalf("DeleteById() error: %v", err)
	}
	if affected != 1 {
		t.Errorf("DeleteById() affected = %d, want 1", affected)
	}

	// ── DeleteMany ────────────────────────────────────────────────────
	affected, err = c.DeleteMany(ctx, "clientes", &Filter{
		Conditions: []Condition{{Field: "name", Op: "eq", Value: "Diana"}},
	})
	if err != nil {
		t.Fatalf("DeleteMany() error: %v", err)
	}
	if affected != 1 {
		t.Errorf("DeleteMany() affected = %d, want 1", affected)
	}

	// ── Final count ───────────────────────────────────────────────────
	countFinal, err := c.Count(ctx, "clientes", nil)
	if err != nil {
		t.Fatalf("Count(final) error: %v", err)
	}
	if countFinal != 3 {
		t.Errorf("Count(final) = %d, want 3 (Alice Updated, Bob, Charlie)", countFinal)
	}

	// ── SearchSimilar (KNN) ───────────────────────────────────────────
	queryVec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	results, err := c.SearchSimilar(ctx, "clientes", queryVec, 3, nil)
	if err != nil {
		t.Fatalf("SearchSimilar() error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SearchSimilar() returned 0 results")
	}
	if len(results) > 3 {
		t.Errorf("SearchSimilar() returned %d results, want at most 3", len(results))
	}
	// c-001 embedding is [0.1,0.2,0.3] — closest to query
	if results[0].ID != "c-001" {
		t.Errorf("SearchSimilar() first result ID = %q, want c-001", results[0].ID)
	}
	// Results should be ordered by ascending distance
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("Results not ordered: result[%d].Distance=%f < result[%d].Distance=%f",
				i, results[i].Distance, i-1, results[i-1].Distance)
		}
	}
	t.Logf("SearchSimilar results: %+v", results)
}

func TestIntegration_SearchSimilar_WithFilter(t *testing.T) {
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
		Name:       "Documento",
		Collection: "documentos",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "content", Type: PropString},
			{Name: "tenant_id", Type: PropString},
			{Name: "embedding", Type: PropVector, Length: 3},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Insert docs from two tenants
	for _, d := range []struct {
		id, content, tenant string
		vec                 string
	}{
		{"d1", "About AI", "t1", "[0.1,0.2,0.3]"},
		{"d2", "About cats", "t1", "[0.4,0.5,0.6]"},
		{"d3", "About AI", "t2", "[0.15,0.25,0.35]"},
	} {
		_, err := c.Create(ctx, "documentos", map[string]interface{}{
			"id":        d.id,
			"content":   d.content,
			"tenant_id": d.tenant,
			"embedding": d.vec,
		})
		if err != nil {
			t.Fatalf("Create(%s) error: %v", d.id, err)
		}
	}

	// Search with tenant filter — should only return t1 docs
	queryVec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	filter := &Filter{Conditions: []Condition{
		{Field: "tenant_id", Op: "eq", Value: "t1"},
	}}
	results, err := c.SearchSimilar(ctx, "documentos", queryVec, 10, filter)
	if err != nil {
		t.Fatalf("SearchSimilar() error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchSimilar(filtered) returned %d results, want 2", len(results))
	}
	for _, r := range results {
		t.Logf("SearchSimilar filtered result: id=%s distance=%f", r.ID, r.Distance)
	}
}

func TestIntegration_CreateVectorIndex(t *testing.T) {
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
		Name:       "Embedding",
		Collection: "embeddings",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "vec", Type: PropVector, Length: 128},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Create HNSW index (128 dims < 2000 limit)
	if err := c.CreateVectorIndex(ctx, "embeddings", "vec", 128, "cosine"); err != nil {
		t.Fatalf("CreateVectorIndex(128) error: %v", err)
	}

	// Verify index exists
	var idxExists bool
	err := c.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE schemaname=$1 AND tablename=$2 AND indexname LIKE '%hnsw%')`,
		schemaName, "embeddings",
	).Scan(&idxExists)
	if err != nil {
		t.Fatalf("check index: %v", err)
	}
	if !idxExists {
		t.Error("HNSW index not found after CreateVectorIndex")
	}

	// 4096 dims — should skip (no-op) without error
	if err := c.CreateVectorIndex(ctx, "embeddings", "vec", 4096, "cosine"); err != nil {
		t.Fatalf("CreateVectorIndex(4096) error: %v", err)
	}
}

func TestIntegration_FindById_NotFound(t *testing.T) {
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
		Name:       "Item",
		Collection: "items",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	found, err := c.FindById(ctx, "items", "nonexistent")
	if err != nil {
		t.Fatalf("FindById(nonexistent) error: %v", err)
	}
	if found != nil {
		t.Errorf("FindById(nonexistent) = %v, want nil", found)
	}
}

func TestIntegration_DeleteById_NotFound(t *testing.T) {
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
		Name:       "Item",
		Collection: "items",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
		},
	}
	if err := c.Migrate(ctx, model); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	affected, err := c.DeleteById(ctx, "items", "nonexistent")
	if err != nil {
		t.Fatalf("DeleteById(nonexistent) error: %v", err)
	}
	if affected != 0 {
		t.Errorf("DeleteById(nonexistent) affected = %d, want 0", affected)
	}
}

// Ensure pgx.ErrNoRows is handled correctly (import used).
var _ = pgx.ErrNoRows
