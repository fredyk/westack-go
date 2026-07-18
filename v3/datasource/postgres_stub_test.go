package datasource

import (
	"context"
	"strings"
	"testing"
)

// Tests for PostgresConnector stub: only pure unit tests that don't require a DB.
// The "ActuallyWorks" tests have moved to postgres_integration_test.go.

func TestStubPostgresConnector_GetName(t *testing.T) {
	c := &stubPostgresConnector{}
	got := c.GetName()
	if got != "postgres" {
		t.Errorf("GetName() = %q, want %q", got, "postgres")
	}
}

func TestStubPostgresConnector_ImplementsInterfaces(t *testing.T) {
	var _ PersistedConnector = (*stubPostgresConnector)(nil)
	var _ VectorConnector = (*stubPostgresConnector)(nil)
}

func TestStubPostgresConnector_Connect(t *testing.T) {
	c := &stubPostgresConnector{}
	err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error should contain 'not implemented': %v", err)
	}
}

func TestStubPostgresConnector_Disconnect(t *testing.T) {
	c := &stubPostgresConnector{}
	err := c.Disconnect()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_Ping(t *testing.T) {
	c := &stubPostgresConnector{}
	err := c.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_FindMany(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.FindMany(context.Background(), "users", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_FindById(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.FindById(context.Background(), "users", "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_Count(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.Count(context.Background(), "users", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_Create(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.Create(context.Background(), "users", map[string]interface{}{"name": "Alice"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_CreateMany(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.CreateMany(context.Background(), "users", []map[string]interface{}{{"name": "Alice"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_UpdateById(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.UpdateById(context.Background(), "users", "1", map[string]interface{}{"name": "Bob"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_DeleteById(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.DeleteById(context.Background(), "users", "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_DeleteMany(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.DeleteMany(context.Background(), "users", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_Migrate(t *testing.T) {
	c := &stubPostgresConnector{}
	err := c.Migrate(context.Background(), ModelDef{Name: "User", Collection: "users"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_SearchSimilar(t *testing.T) {
	c := &stubPostgresConnector{}
	_, err := c.SearchSimilar(context.Background(), "embeddings", Vector{Values: []float32{0.1, 0.2}}, 10, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStubPostgresConnector_CreateVectorIndex(t *testing.T) {
	c := &stubPostgresConnector{}
	err := c.CreateVectorIndex(context.Background(), "embeddings", "embedding", 1536, "cosine")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
