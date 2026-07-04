package graphql

import (
	"reflect"
	"time"
)

// GoTypeToGraphQL maps a Go type to its GraphQL scalar/type string.
// This is a pure function: no side effects, fully testable.
func GoTypeToGraphQL(v any) string {
	t := reflect.TypeOf(v)
	if t == nil {
		return "String"
	}

	// Handle pointers: unwrap to the element type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle slices
	if t.Kind() == reflect.Slice {
		elem := t.Elem()
		elemType := GoTypeToGraphQL(reflect.New(elem).Interface())
		// Special case: []float32 is treated as Vector
		if elem.Kind() == reflect.Float32 {
			return "Vector"
		}
		return "[" + elemType + "]"
	}

	// Primitive types
	switch t.Kind() {
	case reflect.String:
		return "String"
	case reflect.Int, reflect.Int64:
		return "Int"
	case reflect.Float32, reflect.Float64:
		return "Float"
	case reflect.Bool:
		return "Boolean"
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return "DateTime"
		}
		// Named structs get their name
		if t.Name() != "" {
			return t.Name()
		}
		// Anonymous struct fallback
		return "String"
	default:
		return "String"
	}
}

// GoTypeName returns the GraphQL type name for a Go reflect.Type.
// Used by SchemaHelper for generating input/type definitions.
func GoTypeName(t reflect.Type) string {
	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Anonymous structs
	if t.Name() == "" {
		return ""
	}

	return t.Name()
}

// GoTypeToSDLFragment returns the SDL fragment for a Go type (e.g., "String", "Int!", "[String]", "MyTypeInput").
// Handles nullability: pointers and omitempty fields are nullable (no '!').
func GoTypeToSDLFragment(t reflect.Type, nullable bool) string {
	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		nullable = true
		t = t.Elem()
	}

	base := GoTypeToGraphQL(reflect.New(t).Interface())

	if nullable {
		return base
	}
	return base + "!"
}
