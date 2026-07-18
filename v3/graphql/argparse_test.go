package graphql

import (
	"reflect"
	"testing"
)

func TestParseFieldArgs(t *testing.T) {
	cases := []struct {
		name  string
		query string
		op    string
		want  map[string]any
	}{
		{"string arg", `{expediente(id:"e1"){id estado}}`, "expediente", map[string]any{"id": "e1"}},
		{"int arg", `{expedientes(limit:10){id}}`, "expedientes", map[string]any{"limit": int64(10)}},
		{"string with plus", `{clientePorWhatsapp(numero:"+34600111222"){id}}`, "clientePorWhatsapp", map[string]any{"numero": "+34600111222"}},
		{"scalar + enum", `mutation{registrarDocumento(expedienteId:"e1",tipo:PASAPORTE){id}}`, "registrarDocumento", map[string]any{"expedienteId": "e1", "tipo": "PASAPORTE"}},
		{"bool", `{x(flag:true){y}}`, "x", map[string]any{"flag": true}},
		{"list", `{buscar(embedding:[1,2],k:5){id}}`, "buscar", map[string]any{"embedding": []any{int64(1), int64(2)}, "k": int64(5)}},
		{"nested object", `mutation{crear(input:{clientId:"c1",tramiteId:"t1"}){id}}`, "crear", map[string]any{"input": map[string]any{"clientId": "c1", "tramiteId": "t1"}}},
		{"no args", `{expediente{id}}`, "expediente", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseFieldArgs(c.query, c.op)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("parseFieldArgs(%q) = %#v, want %#v", c.query, got, c.want)
			}
		})
	}
}

func TestResolveVariables(t *testing.T) {
	args := map[string]any{"id": "$oid"}
	vars := map[string]any{"oid": "e42", "extra": 7}
	got := resolveVariables(args, vars)
	if got["id"] != "e42" {
		t.Errorf("variable $oid no resuelta: %v", got["id"])
	}
	if got["extra"] != 7 {
		t.Errorf("variable declarada no inyectada: %v", got["extra"])
	}
}

func TestParseFieldArgs_EdgeCases(t *testing.T) {
	// empty opName → nil (no infinite loop)
	got := parseFieldArgs(`{foo(id:"1"){id}}`, "")
	if got != nil {
		t.Errorf("empty opName should return nil, got %v", got)
	}

	// no matching operation
	got = parseFieldArgs(`{foo(id:"1"){id}}`, "bar")
	if got != nil {
		t.Errorf("no matching opName should return nil, got %v", got)
	}

	// name embedded in larger identifier: "foo" inside "foobar"
	got = parseFieldArgs(`{foobar(id:"1"){id}}`, "foo")
	if got != nil {
		t.Errorf("embedded name should not match, got %v", got)
	}

	// name embedded at end: "bar" inside "foobar"
	got = parseFieldArgs(`{foobar(id:"1"){id}}`, "bar")
	if got != nil {
		t.Errorf("embedded name at end should not match, got %v", got)
	}

	// operation with spaces in args
	got = parseFieldArgs(`{foo( id : "1" , name : "test" ) {id}}`, "foo")
	if got == nil {
		t.Fatal("expected args with spaces, got nil")
	}
	if got["id"] != "1" {
		t.Errorf("id = %v, want %v", got["id"], "1")
	}
	if got["name"] != "test" {
		t.Errorf("name = %v, want %v", got["name"], "test")
	}
}

func TestParseString_EscapeSequences(t *testing.T) {
	// \n
	p := &argParser{s: `"hello\nworld"`, pos: 0}
	got := p.parseString()
	if got != "hello\nworld" {
		t.Errorf("escape \\n: got %q, want %q", got, "hello\nworld")
	}

	// \t
	p = &argParser{s: `"a\tb"`, pos: 0}
	got = p.parseString()
	if got != "a\tb" {
		t.Errorf("escape \\t: got %q, want %q", got, "a\tb")
	}

	// \"
	p = &argParser{s: `"say \"hi\""`, pos: 0}
	got = p.parseString()
	if got != `say "hi"` {
		t.Errorf("escape \\\" : got %q, want %q", got, `say "hi"`)
	}

	// \\
	p = &argParser{s: `"path\\to\\file"`, pos: 0}
	got = p.parseString()
	if got != `path\to\file` {
		t.Errorf("escape \\\\: got %q, want %q", got, `path\to\file`)
	}

	// \/
	p = &argParser{s: `"url\/path"`, pos: 0}
	got = p.parseString()
	if got != `url/path` {
		t.Errorf("escape \\/: got %q, want %q", got, `url/path`)
	}

	// unknown escape (should pass through the escape char itself)
	p = &argParser{s: `"a\x"`, pos: 0}
	got = p.parseString()
	if got != "ax" {
		t.Errorf("unknown escape \\x: got %q, want %q", got, "ax")
	}
}

