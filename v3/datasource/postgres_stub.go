package datasource

import (
	"context"
	"errors"
)

// PostgresConnector is a stub implementing PersistedConnector and VectorConnector.
// All methods return "not implemented" until real implementation is done.
type PostgresConnector struct {
	connected bool
}

func (c *PostgresConnector) GetName() string {
	return "postgres"
}

func (c *PostgresConnector) Connect(ctx context.Context) error {
	return errors.New("not implemented: PostgresConnector.Connect")
}

func (c *PostgresConnector) Disconnect() error {
	return errors.New("not implemented: PostgresConnector.Disconnect")
}

func (c *PostgresConnector) Ping(ctx context.Context) error {
	return errors.New("not implemented: PostgresConnector.Ping")
}

func (c *PostgresConnector) FindMany(ctx context.Context, collection string, query *Query) (Cursor, error) {
	return nil, errors.New("not implemented: PostgresConnector.FindMany")
}

func (c *PostgresConnector) FindById(ctx context.Context, collection string, id interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.FindById")
}

func (c *PostgresConnector) Count(ctx context.Context, collection string, filter *Filter) (int64, error) {
	return 0, errors.New("not implemented: PostgresConnector.Count")
}

func (c *PostgresConnector) Create(ctx context.Context, collection string, data map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.Create")
}

func (c *PostgresConnector) CreateMany(ctx context.Context, collection string, data []map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.CreateMany")
}

func (c *PostgresConnector) UpdateById(ctx context.Context, collection string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented: PostgresConnector.UpdateById")
}

func (c *PostgresConnector) DeleteById(ctx context.Context, collection string, id interface{}) (int64, error) {
	return 0, errors.New("not implemented: PostgresConnector.DeleteById")
}

func (c *PostgresConnector) DeleteMany(ctx context.Context, collection string, filter *Filter) (int64, error) {
	return 0, errors.New("not implemented: PostgresConnector.DeleteMany")
}

func (c *PostgresConnector) Migrate(ctx context.Context, model ModelDef) error {
	return errors.New("not implemented: PostgresConnector.Migrate")
}

// SearchSimilar implements VectorConnector (stub).
func (c *PostgresConnector) SearchSimilar(ctx context.Context, collection string, vec Vector, k int, filter *Filter) ([]SimilarityResult, error) {
	return nil, errors.New("not implemented: PostgresConnector.SearchSimilar")
}

// CreateVectorIndex implements VectorConnector (stub).
func (c *PostgresConnector) CreateVectorIndex(ctx context.Context, collection string, column string, dims int, metric string) error {
	return errors.New("not implemented: PostgresConnector.CreateVectorIndex")
}

// Compile-time checks that PostgresConnector satisfies the interfaces.
var _ PersistedConnector = (*PostgresConnector)(nil)
var _ VectorConnector = (*PostgresConnector)(nil)
