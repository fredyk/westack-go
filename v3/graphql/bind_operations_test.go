package graphql

import (
	"testing"
)

// localPlaceholder is a minimal stub model used by Bind* stubs.
type localPlaceholder struct {
	name   string
	schema *SchemaHelper
}

func (m *localPlaceholder) SchemaHelper() *SchemaHelper {
	if m.schema == nil {
		m.schema = NewSchemaHelper()
	}
	return m.schema
}

func TestBindGraphQLOperation_StubsExist(t *testing.T) {
	m := &localPlaceholder{}
	field := BindGraphQLOperation(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLOperation returned nil")
	}
}

func TestBindGraphQLOperationWithContext_StubsExist(t *testing.T) {
	m := &localPlaceholder{}
	field := BindGraphQLOperationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Error("BindGraphQLOperationWithContext returned nil")
	}
}

func TestBindGraphQLQuery_StubsExist(t *testing.T) {
	m := &localPlaceholder{}
	field := BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLQuery returned nil")
	}
}

func TestBindGraphQLMutation_StubsExist(t *testing.T) {
	m := &localPlaceholder{}
	field := BindGraphQLMutation(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLMutation returned nil")
	}
}
