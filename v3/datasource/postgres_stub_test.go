package datasource

import (
	"context"
	"testing"
)

// Tests for PostgresConnector stub methods. These tests express the DESIRED behavior
// (no errors, valid returns) and FAIL because the stub returns "not implemented" errors.
// This is the red phase of TDD.

func TestPostgresConnector_GetName(t *testing.T) {
	c := &PostgresConnector{}
	got := c.GetName()
	if got != "postgres" {
		t.Errorf("GetName() = %q, want %q", got, "postgres")
	}
}

func TestPostgresConnector_Connect_ActuallyWorks(t *testing.T) {
	c := &PostgresConnector{}
	err := c.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
}

func TestPostgresConnector_Disconnect_ActuallyWorks(t *testing.T) {
	c := &PostgresConnector{connected: true}
	err := c.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect() error: %v", err)
	}
}

func TestPostgresConnector_Ping_ActuallyWorks(t *testing.T) {
	c := &PostgresConnector{connected: true}
	err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestPostgresConnector_FindMany_ActuallyReturnsCursor(t *testing.T) {
	c := &PostgresConnector{connected: true}
	cur, err := c.FindMany(context.Background(), "clientes", &Query{
		Filter: &Filter{Conditions: []Condition{{Field: "active", Op: "eq", Value: true}}},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("FindMany() error: %v", err)
	}
	if cur == nil {
		t.Fatal("FindMany() returned nil cursor")
	}
}

func TestPostgresConnector_FindById_ActuallyReturnsDoc(t *testing.T) {
	c := &PostgresConnector{connected: true}
	doc, err := c.FindById(context.Background(), "clientes", "uuid-123")
	if err != nil {
		t.Fatalf("FindById() error: %v", err)
	}
	if doc == nil {
		t.Fatal("FindById() returned nil document")
	}
}

func TestPostgresConnector_Count_ActuallyReturnsCount(t *testing.T) {
	c := &PostgresConnector{connected: true}
	count, err := c.Count(context.Background(), "clientes", &Filter{
		Conditions: []Condition{{Field: "active", Op: "eq", Value: true}},
	})
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count < 0 {
		t.Errorf("Count() returned negative: %d", count)
	}
}

func TestPostgresConnector_Create_ActuallyReturnsDoc(t *testing.T) {
	c := &PostgresConnector{connected: true}
	doc, err := c.Create(context.Background(), "clientes", map[string]interface{}{
		"name":  "Test Cliente",
		"email": "test@example.com",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if doc == nil {
		t.Fatal("Create() returned nil document")
	}
}

func TestPostgresConnector_CreateMany_ActuallyReturnsDocs(t *testing.T) {
	c := &PostgresConnector{connected: true}
	docs, err := c.CreateMany(context.Background(), "clientes", []map[string]interface{}{
		{"name": "A"},
		{"name": "B"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("CreateMany() returned %d docs, want 2", len(docs))
	}
}

func TestPostgresConnector_UpdateById_ActuallyReturnsDoc(t *testing.T) {
	c := &PostgresConnector{connected: true}
	doc, err := c.UpdateById(context.Background(), "clientes", "uuid-123", map[string]interface{}{
		"name": "Updated",
	})
	if err != nil {
		t.Fatalf("UpdateById() error: %v", err)
	}
	if doc == nil {
		t.Fatal("UpdateById() returned nil document")
	}
}

func TestPostgresConnector_DeleteById_ActuallyReturnsAffected(t *testing.T) {
	c := &PostgresConnector{connected: true}
	affected, err := c.DeleteById(context.Background(), "clientes", "uuid-123")
	if err != nil {
		t.Fatalf("DeleteById() error: %v", err)
	}
	if affected < 0 {
		t.Errorf("DeleteById() returned negative: %d", affected)
	}
}

func TestPostgresConnector_DeleteMany_ActuallyReturnsAffected(t *testing.T) {
	c := &PostgresConnector{connected: true}
	affected, err := c.DeleteMany(context.Background(), "clientes", &Filter{
		Conditions: []Condition{{Field: "active", Op: "eq", Value: false}},
	})
	if err != nil {
		t.Fatalf("DeleteMany() error: %v", err)
	}
	if affected < 0 {
		t.Errorf("DeleteMany() returned negative: %d", affected)
	}
}

func TestPostgresConnector_Migrate_ActuallyCreatesSchema(t *testing.T) {
	c := &PostgresConnector{connected: true}
	err := c.Migrate(context.Background(), ModelDef{
		Name:       "Cliente",
		Collection: "clientes",
		Properties: []PropertyDef{
			{Name: "id", Type: PropString, PrimaryKey: true},
			{Name: "name", Type: PropString},
			{Name: "email", Type: PropString},
		},
	})
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
}

func TestPostgresConnector_ImplementsInterfaces(t *testing.T) {
	var _ PersistedConnector = (*PostgresConnector)(nil)
	var _ VectorConnector = (*PostgresConnector)(nil)
}
