package datasource

import (
	"context"
	"testing"
)

// Tests for VectorConnector/SearchSimilar contract. These tests express the DESIRED
// behavior and FAIL because the stub returns "not implemented" errors.

func TestSearchSimilar_ReturnsResultsOrdered(t *testing.T) {
	c := &PostgresConnector{connected: true}
	vec := Vector{Values: []float32{0.1, 0.2, 0.3}, Dimensions: 3}
	filter := &Filter{Conditions: []Condition{
		{Field: "tenant_id", Op: "eq", Value: "t-1"},
	}}

	results, err := c.SearchSimilar(context.Background(), "knowledge_chunks", vec, 5, filter)
	if err != nil {
		t.Fatalf("SearchSimilar() error: %v", err)
	}
	if len(results) > 5 {
		t.Errorf("SearchSimilar() returned %d results, want at most 5", len(results))
	}
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("Results not ordered: result[%d].Distance=%f < result[%d].Distance=%f",
				i, results[i].Distance, i-1, results[i-1].Distance)
		}
	}
}

func TestSearchSimilar_WithNilFilter(t *testing.T) {
	c := &PostgresConnector{connected: true}
	vec := Vector{Values: []float32{0.1, 0.2}, Dimensions: 2}

	results, err := c.SearchSimilar(context.Background(), "embeddings", vec, 10, nil)
	if err != nil {
		t.Fatalf("SearchSimilar() error: %v", err)
	}
	_ = results
}

func TestSearchSimilar_HighDimensional(t *testing.T) {
	c := &PostgresConnector{connected: true}
	vals := make([]float32, 4096)
	for i := range vals {
		vals[i] = float32(i) / 4096.0
	}
	vec := Vector{Values: vals, Dimensions: 4096}
	// At 4096 dims, pgvector does NOT support ANN indexes. KNN is exact.
	results, err := c.SearchSimilar(context.Background(), "embeddings_4096", vec, 3, nil)
	if err != nil {
		t.Fatalf("SearchSimilar() error: %v", err)
	}
	if len(results) > 3 {
		t.Errorf("SearchSimilar() returned %d results, want at most 3", len(results))
	}
}

func TestCreateVectorIndex_WithSupportedDimensions(t *testing.T) {
	c := &PostgresConnector{connected: true}
	// 768 dims — within HNSW support range (<2000)
	err := c.CreateVectorIndex(context.Background(), "knowledge_chunks", "embedding", 768, "cosine")
	if err != nil {
		t.Fatalf("CreateVectorIndex() error: %v", err)
	}
}

func TestCreateVectorIndex_WithHighDimensions(t *testing.T) {
	c := &PostgresConnector{connected: true}
	// 4096 dims — exceeds 2000 limit for HNSW
	err := c.CreateVectorIndex(context.Background(), "embeddings_4096", "embedding", 4096, "cosine")
	_ = err
}
