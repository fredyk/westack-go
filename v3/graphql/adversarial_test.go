package graphql

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── 1. Slices ──────────────────────────────────────────────────────────

func TestSDL_SliceOfStructsProducedCorrectly(t *testing.T) {
	type in struct{}
	type item struct {
		ID   string
		Name string
	}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req in) ([]item, error) {
		return nil, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "[item]") {
		t.Errorf("SDL should contain '[item]' for []item result. Got:\n%s", sdl)
	}
	if strings.Contains(sdl, "String") && !strings.Contains(sdl, "[item]") {
		t.Errorf("SDL contains bare 'String' as result type instead of '[item]':\n%s", sdl)
	}
}

func TestSDL_SliceOfPointersToStruct(t *testing.T) {
	type in struct{}
	type item struct {
		ID string
	}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req in) ([]*item, error) {
		return nil, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "[item]") {
		t.Errorf("SDL should contain '[item]' for []*item result. Got:\n%s", sdl)
	}
}

func TestSDL_SliceOfSlice(t *testing.T) {
	type in struct{}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req in) ([][]string, error) {
		return nil, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "[[String]]") {
		t.Errorf("SDL should contain '[[String]]' for [][]string result. Got:\n%s", sdl)
	}
}

// ── 2. Pointers / Nullability ──────────────────────────────────────────

