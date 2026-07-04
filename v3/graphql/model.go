package graphql

// Model is the interface that a model must implement to support GraphQL operations.
type Model interface {
	SchemaHelper() *SchemaHelper
}

// ModelImpl is the concrete implementation of Model for GraphQL bindings.
type ModelImpl struct {
	Name   string
	schema *SchemaHelper
}

func (m *ModelImpl) SchemaHelper() *SchemaHelper {
	if m.schema == nil {
		m.schema = NewSchemaHelper()
	}
	return m.schema
}