func TestParseValue_NonVariableStrings(t *testing.T) {
	// Non-$ strings should be returned as-is
	p := &argParser{s: `"hello"`, pos: 0}
	got := p.parseValue()
	if got != "hello" {
		t.Errorf("plain string: got %v, want %q", got, "hello")
	}

	// Variable reference
	p = &argParser{s: `$myVar`, pos: 0}
	got = p.parseValue()
	if got != "$myVar" {
		t.Errorf("variable ref: got %v, want %q", got, "$myVar")
	}
}

func TestParseValue_MapAndSlice(t *testing.T) {
	// Map
	p := &argParser{s: `{key:"val"}`, pos: 0}
	got := p.parseValue()
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("map key = %v, want %v", m["key"], "val")
	}

	// Slice
	p = &argParser{s: `[1,"two",true]`, pos: 0}
	got = p.parseValue()
	sl, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}
	if len(sl) != 3 {
		t.Fatalf("slice len = %d, want 3", len(sl))
	}
}

func TestParseScalarOrEnum(t *testing.T) {
	p := &argParser{s: `true`, pos: 0}
	got := p.parseScalarOrEnum()
	if got != true {
		t.Errorf("true: got %v, want true", got)
	}

	p = &argParser{s: `false`, pos: 0}
	got = p.parseScalarOrEnum()
	if got != false {
		t.Errorf("false: got %v, want false", got)
	}

	p = &argParser{s: `null`, pos: 0}
	got = p.parseScalarOrEnum()
	if got != nil {
		t.Errorf("null: got %v, want nil", got)
	}

	p = &argParser{s: `MY_ENUM_VALUE`, pos: 0}
	got = p.parseScalarOrEnum()
	if got != "MY_ENUM_VALUE" {
		t.Errorf("enum: got %v, want %q", got, "MY_ENUM_VALUE")
	}

	p = &argParser{s: `3.14`, pos: 0}
	got = p.parseScalarOrEnum()
	if f, ok := got.(float64); !ok || f != 3.14 {
		t.Errorf("float: got %v, want 3.14", got)
	}
}

func TestArgParser_EmptyList(t *testing.T) {
	p := &argParser{s: `)`, pos: 0}
	got := p.parseArgList()
	if got != nil {
		t.Errorf("empty arg list should return nil, got %v", got)
	}
}

func TestArgParser_EmptyObject(t *testing.T) {
	p := &argParser{s: `}`, pos: 0}
	got := p.parseObject()
	if got == nil {
		t.Fatal("empty object should return empty map, not nil")
	}
	if len(got) != 0 {
		t.Errorf("empty object: got %v, want empty map", got)
	}
}

func TestArgParser_EmptyListValue(t *testing.T) {
	// parseList: empty [...] returns nil (var list []any never appended to)
	// This is the actual behavior of the source code.
	p := &argParser{s: `[]`, pos: 0}
	got := p.parseList()
	if got != nil {
		t.Errorf("empty list: got %v, want nil (var list []any not appended to)", got)
	}
}

func TestArgParser_NonEmptyList(t *testing.T) {
	// Non-empty list should work correctly
	p := &argParser{s: `[1,2,3]`, pos: 0}
	got := p.parseList()
	if got == nil {
		t.Fatal("non-empty list should not be nil")
	}
	if len(got) != 3 {
		t.Errorf("list len = %d, want 3", len(got))
	}
}

func TestArgParser_EndOfInput(t *testing.T) {
	p := &argParser{s: ``, pos: 0}
	got := p.parseValue()
	if got != nil {
		t.Errorf("end of input: got %v, want nil", got)
	}
}

func TestResolveValue_NonVariableStrings(t *testing.T) {
	vars := map[string]any{}

	// Regular string (no $) should pass through unchanged
	got := resolveValue("hello", vars)
	if got != "hello" {
		t.Errorf("non-variable string: got %v, want %q", got, "hello")
	}

	// String that looks like a variable but doesn't exist in vars
	got = resolveValue("$missing", vars)
	if got != "$missing" {
		t.Errorf("missing variable: got %v, want %q", got, "$missing")
	}

	// Integer should pass through
	got = resolveValue(42, vars)
	if got != 42 {
		t.Errorf("int: got %v, want 42", got)
	}

	// Float should pass through
	got = resolveValue(3.14, vars)
	if got != 3.14 {
		t.Errorf("float: got %v, want 3.14", got)
	}

	// Bool should pass through
	got = resolveValue(true, vars)
	if got != true {
		t.Errorf("bool: got %v, want true", got)
	}

	// Nil should pass through
	got = resolveValue(nil, vars)
	if got != nil {
		t.Errorf("nil: got %v, want nil", got)
	}
}