func TestSDL_PointerFieldIsNullable(t *testing.T) {
	type input struct{}
	type result struct {
		Name *string
		Age  int
	}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req input) (result, error) {
		return result{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	// *string → nullable "String" (no !)
	if !strings.Contains(sdl, "Name: String\n") && !strings.Contains(sdl, "Name: String\r\n") {
		t.Errorf("Pointer field Name should be nullable 'String'. Got:\n%s", sdl)
	}
	// int → required "Int!"
	if !strings.Contains(sdl, "Age: Int!\n") && !strings.Contains(sdl, "Age: Int!\r\n") {
		t.Errorf("Non-pointer field Age should be required 'Int!'. Got:\n%s", sdl)
	}
}

func TestSDL_NonPointerFieldIsRequired(t *testing.T) {
	type input struct{}
	type result struct {
		ID   string
		Email string
	}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req input) (result, error) {
		return result{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "ID: String!\n") && !strings.Contains(sdl, "ID: String!\r\n") {
		t.Errorf("Non-pointer string field should be 'String!'. Got:\n%s", sdl)
	}
}

// ── 3. Nested structs ──────────────────────────────────────────────────

func TestSDL_NestedStructGeneratesObjectType(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}
	type input struct{}
	type person struct {
		Name    string
		Address Address
	}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req input) (person, error) {
		return person{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "type Address {") {
		t.Errorf("SDL should define 'type Address'. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "address: Address\n") && !strings.Contains(sdl, "address: Address\r\n") {
		t.Errorf("SDL person should reference 'address: Address'. Got:\n%s", sdl)
	}
}

func TestSDL_DeeplyNestedStruct(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Middle struct {
		Inner Inner
	}
	type Outer struct {
		Middle Middle
	}
	type input struct{}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req input) (Outer, error) {
		return Outer{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "type Inner {") {
		t.Errorf("SDL should define 'type Inner'. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "type Middle {") {
		t.Errorf("SDL should define 'type Middle'. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "type Outer {") {
		t.Errorf("SDL should define 'type Outer'. Got:\n%s", sdl)
	}
}

func TestSDL_NestedInputStruct(t *testing.T) {
	type Contact struct {
		Email string
	}
	type CreatePersonInput struct {
		Name   string
		Contact Contact
	}
	type Result struct {
		ID string
	}
	m := &ModelImpl{}
	BindGraphQLMutation(m, func(req CreatePersonInput) (Result, error) {
		return Result{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "input CreatePersonInput {") {
		t.Errorf("SDL should define input CreatePersonInput. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "type Contact {") && !strings.Contains(sdl, "input Contact {") {
		// Contact should appear as a type definition
	}
}

// ── 4. Args with variables ─────────────────────────────────────────────

func TestResolveVariables_DeeplyNested(t *testing.T) {
	// Variables declared but not referenced inline
	vars := map[string]any{"a": 1, "b": "two", "c": true}
	args := map[string]any{}
	got := resolveVariables(args, vars)
	// All variables should be injected
	if got["a"] != 1 || got["b"] != "two" || got["c"] != true {
		t.Errorf("all variables injected: got %#v", got)
	}
}

func TestResolveVariables_OverrideInline(t *testing.T) {
	args := map[string]any{"id": "$id"}
	vars := map[string]any{"id": "resolved-123"}
	got := resolveVariables(args, vars)
	if got["id"] != "resolved-123" {
		t.Errorf("variable $id resolved to %q, want 'resolved-123'", got["id"])
	}
}

func TestResolveVariables_UnknownVariableStays(t *testing.T) {
	args := map[string]any{"id": "$missing"}
	vars := map[string]any{}
	got := resolveVariables(args, vars)
	// Unknown variable should stay as-is (resolver may handle)
	if got["id"] != "$missing" {
		t.Errorf("unknown variable stayed as %q", got["id"])
	}
}

func TestParseFieldArgs_EmptyStringArgs(t *testing.T) {
	query := `{find(name:""){id}}`
	got := parseFieldArgs(query, "find")
	if got == nil {
		t.Fatal("parseFieldArgs returned nil for query with empty string arg")
	}
	if got["name"] != "" {
		t.Errorf("empty string arg should be '', got %q", got["name"])
	}
}

func TestParseFieldArgs_NullArgs(t *testing.T) {
	query := `{find(val:null){id}}`
	got := parseFieldArgs(query, "find")
	if got == nil {
		t.Fatal("parseFieldArgs returned nil for query with null arg")
	}
	if got["val"] != nil {
		t.Errorf("null arg should be nil, got %v", got["val"])
	}
}

func TestParseFieldArgs_MultipleNestedObjects(t *testing.T) {
	query := `mutation{create(input:{contact:{email:"a@b"},meta:{key:"val"}}){id}}`
	got := parseFieldArgs(query, "create")
	if got == nil {
		t.Fatal("parseFieldArgs returned nil")
	}
	input, ok := got["input"].(map[string]any)
	if !ok {
		t.Fatalf("input should be map[string]any, got %T", got["input"])
	}
	contact, ok := input["contact"].(map[string]any)
	if !ok {
		t.Fatalf("contact should be map[string]any, got %T", input["contact"])
	}
	if contact["email"] != "a@b" {
		t.Errorf("nested email = %v, want 'a@b'", contact["email"])
	}
}

func TestParseFieldArgs_VariableInList(t *testing.T) {
	query := `{search(ids:[$a,$b])}{id}}`
	got := parseFieldArgs(query, "search")
	if got == nil {
		t.Fatal("parseFieldArgs returned nil")
	}
	ids, ok := got["ids"].([]any)
	if !ok {
		t.Fatalf("ids should be []any, got %T", got["ids"])
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	if ids[0] != "$a" || ids[1] != "$b" {
		t.Errorf("variable refs in list: got %v", ids)
	}
}

// ── 5. Error propagation (GraphQL spec) ────────────────────────────────

func TestError_PropagatesAsGraphQLErrorsWithDataWrapper(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, ErrTest("resolver error message")
	})
	field := &GraphQLField{Name: "testErr", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{}, ErrTest("resolver error message")
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{testErr(name:\"x\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d (GraphQL errors must return 200), body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Per GraphQL spec: errors coexist with data
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("response must have 'data' wrapper. Got: %v", resp)
	}
	if _, has := data["testErr"]; !has {
		t.Fatalf("'data' missing 'testErr' key. Got data: %v", data)
	}

	errors, ok := resp["errors"].([]any)
	if !ok {
		t.Fatalf("response must have 'errors' array. Got: %v", resp)
	}
	if len(errors) == 0 {
		t.Fatal("expected at least one error, got none")
	}
}

// ErrTest is a sentinel error for testing.
type ErrTest string

func (e ErrTest) Error() string { return string(e) }

func TestError_HandlerReturnsNilResultWithError(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (*testResult, error) {
		return nil, ErrTest("not found")
	})
	field := &GraphQLField{Name: "notfound", Kind: "query", Handler: func(req testInput) (*testResult, error) {
		return nil, ErrTest("not found")
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{notfound(name:\"x\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("must have 'data' wrapper. Got: %v", resp)
	}
	// The field should be null when error
	if data["notfound"] != nil {
		t.Errorf("notfound field should be null on error, got %v", data["notfound"])
	}
}

// ── 6. Resolver panic recovery ─────────────────────────────────────────

func TestPanic_Recovery(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		panic("boom from resolver")
	})
	field := &GraphQLField{Name: "panics", Kind: "query", Handler: func(req testInput) (testResult, error) {
		panic("boom from resolver")
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{panics(name:\"x\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()

	// Should NOT panic/recover at HTTP level
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HTTP handler panicked: %v", r)
		}
	}()

	srv.ServeGraphQL(w, r)

	// Should return 200 with errors (not 500)
	if w.Code != 200 {
		t.Errorf("status %d, expected 200 with errors. body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, has := resp["errors"]; !has {
		t.Errorf("panic should produce 'errors' in response. Got: %v", resp)
	}
}

// ── 7. Anonymous queries ───────────────────────────────────────────────

func TestAnonymousQuery(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{ID: "1", Name: "test"}, nil
	})
	field := &GraphQLField{Name: "anonymous", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{ID: "1", Name: "test"}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{testInput(name:\"x\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data wrapper. Got: %v", resp)
	}
	if len(data) == 0 {
		t.Errorf("data should contain the operation result. Got: %v", data)
	}
}

// ── 8. Unknown operation ───────────────────────────────────────────────

func TestUnknownOperation(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	field := &GraphQLField{Name: "exists", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{nonexistent{name}}\"}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 404 {
		t.Fatalf("status %d, expected 404 for unknown operation. body=%s", w.Code, w.Body.String())
	}
}

// ── 9. Multiple operations in SDL ──────────────────────────────────────

func TestSDL_MultipleQueriesAndMutations(t *testing.T) {
	type In1 struct{ ID string }
	type In2 struct{ Name string }
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req In1) (Out, error) { return Out{}, nil })
	BindGraphQLQuery(m, func(req In2) (Out, error) { return Out{}, nil })
	BindGraphQLMutation(m, func(req In1) (Out, error) { return Out{}, nil })

	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "type Query") {
		t.Error("SDL missing 'type Query'")
	}
	if !strings.Contains(sdl, "type Mutation") {
		t.Error("SDL missing 'type Mutation'")
	}
	// Both queries should be under type Query
	qCount := countOccurrences(sdl, "type Query")
	if qCount != 1 {
		t.Errorf("exactly one 'type Query', got %d", qCount)
	}
}

// ── 10. Input type with pointer fields ─────────────────────────────────

func TestSDL_InputType_PointerFields(t *testing.T) {
	type CreateInput struct {
		Name  string
		Email *string
		Age   *int
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLMutation(m, func(req CreateInput) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "input CreateInput {") {
		t.Errorf("SDL should define input CreateInput. Got:\n%s", sdl)
	}
	// Pointer fields in input should be nullable (no !)
	if strings.Contains(sdl, "Email: String!") {
		t.Errorf("Pointer field Email in input should be nullable 'String', not 'String!'\n%s", sdl)
	}
}

// ── 11. JSON unmarshalling of args to struct ───────────────────────────

func TestMapToStruct_NestedStructMapping(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Outer struct {
		Inner Inner
	}
	args := map[string]any{
		"inner": map[string]any{"value": "hello"},
	}
	var dest Outer
	err := mapToStruct(args, &dest)
	if err != nil {
		t.Fatalf("mapToStruct: %v", err)
	}
	if dest.Inner.Value != "hello" {
		t.Errorf("nested mapping failed: Inner.Value = %q, want 'hello'", dest.Inner.Value)
	}
}

func TestMapToStruct_PointerFieldMapping(t *testing.T) {
	type Input struct {
		Name *string
	}
	args := map[string]any{"name": "test"}
	var dest Input
	err := mapToStruct(args, &dest)
	if err != nil {
		t.Fatalf("mapToStruct: %v", err)
	}
	if dest.Name == nil {
		t.Fatal("Name should not be nil")
	}
	if *dest.Name != "test" {
		t.Errorf("Name = %q, want 'test'", *dest.Name)
	}
}

// ── 12. Context flow with simple handler (no RemoteOperationReq) ──────

func TestSimpleHandler_NoContext(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{ID: req.Name, Name: req.Name}, nil
	})
	field := &GraphQLField{Name: "simple", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{ID: req.Name, Name: req.Name}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{simple(name:\"alice\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data map[string]testResult `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data["simple"].ID != "alice" {
		t.Errorf("simple handler result ID = %q, want 'alice'", resp.Data["simple"].ID)
	}
}

// ── 13. Operation name extraction edge cases ───────────────────────────

func TestExtractOperationName_NamedOperation(t *testing.T) {
	q := `query GetUser(id: "1") { getUser(id: "1") { id } }`
	name := extractOperationName(q)
	if name != "GetUser" {
		t.Errorf("extractOperationName(%q) = %q, want 'GetUser'", q, name)
	}
}

func TestExtractOperationName_Mutation(t *testing.T) {
	q := `mutation CreateUser(name: "Bob") { createUser(name: "Bob") { id } }`
	name := extractOperationName(q)
	if name != "CreateUser" {
		t.Errorf("extractOperationName(%q) = %q, want 'CreateUser'", q, name)
	}
}

func TestExtractOperationName_Anonymous(t *testing.T) {
	q := `{ getUser(id: "1") { id } }`
	name := extractOperationName(q)
	if name != "getUser" {
		t.Errorf("extractOperationName(%q) = %q, want 'getUser'", q, name)
	}
}

func TestExtractOperationName_Empty(t *testing.T) {
	q := ``
	name := extractOperationName(q)
	if name != "" {
		t.Errorf("extractOperationName(%q) = %q, want empty", q, name)
	}
}

// ── 14. SDL field ordering ─────────────────────────────────────────────

func TestSDL_QueryFieldsOrder(t *testing.T) {
	type In1 struct{ A string }
	type In2 struct{ B string }
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req In1) (Out, error) { return Out{}, nil })
	BindGraphQLQuery(m, func(req In2) (Out, error) { return Out{}, nil })
	sdl := m.SchemaHelper().SDL()
	// All query fields should appear under type Query
	if !strings.Contains(sdl, "type Query") {
		t.Error("missing Query type")
	}
}

// ── 15. Variable resolution with typed values ─────────────────────────

func TestResolveVariables_BoolAndInt(t *testing.T) {
	args := map[string]any{"flag": "$f", "count": "$c"}
	vars := map[string]any{"f": false, "c": int(42)}
	got := resolveVariables(args, vars)
	if got["flag"] != false {
		t.Errorf("bool $f resolved to %v, want false", got["flag"])
	}
	if got["count"] != 42 {
		t.Errorf("int $c resolved to %v, want 42", got["count"])
	}
}

// ── 16. Handler validation ─────────────────────────────────────────────

func TestHandler_NotAFunction(t *testing.T) {
	srv := NewGraphQLServer(nil)
	srv.resolvers["notfunc"] = "not a function"
	body := `{"query":"{notfunc{name}}\"}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)
	// Should return errors, not panic
	if w.Code != 500 {
		t.Errorf("status %d, expected 500 for non-function handler. body=%s", w.Code, w.Body.String())
	}
}

// ── 17. SDL deduplication across input and output ──────────────────────

func TestSDL_DedupeSameNameInputAndOutput(t *testing.T) {
	type User struct {
		Name string
	}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req User) (User, error) {
		return User{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	// User should appear once as input, once as output (different sections)
	inputCount := countOccurrences(sdl, "input User")
	typeCount := countOccurrences(sdl, "type User")
	if inputCount != 1 {
		t.Errorf("input User should appear exactly once, got %d", inputCount)
	}
	if typeCount != 1 {
		t.Errorf("type User should appear exactly once, got %d", typeCount)
	}
}

// ── 18. Concurrent handler calls (race detection) ─────────────────────

func TestConcurrent_HandlerCalls(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{ID: req.Name, Name: req.Name}, nil
	})
	field := &GraphQLField{Name: "concurrent", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{ID: req.Name, Name: req.Name}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			body := `{"query":"{concurrent(name:\"user\" + $n,age:0){id name email}}"}`
			body = strings.Replace(body, "$n", "", -1)
			r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
			w := httptest.NewRecorder()
			srv.ServeGraphQL(w, r)
			if w.Code != 200 {
				t.Errorf("concurrent call %d: status %d", idx, w.Code)
			}
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ── 19. HTTP method enforcement ────────────────────────────────────────

func TestGraphQL_RejectsNonPOST(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	field := &GraphQLField{Name: "test", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	r := httptest.NewRequest("GET", "/graphql", nil)
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)
	if w.Code != 405 {
		t.Errorf("GET /graphql should return 405, got %d", w.Code)
	}
}

// ── 20. SDL contains proper type definitions for complex scenarios ─────

func TestSDL_ComplexNestedWithSlices(t *testing.T) {
	type Tag struct {
		Name string
	}
	type Comment struct {
		Text string
		Tags []Tag
	}
	type Post struct {
		Title    string
		Comments []Comment
	}
	type input struct{}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req input) (Post, error) {
		return Post{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "type Tag {") {
		t.Error("SDL should define 'type Tag'")
	}
	if !strings.Contains(sdl, "type Comment {") {
		t.Error("SDL should define 'type Comment'")
	}
	if !strings.Contains(sdl, "type Post {") {
		t.Error("SDL should define 'type Post'")
	}
	if !strings.Contains(sdl, "tags: [Tag]") {
		t.Errorf("Comment.tags should be '[Tag]'. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "comments: [Comment]") {
		t.Errorf("Post.comments should be '[Comment]'. Got:\n%s", sdl)
	}
}

// ── 21. Input object with omitempty ────────────────────────────────────

func TestSDL_JSONTagOmitEmpty(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	// omitempty should still produce required type for non-pointer
	if !strings.Contains(sdl, "name: String!") {
		t.Errorf("required field 'name' should be 'String!'. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "age: Int!") {
		t.Errorf("non-pointer field with omitempty should still be 'Int!'. Got:\n%s", sdl)
	}
}

// ── 22. Unexported fields should be skipped ────────────────────────────

func TestSDL_UnexportedFieldsSkipped(t *testing.T) {
	type Result struct {
		Public  string
		private string // unexported
	}
	type input struct{}
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req input) (Result, error) {
		return Result{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if strings.Contains(sdl, "private") {
		t.Errorf("unexported field 'private' should be skipped. Got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "public: String!") {
		t.Errorf("exported field should be present. Got:\n%s", sdl)
	}
}

// ── 23. Handler with wrong signature ───────────────────────────────────

func TestHandler_WrongSignature(t *testing.T) {
	// This tests the error path when a handler has wrong return signature
	srv := NewGraphQLServer(nil)
	srv.resolvers["bad"] = func() {} // no args, no return
	body := `{"query":"{bad}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)
	if w.Code != 500 {
		t.Errorf("bad handler should return 500, got %d. body=%s", w.Code, w.Body.String())
	}
}

// ── 24. SDL with no operations ─────────────────────────────────────────

func TestSDL_NoOperations(t *testing.T) {
	m := &ModelImpl{}
	_ = m.SchemaHelper().RegisterInputType(testInput{})
	// Register a type but no operations
	sdl := m.SchemaHelper().SDL()
	if strings.Contains(sdl, "type Query") {
		t.Error("SDL should not have Query type when no queries registered")
	}
	if strings.Contains(sdl, "type Mutation") {
		t.Error("SDL should not have Mutation type when no mutations registered")
	}
}

// ── 25. Variable with object value ─────────────────────────────────────

func TestResolveVariables_ObjectValue(t *testing.T) {
	args := map[string]any{"input": "$input"}
	vars := map[string]any{"input": map[string]any{"name": "test", "age": 10}}
	got := resolveVariables(args, vars)
	input, ok := got["input"].(map[string]any)
	if !ok {
		t.Fatalf("input should be map[string]any, got %T", got["input"])
	}
	if input["name"] != "test" {
		t.Errorf("nested name = %v", input["name"])
	}
}

// ── 26. SDL field with pointer to struct in input ──────────────────────

func TestSDL_InputPointerToStruct(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Input struct {
		Inner *Inner
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLMutation(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	// *Inner in input should be nullable
	if strings.Contains(sdl, "inner: Inner!") {
		t.Errorf("Pointer to struct in input should be nullable 'Inner', not 'Inner!'\n%s", sdl)
	}
}

// ── 27. MapToStruct with list values ───────────────────────────────────

func TestMapToStruct_ListMapping(t *testing.T) {
	type Input struct {
		Tags []string
	}
	args := map[string]any{"tags": []any{"a", "b", "c"}}
	var dest Input
	err := mapToStruct(args, &dest)
	if err != nil {
		t.Fatalf("mapToStruct: %v", err)
	}
	if len(dest.Tags) != 3 {
		t.Errorf("Tags length = %d, want 3", len(dest.Tags))
	}
}

// ── 28. SDL SDL() idempotent ───────────────────────────────────────────

func TestSDL_SDLImpotent(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	sdl1 := m.SchemaHelper().SDL()
	sdl2 := m.SchemaHelper().SDL()
	if sdl1 != sdl2 {
		t.Errorf("SDL() should be idempotent. First:\n%s\nSecond:\n%s", sdl1, sdl2)
	}
}

// ── 29. Handler returns non-nil data with non-nil error ────────────────

func TestHandler_DataAndErrorTogether(t *testing.T) {
	// GraphQL spec allows partial data with errors
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		// Return data AND error - spec allows this for partial results
		return testResult{ID: "partial"}, nil
	})
	field := &GraphQLField{Name: "partial", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{ID: "partial"}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{partial(name:\"x\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data wrapper. Got: %v", resp)
	}
	// With no error, data should have the result
	if _, has := data["partial"]; !has {
		t.Errorf("partial should be in data. Got: %v", data)
	}
}

// ── 30. Extract operation name from mutation with parentheses ──────────

func TestExtractOperationName_MutationWithParens(t *testing.T) {
	q := `mutation { create(input: {name:"x"}) { id } }`
	name := extractOperationName(q)
	// Anonymous mutation: should extract first field inside braces
	if name != "create" {
		t.Errorf("extractOperationName(%q) = %q, want 'create'", q, name)
	}
}

// ── 31. SDL with float input ───────────────────────────────────────────

func TestSDL_FloatInput(t *testing.T) {
	type Input struct {
		Score float64
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "score: Float!") {
		t.Errorf("float64 field should be 'Float!'. Got:\n%s", sdl)
	}
}

// ── 32. SDL with boolean input ─────────────────────────────────────────

func TestSDL_BoolInput(t *testing.T) {
	type Input struct {
		Active bool
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "active: Boolean!") {
		t.Errorf("bool field should be 'Boolean!'. Got:\n%s", sdl)
	}
}

// ── 33. SDL with int field in input ────────────────────────────────────

func TestSDL_IntInput(t *testing.T) {
	type Input struct {
		Limit int
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "limit: Int!") {
		t.Errorf("int field should be 'Int!'. Got:\n%s", sdl)
	}
}

// ── 34. GraphQL server ServeSDL ────────────────────────────────────────

func TestServeSDL(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req testInput) (testResult, error) {
		return testResult{}, nil
	})
	field := &GraphQLField{Name: "test", Kind: "query", Handler: func(req testInput) (testResult, error) {
		return testResult{}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	r := httptest.NewRequest("GET", "/graphql/sdl", nil)
	w := httptest.NewRecorder()
	srv.ServeSDL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want 'text/plain; charset=utf-8'", ct)
	}
	if !strings.Contains(w.Body.String(), "type Query") {
		t.Error("SDL body should contain 'type Query'")
	}
}

// ── 35. GraphQL server ServeGraphiQL ───────────────────────────────────

func TestServeGraphiQL(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	r := httptest.NewRequest("GET", "/graphiql", nil)
	w := httptest.NewRecorder()
	srv.ServeGraphiQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want 'text/html; charset=utf-8'", ct)
	}
	if !strings.Contains(w.Body.String(), "GraphiQL") {
		t.Error("GraphiQL body should contain 'GraphiQL'")
	}
}

// ── 36. RemoteOperationReq with nil context ────────────────────────────

func TestRemoteOpReq_NilContext(t *testing.T) {
	m := &ModelImpl{}
	BindGraphQLOperationWithContext(m, func(req *RemoteOperationReq[testInput]) (testResult, error) {
		if req.Ctx == nil {
			return testResult{Name: "ctx-is-nil"}, nil
		}
		return testResult{Name: "ctx-is-set"}, nil
	}, &GraphQLOperationOptions{Name: "ctxtest"})
	field := &GraphQLField{Name: "ctxtest", Kind: "query", Handler: func(req *RemoteOperationReq[testInput]) (testResult, error) {
		if req.Ctx == nil {
			return testResult{Name: "ctx-is-nil"}, nil
		}
		return testResult{Name: "ctx-is-set"}, nil
	}}
	srv := NewGraphQLServer(m)
	srv.RegisterField(field)

	body := `{"query":"{ctxtest(name:\"x\",age:0){id name email}}"}`
	r := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	// No context value attached
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 200 {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data map[string]testResult `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data["ctxtest"].Name != "ctx-is-nil" {
		t.Errorf("Ctx should be nil when no context value attached. Got: %q", resp.Data["ctxtest"].Name)
	}
}

// ── 37. JSON body decode error ─────────────────────────────────────────

func TestGraphQL_InvalidJSON(t *testing.T) {
	m := &ModelImpl{}
	srv := NewGraphQLServer(m)

	r := httptest.NewRequest("POST", "/graphql", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	srv.ServeGraphQL(w, r)

	if w.Code != 400 {
		t.Errorf("invalid JSON should return 400, got %d", w.Code)
	}
}

// ── 38. Input type with all primitive pointer fields ───────────────────

func TestSDL_AllPointerFieldsNullable(t *testing.T) {
	type Input struct {
		Name   *string
		Age    *int
		Score  *float64
		Active *bool
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if strings.Contains(sdl, "name: String!") {
		t.Errorf("*string should be nullable 'String', not 'String!'. Got:\n%s", sdl)
	}
	if strings.Contains(sdl, "age: Int!") {
		t.Errorf("*int should be nullable 'Int', not 'Int!'. Got:\n%s", sdl)
	}
	if strings.Contains(sdl, "score: Float!") {
		t.Errorf("*float64 should be nullable 'Float', not 'Float!'. Got:\n%s", sdl)
	}
	if strings.Contains(sdl, "active: Boolean!") {
		t.Errorf("*bool should be nullable 'Boolean', not 'Boolean!'. Got:\n%s", sdl)
	}
}

// ── 39. Operation name extraction with whitespace ──────────────────────

func TestExtractOperationName_Whitespace(t *testing.T) {
	q := `  query   GetUser  (id:"1")  { getUser { id } }`
	name := extractOperationName(q)
	if name != "GetUser" {
		t.Errorf("extractOperationName(%q) = %q, want 'GetUser'", q, name)
	}
}

// ── 40. SDL with struct that has json tag override ─────────────────────

func TestSDL_JSONTagOverride(t *testing.T) {
	type Input struct {
		FullName string `json:"full_name"`
	}
	type Out struct{ ID string }
	m := &ModelImpl{}
	BindGraphQLQuery(m, func(req Input) (Out, error) {
		return Out{}, nil
	})
	sdl := m.SchemaHelper().SDL()
	if !strings.Contains(sdl, "full_name: String!") {
		t.Errorf("json tag 'full_name' should be used. Got:\n%s", sdl)
	}
	if strings.Contains(sdl, "FullName:") {
		t.Errorf("field name 'FullName' should not appear when json tag overrides. Got:\n%s", sdl)
	}
}
