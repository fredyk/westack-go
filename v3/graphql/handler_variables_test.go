package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// #34b: el input-object pasado por VARIABLES (`op(op: $in)` + variables:{in:{...}})
// debe poblar el struct del resolver igual que el caso inline. Antes de resolveValue
// recursivo, la referencia "$in" no se sustituía dentro del argumento nombrado.
func TestServeGraphQL_VariablePathInputObjectReachesResolver(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithOptions(m, func(in testInput) (testResult, error) {
		return testResult{Name: in.Name}, nil
	}, &GraphQLOperationOptions{Name: "makeThing"})
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"mutation($in: X){makeThing(makeThing:$in){id name}}","variables":{"in":{"name":"viavar"}}}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data   map[string]testResult `json:"data"`
		Errors []map[string]any      `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("graphql errors: %v", resp.Errors)
	}
	if got := resp.Data["makeThing"]; got.Name != "viavar" {
		t.Errorf("variable-path input NO llegó al resolver: Name esperado 'viavar', got %q (body=%s)", got.Name, w.Body.String())
	}
}

// Un panic en el resolver debe convertirse en `errors` (HTTP 200), no tumbar el server.
func TestServeGraphQL_ResolverPanicBecomesError(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithOptions(m, func(in testInput) (testResult, error) {
		panic("boom")
	}, &GraphQLOperationOptions{Name: "boomOp"})
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"mutation{boomOp(boomOp:{name:\"x\"}){id}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("panic debería dar 200+errors, got %d", w.Code)
	}
	var resp struct {
		Errors []map[string]any `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Errors) == 0 {
		t.Errorf("un panic del resolver debe aparecer en errors; body=%s", w.Body.String())
	}
}

func TestGraphQLServer_RegisterFields(t *testing.T) {
	m := &ModelImpl{}
	f1 := BindGraphQLQueryWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "op1"})
	f2 := BindGraphQLMutationWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "op2"})

	srv := NewGraphQLServer(m)
	srv.RegisterFields(f1, f2)

	// Both should be registered
	if len(srv.resolvers) != 2 {
		t.Errorf("expected 2 resolvers, got %d", len(srv.resolvers))
	}
	if _, ok := srv.resolvers["op1"]; !ok {
		t.Error("op1 not registered")
	}
	if _, ok := srv.resolvers["op2"]; !ok {
		t.Error("op2 not registered")
	}
}

func TestGraphQLServer_RegisterFields_NilField(t *testing.T) {
	m := &ModelImpl{}
	f1 := BindGraphQLQueryWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "op1"})

	srv := NewGraphQLServer(m)
	srv.RegisterFields(nil, f1, nil)

	if len(srv.resolvers) != 1 {
		t.Errorf("expected 1 resolver (nil fields skipped), got %d", len(srv.resolvers))
	}
}

func TestGraphQLServer_RegisterField_NilField(t *testing.T) {
	srv := NewGraphQLServer(&ModelImpl{})
	srv.RegisterField(nil)
	if len(srv.resolvers) != 0 {
		t.Errorf("nil field should not register, got %d resolvers", len(srv.resolvers))
	}
}

func TestGraphQLServer_RegisterField_NilHandler(t *testing.T) {
	srv := NewGraphQLServer(&ModelImpl{})
	srv.RegisterField(&GraphQLField{Name: "nope", Handler: nil})
	if len(srv.resolvers) != 0 {
		t.Errorf("nil handler should not register, got %d resolvers", len(srv.resolvers))
	}
}

func TestServeSDL(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLQueryWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "testQuery"})
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	r := httptest.NewRequest("GET", "/graphql/sdl", nil)
	w := httptest.NewRecorder()
	srv.ServeSDL(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("ServeSDL status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/plain; charset=utf-8")
	}
	body := w.Body.String()
	if body == "" {
		t.Error("ServeSDL body is empty")
	}
	if !strings.Contains(body, "testQuery") {
		t.Errorf("SDL should contain testQuery: %s", body)
	}
}

func TestServeGraphiQL(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	r := httptest.NewRequest("GET", "/graphiql", nil)
	w := httptest.NewRecorder()
	srv.ServeGraphiQL(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("ServeGraphiQL status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}
	body := w.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("ServeGraphiQL body should contain DOCTYPE")
	}
	if !strings.Contains(body, "graphiql") {
		t.Error("ServeGraphiQL body should contain 'graphiql'")
	}
}

func TestServeGraphQL_MethodNotAllowed(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	r := httptest.NewRequest("GET", "/graphql", nil)
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /graphql status = %d, want 405", w.Code)
	}
}

func TestServeGraphQL_InvalidJSON(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	r := httptest.NewRequest("POST", "/graphql", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON status = %d, want 400", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}
}

func TestServeGraphQL_UnknownOperation(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	body := `{"query":"{unknownOp(id:\"1\"){id}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("unknown operation status = %d, want 404", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}
}

func TestServeGraphQL_EmptyOperationName(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLQueryWithOptions(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	}, &GraphQLOperationOptions{Name: "someField"})
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	// Valid query with a registered operation that exists
	body := `{"query":"{someField(id:\"1\"){id}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("valid operation status = %d, want 200", w.Code)
	}
}

func TestCallResolver_NonFunctionHandler(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	result, err := srv.callResolver(context.Background(), "not a function", nil)
	if err == nil {
		t.Error("expected error for non-function handler")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "not a function") {
		t.Errorf("error message should contain 'not a function': %v", err)
	}
}

func TestCallResolver_WrongArgCount(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	handler := func(a, b int) (string, error) {
		return "", nil
	}
	result, err := srv.callResolver(context.Background(), handler, nil)
	if err == nil {
		t.Error("expected error for wrong arg count")
	}
	if !strings.Contains(err.Error(), "exactly one argument") {
		t.Errorf("error should mention 'exactly one argument': %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestCallResolver_WrongReturnCount(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	handler := func(req testInput) int {
		return 42
	}
	result, err := srv.callResolver(context.Background(), handler, nil)
	if err == nil {
		t.Error("expected error for wrong return count")
	}
	if !strings.Contains(err.Error(), "return (R, error)") {
		t.Errorf("error should mention 'return (R, error)': %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestIsRemoteOpReq_NonPointer(t *testing.T) {
	// testInput is not a pointer → should return false
	result := isRemoteOpReq(reflect.TypeOf(testInput{}))
	if result {
		t.Error("non-pointer type should not be RemoteOperationReq")
	}
}

func TestIsRemoteOpReq_PointerToNonStruct(t *testing.T) {
	// *string is a pointer but not to a struct
	result := isRemoteOpReq(reflect.TypeOf((*string)(nil)))
	if result {
		t.Error("pointer to non-struct should not be RemoteOperationReq")
	}
}

func TestIsRemoteOpReq_StructMissingCtx(t *testing.T) {
	type noCtx struct {
		Input int
	}
	result := isRemoteOpReq(reflect.TypeOf(&noCtx{}))
	if result {
		t.Error("struct missing Ctx field should not be RemoteOperationReq")
	}
}

func TestIsRemoteOpReq_StructCtxWrongType(t *testing.T) {
	type wrongCtx struct {
		Ctx int
	}
	result := isRemoteOpReq(reflect.TypeOf(&wrongCtx{}))
	if result {
		t.Error("struct with Ctx of wrong type should not be RemoteOperationReq")
	}
}

func TestIsRemoteOpReq_Valid(t *testing.T) {
	result := isRemoteOpReq(reflect.TypeOf(&RemoteOperationReq[testInput]{}))
	if !result {
		t.Error("valid RemoteOperationReq should return true")
	}
}
