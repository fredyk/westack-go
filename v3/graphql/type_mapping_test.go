package graphql

import (
	"reflect"
	"testing"
	"time"
)

func TestGoTypeToGraphQL(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string maps to String",
			input:    "",
			expected: "String",
		},
		{
			name:     "int maps to Int",
			input:    0,
			expected: "Int",
		},
		{
			name:     "int64 maps to Int",
			input:    int64(0),
			expected: "Int",
		},
		{
			name:     "float64 maps to Float",
			input:    0.0,
			expected: "Float",
		},
		{
			name:     "bool maps to Boolean",
			input:    true,
			expected: "Boolean",
		},
		{
			name:     "time.Time maps to DateTime",
			input:    time.Time{},
			expected: "DateTime",
		},
		{
			name:     "[]string maps to [String]",
			input:    []string{},
			expected: "[String]",
		},
		{
			name:     "[]int maps to [Int]",
			input:    []int{},
			expected: "[Int]",
		},
		{
			name:     "*string (pointer) maps to nullable String",
			input:    (*string)(nil),
			expected: "String",
		},
		{
			name:     "[]float32 with vector tag maps to Vector",
			input:    []float32{},
			expected: "Vector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GoTypeToGraphQL(tt.input)
			if result != tt.expected {
				t.Errorf("GoTypeToGraphQL(%T) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGoTypeToGraphQL_NonPointerRequired(t *testing.T) {
	// A non-pointer, non-scalar struct field should return a type name
	type MyStruct struct {
		Name string
	}
	result := GoTypeToGraphQL(MyStruct{})
	if result != "MyStruct" {
		t.Errorf("GoTypeToGraphQL(MyStruct{}) = %q, want %q", result, "MyStruct")
	}
}

func TestGoTypeToGraphQL_NilType(t *testing.T) {
	result := GoTypeToGraphQL(nil)
	if result != "String" {
		t.Errorf("GoTypeToGraphQL(nil) = %q, want %q", result, "String")
	}
}

func TestGoTypeToGraphQL_AnonymousStruct(t *testing.T) {
	result := GoTypeToGraphQL(struct{ Name string }{})
	if result != "String" {
		// Anonymous structs fallback to String
		t.Errorf("GoTypeToGraphQL anonymous struct = %q, want %q", result, "String")
	}
}

func TestGoTypeToGraphQL_Float32(t *testing.T) {
	result := GoTypeToGraphQL(float32(0))
	if result != "Float" {
		t.Errorf("GoTypeToGraphQL(float32) = %q, want %q", result, "Float")
	}
}

func TestGoTypeToGraphQL_Int32(t *testing.T) {
	// int32 falls through to default case (only int/int64 are matched)
	result := GoTypeToGraphQL(int32(0))
	if result != "String" {
		t.Errorf("GoTypeToGraphQL(int32) = %q, want %q", result, "String")
	}
}

func TestGoTypeToGraphQL_Int8(t *testing.T) {
	// int8 falls through to default case (only int/int64 are matched)
	result := GoTypeToGraphQL(int8(0))
	if result != "String" {
		t.Errorf("GoTypeToGraphQL(int8) = %q, want %q", result, "String")
	}
}

func TestGoTypeToGraphQL_Uint64(t *testing.T) {
	result := GoTypeToGraphQL(uint64(0))
	// uint64 falls through to default → String
	if result != "String" {
		t.Errorf("GoTypeToGraphQL(uint64) = %q, want %q", result, "String")
	}
}

func TestGoTypeName(t *testing.T) {
	type MyNamed struct{}
	t2 := reflect.TypeOf(MyNamed{})
	name := GoTypeName(t2)
	if name != "MyNamed" {
		t.Errorf("GoTypeName(MyNamed) = %q, want %q", name, "MyNamed")
	}

	// Anonymous struct
	anon := reflect.TypeOf(struct{}{})
	name = GoTypeName(anon)
	if name != "" {
		t.Errorf("GoTypeName anonymous = %q, want %q", name, "")
	}

	// Pointer to struct
	ptr := reflect.TypeOf(&MyNamed{})
	name = GoTypeName(ptr)
	if name != "MyNamed" {
		t.Errorf("GoTypeName pointer = %q, want %q", name, "MyNamed")
	}
}

func TestGoTypeToSDLFragment_NonNullable(t *testing.T) {
	t2 := reflect.TypeOf("hello")
	result := GoTypeToSDLFragment(t2, false)
	if result != "String!" {
		t.Errorf("non-nullable String = %q, want %q", result, "String!")
	}
}

func TestGoTypeToSDLFragment_Nullable(t *testing.T) {
	t2 := reflect.TypeOf("hello")
	result := GoTypeToSDLFragment(t2, true)
	if result != "String" {
		t.Errorf("nullable String = %q, want %q", result, "String")
	}
}

func TestGoTypeToSDLFragment_PointerType(t *testing.T) {
	// Pointer type should be nullable regardless of the nullable param
	t2 := reflect.TypeOf((*string)(nil))
	result := GoTypeToSDLFragment(t2, false)
	if result != "String" {
		t.Errorf("pointer should be nullable: got %q, want %q", result, "String")
	}
}

func TestGoTypeToSDLFragment_Int(t *testing.T) {
	t2 := reflect.TypeOf(0)
	result := GoTypeToSDLFragment(t2, false)
	if result != "Int!" {
		t.Errorf("non-nullable Int = %q, want %q", result, "Int!")
	}
}

func TestGoTypeToSDLFragment_Bool(t *testing.T) {
	t2 := reflect.TypeOf(true)
	result := GoTypeToSDLFragment(t2, false)
	if result != "Boolean!" {
		t.Errorf("non-nullable Boolean = %q, want %q", result, "Boolean!")
	}
}

func TestGoTypeToSDLFragment_Float(t *testing.T) {
	t2 := reflect.TypeOf(0.0)
	result := GoTypeToSDLFragment(t2, false)
	if result != "Float!" {
		t.Errorf("non-nullable Float = %q, want %q", result, "Float!")
	}
}

func TestGoTypeToSDLFragment_Time(t *testing.T) {
	t2 := reflect.TypeOf(time.Time{})
	result := GoTypeToSDLFragment(t2, false)
	if result != "DateTime!" {
		t.Errorf("non-nullable DateTime = %q, want %q", result, "DateTime!")
	}
}

func TestGoTypeToSDLFragment_PointerToStruct(t *testing.T) {
	type Foo struct{}
	t2 := reflect.TypeOf(&Foo{})
	result := GoTypeToSDLFragment(t2, false)
	if result != "Foo" {
		t.Errorf("pointer to struct should be nullable: got %q, want %q", result, "Foo")
	}
}
