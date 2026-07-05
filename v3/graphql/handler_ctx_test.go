package graphql

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

type ctxTestKey string

// Un handler registrado con BindGraphQLOperationWithContext debe recibir, por
// petición: (a) el context.Context de la request HTTP (para auth/cancelación) y
// (b) el input decodificado de los argumentos GraphQL. Antes de #24 el wrapper
// no llevaba Ctx y callResolver no lo poblaba → este test falla.
func TestServeGraphQL_ContextAndInputFlowToHandler(t *testing.T) {
	const key ctxTestKey = "principal"
	m := &ModelImpl{}
	field := BindGraphQLOperationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		principal := ""
		if req != nil && req.Ctx != nil {
			principal, _ = req.Ctx.Value(key).(string)
		}
		name := ""
		if req != nil {
			name = req.Input.Name
		}
		return testResult{ID: principal, Name: name}, nil
	}, &GraphQLOperationOptions{Name: "whoami"})

	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{whoami(name:\"bob\",age:1){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), key, "alice"))
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
	got := resp.Data["whoami"]
	if got.ID != "alice" {
		t.Errorf("ctx NO fluyó: principal esperado 'alice' en ID, got %q (body=%s)", got.ID, w.Body.String())
	}
	if got.Name != "bob" {
		t.Errorf("input NO fluyó: Name esperado 'bob', got %q", got.Name)
	}
}
