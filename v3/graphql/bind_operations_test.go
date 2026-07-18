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

func TestBindGraphQLOperationWithOptions(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperationWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "customQuery", Description: "A custom query"})
	if field == nil {
		t.Fatal("BindGraphQLOperationWithOptions returned nil")
	}
	if field.Name != "customQuery" {
		t.Errorf("Name = %q, want %q", field.Name, "customQuery")
	}
	if field.Kind != "query" {
		t.Errorf("Kind = %q, want %q", field.Kind, "query")
	}
	if field.Handler == nil {
		t.Error("Handler should not be nil")
	}
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "customQuery") {
		t.Errorf("SDL should contain customQuery. SDL:\n%s", sdl)
	}
}

func TestBindGraphQLOperationWithOptions_NilOptions(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperationWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLOperationWithOptions(nil opts) returned nil")
	}
	if field.Kind != "query" {
		t.Errorf("Kind = %q, want %q", field.Kind, "query")
	}
}

func TestBindGraphQLMutationWithContext(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "createUser"})
	if field == nil {
		t.Fatal("BindGraphQLMutationWithContext returned nil")
	}
	if field.Name != "createUser" {
		t.Errorf("Name = %q, want %q", field.Name, "createUser")
	}
	if field.Kind != "mutation" {
		t.Errorf("Kind = %q, want %q", field.Kind, "mutation")
	}
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "createUser") {
		t.Errorf("SDL should contain createUser. SDL:\n%s", sdl)
	}
}

func TestBindGraphQLMutationWithContext_NilOptions(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLMutationWithContext(nil opts) returned nil")
	}
	if field.Kind != "mutation" {
		t.Errorf("Kind = %q, want %q", field.Kind, "mutation")
	}
}

func TestBindGraphQLOperationWithContext_NilOptions(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLOperationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLOperationWithContext(nil opts) returned nil")
	}
}

func TestBindGraphQLOperation_NilModel(t *testing.T) {
	field := BindGraphQLOperation[testInput, testResult](nil, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Fatal("BindGraphQLOperation(nil model) returned nil")
	}
	if field.Kind != "query" {
		t.Errorf("nil model Kind = %q, want %q", field.Kind, "query")
	}
}

func TestBindGraphQLMutation_NilModel(t *testing.T) {
	field := BindGraphQLMutation[testInput, testResult](nil, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Fatal("BindGraphQLMutation(nil model) returned nil")
	}
}

func TestBindGraphQLQuery_NilModel(t *testing.T) {
	field := BindGraphQLQuery[testInput, testResult](nil, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if field == nil {
		t.Fatal("BindGraphQLQuery(nil model) returned nil")
	}
}

func TestBindGraphQLOperationWithContext_NilModel(t *testing.T) {
	field := BindGraphQLOperationWithContext[testInput, testResult](nil, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLOperationWithContext(nil model) returned nil")
	}
}

func TestBindGraphQLMutationWithContext_NilModel(t *testing.T) {
	field := BindGraphQLMutationWithContext[testInput, testResult](nil, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLMutationWithContext(nil model) returned nil")
	}
}

func TestBindGraphQLQueryWithOptions_NilOptions(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLQueryWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLQueryWithOptions(nil opts) returned nil")
	}
	if field.Kind != "query" {
		t.Errorf("Kind = %q, want %q", field.Kind, "query")
	}
}

func TestBindGraphQLMutationWithOptions_NilOptions(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, nil)
	if field == nil {
		t.Fatal("BindGraphQLMutationWithOptions(nil opts) returned nil")
	}
	if field.Kind != "mutation" {
		t.Errorf("Kind = %q, want %q", field.Kind, "mutation")
	}
}

func TestNameOr_ExplicitName(t *testing.T) {
	got := nameOr("custom", func() {})
	if got != "custom" {
		t.Errorf("nameOr with explicit: got %q, want %q", got, "custom")
	}
}

func TestNameOr_EmptyName(t *testing.T) {
	got := nameOr("", func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	if got == "" {
		t.Error("nameOr with empty name should derive from handler, got empty")
	}
}

func TestGetFunctionName(t *testing.T) {
	fn := func(req testInput) (testResult, error) {
		return testResult{}, nil
	}
	name := GetFunctionName(fn)
	if name == "" {
		t.Error("GetFunctionName returned empty string")
	}
}

func TestSplitPath(t *testing.T) {
	parts := splitPath("github.com/fredyk/westack-go/v3/graphql.GetFunctionName")
	last := parts[len(parts)-1]
	if last != "GetFunctionName" {
		t.Errorf("splitPath last = %q, want %q", last, "GetFunctionName")
	}
}

