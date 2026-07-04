package datasource

import (
	"context"
	"errors"
)

// stubPostgresConnector is a stub implementing PersistedConnector and VectorConnector.
// All methods return "not implemented" until real implementation is done.
// This stub exists to express the DESIRED behavior in unit tests (red phase of TDD).
type stubPostgresConnector struct {
	connected bool
}

func (c *stubPostgresConnector) GetName() string {
	return "postgres"
}

func (c *stubPostgresConnector) Connect(ctx context.Context) error {
	return errors.New("not implemented: PostgresConnector.Connect")
}

func (c *stubPostgresConnector) Disconnect() error {
	return errors.New("not implemented: PostgresConnector.Disconnect")
}

func (c *stubPostgresConnector) Ping(ctx context.Context) error {
	return errors.New("not implemented: PostgresConnector.Ping")
}

func (c *stubPostgresConnector) FindMany(ctx context.Context, collection string, query *Query) (Cursor, error) {
	return nil, errors.New("not implemented: PostgresConnector.FindMany")
}

func (c *stubPostgresConnector) FindById(ctx context.Context, collection string, id interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.FindById")
}

func (c *stubPostgresConnector) Count(ctx context.Context, collection string, filter *Filter) (int64, error) {
	return 0, errors.New("not implemented: PostgresConnector.Count")
}

func (c *stubPostgresConnector) Create(ctx context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.Create")
}

func (c *stubPostgresConnector) CreateMany(ctx context.Context, collection string, data []map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.CreateMany")
}

func (c *stubPostgresConnector) UpdateById(ctx context.Context, collection string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.UpdateById")
}

func (c *stubPostgresConnector) DeleteById(ctx context.Context, collection string, id interface{}) (int64, error) {
	return 0, errors.New("not implemented: PostgresConnector.DeleteById")
}

func (c *stubPostgresConnector) DeleteMany(ctx context.Context, collection string, filter *Filter) (int64, error) {
	return 0, errors.New("not implemented: PostgresConnector.DeleteMany")
}

func (c *stubPostgresConnector) Migrate(ctx context.Context, model ModelDef) error {
	return errors.New("not implemented: PostgresConnector.Migrate")
}

// SearchSimilar implements VectorConnector (stub).
func (c *stubPostgresConnector) SearchSimilar(ctx context.Context, collection string, vec Vector, k int, filter *Filter) ([]SimilarityResult, error) {
	return nil, errors.New("not implemented: PostgresConnector.SearchSimilar")
}

// CreateVectorIndex implements VectorConnector (stub).
func (c *stubPostgresConnector) CreateVectorIndex(ctx context.Context, collection string, column string, dims int, metric string) error {
	return errors.New("not implemented: PostgresConnector.CreateVectorIndex")
}

// Compile-time checks that stubPostgresConnector satisfies the interfaces.
var _ PersistedConnector = (*stubPostgresConnector)(nil)
var _ VectorConnector = (*stubPostgresConnector)(nil)
