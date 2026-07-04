package datasource

import "testing"

// Pure unit tests for vector types and contracts (no DB required).
// The SearchSimilar/CreateVectorIndex tests are in postgres_integration_test.go.

func TestVector_Fields(t *testing.T) {
	vec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	if vec.Dimensions != 3 {
		t.Errorf("Dimensions = %d, want 3", vec.Dimensions)
	}
	if len(vec.Values) != 3 {
		t.Errorf("len(Values) = %d, want 3", len(vec.Values))
	}
}

func TestSimilarityResult_Fields(t *testing.T) {
	sr := SimilarityResult{ID: "doc-1", Distance: 0.15, Score: 0.85}
	if sr.ID != "doc-1" {
		t.Errorf("ID = %q, want doc-1", sr.ID)
	}
	if sr.Distance != 0.15 {
		t.Errorf("Distance = %f, want 0.15", sr.Distance)
	}
	if sr.Score != 0.85 {
		t.Errorf("Score = %f, want 0.85", sr.Score)
	}
}

func TestVectorConnector_ImplementsPersistedConnector(t *testing.T) {
	var _ PersistedConnector = (*stubPostgresConnector)(nil)
}
