package graphql

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// isInternalError debe reconocer los errores de infraestructura/BD (que arrastran
// SQL, nombres de tabla/columna, SQLSTATE, DSN, …) y NO confundirlos con errores
// de dominio/validación (que sí son seguros de mostrar al cliente).
func TestIsInternalError(t *testing.T) {
	internal := []string{
		`ERROR: column "visibilidad" of relation "conversation_annotations" does not exist (SQLSTATE 42703)`,
		`add column visibilidad: ERROR: ... (SQLSTATE 42703)`,
		`failed to connect to ` + "`host=db`",
		`not connected`,
		`dial tcp 10.0.0.1:5432: connect: connection refused`,
	}
	for _, m := range internal {
		if !isInternalError(m) {
			t.Errorf("isInternalError(%q) = false, want true (no debe filtrarse al cliente)", m)
		}
	}
	safe := []string{
		"cliente no encontrado",
		"NIE inválido",
		"la anotación ya existe",
		"unauthorized",
	}
	for _, m := range safe {
		if isInternalError(m) {
			t.Errorf("isInternalError(%q) = true, want false (es un error de dominio legítimo)", m)
		}
	}
}

// sanitizeClientError: los errores internos se sustituyen por un mensaje genérico
// (y NUNCA contienen SQLSTATE ni la palabra column/relation); los de dominio pasan
// tal cual.
func TestSanitizeClientError(t *testing.T) {
	raw := errors.New(`ERROR: column "visibilidad" of relation "conversation_annotations" does not exist (SQLSTATE 42703)`)
	got := sanitizeClientError("guardarAnotacion", raw)
	if strings.Contains(got, "SQLSTATE") || strings.Contains(got, "visibilidad") || strings.Contains(got, "relation") {
		t.Errorf("mensaje sanitizado filtra detalle SQL: %q", got)
	}
	if got == "" {
		t.Error("un error interno debe producir un mensaje genérico no vacío")
	}

	domain := errors.New("cliente no encontrado")
	if got := sanitizeClientError("expediente", domain); got != "cliente no encontrado" {
		t.Errorf("error de dominio no debe alterarse: got %q", got)
	}

	if got := sanitizeClientError("x", nil); got != "" {
		t.Errorf("nil error → mensaje vacío, got %q", got)
	}
}

// End-to-end: un resolver que devuelve un error de BD con SQLSTATE no debe filtrar
// ese detalle en la respuesta GraphQL enviada al cliente.
func TestServeGraphQL_DBErrorIsSanitized(t *testing.T) {
	m := &ModelImpl{}
	field := BindGraphQLMutationWithOptions(m, func(in testInput) (testResult, error) {
		return testResult{}, errors.New(`ERROR: column "visibilidad" of relation "conversation_annotations" does not exist (SQLSTATE 42703)`)
	}, &GraphQLOperationOptions{Name: "guardarAnotacion"})
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"mutation{guardarAnotacion(guardarAnotacion:{name:\"x\"}){id}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if strings.Contains(w.Body.String(), "SQLSTATE") || strings.Contains(w.Body.String(), "42703") {
		t.Errorf("la respuesta al cliente filtra SQLSTATE crudo: %s", w.Body.String())
	}
	var resp struct {
		Errors []map[string]any `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Errors) == 0 {
		t.Errorf("debe seguir habiendo un error (genérico) en la respuesta; body=%s", w.Body.String())
	}
}
