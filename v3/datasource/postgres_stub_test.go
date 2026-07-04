package datasource

import "testing"

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
