package datasource

import "context"

// Vector represents an embedding vector for pgvector.
// Dimensions must match the vector(N) column definition.
// NOTE: at 4096 dims, pgvector does NOT support ANN indexes (>2000 dims limit for HNSW).
// KNN search at 4096 dims is always exact (brute-force). The contract does NOT assume HNSW.
type Vector struct {
	Values     []float32
	Dimensions int
}

// SimilarityResult is a single result from a KNN search.
type SimilarityResult struct {
	ID         string
	Distance   float32
	Score      float32 // 1 - distance for cosine, or raw dot product
	Document   interface{}
}

// VectorSearchOptions configures a KNN search.
type VectorSearchOptions struct {
	// MaxDimsForIndex is the threshold above which exact KNN is used instead of ANN.
	// Default: 2000. At 4096 dims, this is always exact search.
	MaxDimsForIndex int
}

// VectorConnector extends PersistedConnector with vector search capabilities (pgvector).
type VectorConnector interface {
	PersistedConnector

	// SearchSimilar performs a KNN search on the given collection's vector column.
	// vec is the query vector, k is the number of results, filter constrains results.
	// Returns results ordered by ascending distance.
	SearchSimilar(ctx context.Context, collection string, vec Vector, k int, filter *Filter) ([]SimilarityResult, error)

	// CreateVectorIndex creates an ANN index (HNSW) on the given column, but only if
	// dimensions <= MaxDimsForIndex (2000). For larger vectors, exact search is used.
	CreateVectorIndex(ctx context.Context, collection string, column string, dims int, metric string) error
}
