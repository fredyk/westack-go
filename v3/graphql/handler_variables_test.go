package graphql

import (
	"encoding/json"
	"net/http/httptest"
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
