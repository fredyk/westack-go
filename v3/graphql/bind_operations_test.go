package graphql

import (
	"strings"
	"testing"
)

func TestBindGraphQLOperation_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperation(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLOperation returned nil")
	}
}

func TestBindGraphQLOperationWithContext_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Error("BindGraphQLOperationWithContext returned nil")
	}
}

func TestBindGraphQLQuery_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLQuery returned nil")
	}
}

func TestBindGraphQLMutation_StubsExist(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutation(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Error("BindGraphQLMutation returned nil")
	}
}

func TestBindGraphQLMutationWithOptions_ExplicitName(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "crearExpediente"})
	if field == nil {
		t.Fatal("BindGraphQLMutationWithOptions returned nil")
	}
	if field.Name != "crearExpediente" {
		t.Errorf("expected op name crearExpediente, got %q", field.Name)
	}
	if field.Kind != "mutation" {
		t.Errorf("expected kind mutation, got %q", field.Kind)
	}
	// El SDL auto-generado debe contener la mutation con ese nombre.
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "crearExpediente") {
		t.Errorf("SDL should declare crearExpediente. SDL:\n%s", sdl)
	}
}

func TestBindGraphQLQueryWithOptions_ExplicitName(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLQueryWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "expedientes"})
	if field == nil || field.Name != "expedientes" || field.Kind != "query" {
		t.Fatalf("expected query op 'expedientes', got %+v", field)
	}
}