func TestResolveValue_Map(t *testing.T) {
	vars := map[string]any{"x": "resolved"}
	input := map[string]any{"a": "$x", "b": "literal"}
	got := resolveValue(input, vars).(map[string]any)
	if got["a"] != "resolved" {
		t.Errorf("map resolved: got %v, want %q", got["a"], "resolved")
	}
	if got["b"] != "literal" {
		t.Errorf("map literal: got %v, want %q", got["b"], "literal")
	}
}

func TestResolveValue_Slice(t *testing.T) {
	vars := map[string]any{"v": "found"}
	input := []any{"$v", "keep", int64(1)}
	got := resolveValue(input, vars).([]any)
	if got[0] != "found" {
		t.Errorf("slice[0] resolved: got %v, want %q", got[0], "found")
	}
	if got[1] != "keep" {
		t.Errorf("slice[1] literal: got %v, want %q", got[1], "keep")
	}
	if got[2] != int64(1) {
		t.Errorf("slice[2] int: got %v, want 1", got[2])
	}
}

func TestResolveVariables_NilArgs(t *testing.T) {
	vars := map[string]any{"k": "v"}
	got := resolveVariables(nil, vars)
	if got == nil {
		t.Fatal("nil args should become empty map, not nil")
	}
	if got["k"] != "v" {
		t.Errorf("nil args vars injected: got %v", got)
	}
}

func TestResolveVariables_NoVariables(t *testing.T) {
	args := map[string]any{"a": "b"}
	got := resolveVariables(args, nil)
	if got["a"] != "b" {
		t.Errorf("no variables: got %v", got)
	}
}

func TestResolveVariables_NoVariablesEmpty(t *testing.T) {
	args := map[string]any{"a": "b"}
	got := resolveVariables(args, map[string]any{})
	if got["a"] != "b" {
		t.Errorf("empty variables: got %v", got)
	}
}

func TestExtractOperationName_VariableDeclarations(t *testing.T) {
	// query with variable declarations, anonymous
	q := `query($id: ID!){foo(id:$id){id}}`
	got := extractOperationName(q)
	if got != "foo" {
		t.Errorf("query $v: got %q, want %q", got, "foo")
	}

	// mutation with variable declarations, named
	q = `mutation CreateFoo($id: ID!){foo(id:$id){id}}`
	got = extractOperationName(q)
	if got != "foo" {
		t.Errorf("mutation $v named: got %q, want %q", got, "foo")
	}

	// subscription with variable declarations
	q = `subscription OnFoo($id: ID!){foo(id:$id){id}}`
	got = extractOperationName(q)
	if got != "foo" {
		t.Errorf("subscription $v: got %q, want %q", got, "foo")
	}

	// nested parentheses in variable declaration
	q = `query($input: Input!){foo(input:$input){id}}`
	got = extractOperationName(q)
	if got != "foo" {
		t.Errorf("nested parens: got %q, want %q", got, "foo")
	}

	// no keyword, just field
	q = `{foo(id:"1"){id}}`
	got = extractOperationName(q)
	if got != "foo" {
		t.Errorf("no keyword: got %q, want %q", got, "foo")
	}

	// mutation keyword
	q = `mutation{foo(id:"1"){id}}`
	got = extractOperationName(q)
	if got != "foo" {
		t.Errorf("mutation keyword: got %q, want %q", got, "foo")
	}

	// subscription keyword
	q = `subscription{foo(id:"1"){id}}`
	got = extractOperationName(q)
	if got != "foo" {
		t.Errorf("subscription keyword: got %q, want %q", got, "foo")
	}
}

func TestExtractOperationName_Empty(t *testing.T) {
	got := extractOperationName("")
	if got != "" {
		t.Errorf("empty query: got %q, want %q", got, "")
	}

	got = extractOperationName("   ")
	if got != "" {
		t.Errorf("whitespace: got %q, want %q", got, "")
	}

	// "not valid graphql" has no '{', so it extracts the identifier after skipping "not"
	got = extractOperationName("not valid graphql")
	if got != "valid" {
		t.Errorf("invalid query without braces: got %q, want %q", got, "valid")
	}
}
