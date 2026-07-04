package graphql

import (
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
