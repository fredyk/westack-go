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


