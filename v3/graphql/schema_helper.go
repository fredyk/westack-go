package graphql

import (
	"fmt"
	"reflect"
	"strings"
)

// SchemaHelper accumulates GraphQL type definitions and operation fields,
// producing the full SDL string via SDL().
type SchemaHelper struct {
	types      map[string]string // name → SDL fragment for types/inputs
	operations []operationDef
}

type operationDef struct {
	kind       string // "query" or "mutation"
	name       string
	inputType  string
	resultType string
	desc       string
}

// NewSchemaHelper creates a new empty SchemaHelper.
func NewSchemaHelper() *SchemaHelper {
	return &SchemaHelper{
		types:      make(map[string]string),
		operations: make([]operationDef, 0),
	}
}

// RegisterInputType registers a Go type as a GraphQL input type and returns its name.
// Deduplicates by name.
func (h *SchemaHelper) RegisterInputType(v any) string {
	return h.registerType(v, "input")
}

// RegisterOutputType registers a Go type as a GraphQL output type and returns its name.
// Deduplicates by name.
func (h *SchemaHelper) RegisterOutputType(v any) string {
	return h.registerType(v, "type")
}

// RegisterGenericType is an alias for RegisterInputType (for API compatibility with spec).
func (h *SchemaHelper) RegisterGenericType(v any) string {
	return h.registerType(v, "input")
}

func (h *SchemaHelper) registerType(v any, kind string) string {
	t := reflect.TypeOf(v)
	if t == nil {
		return "String"
	}

	// Unwrap pointer
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := GoTypeName(t)
	if name == "" {
		// Anonymous type
		return "String"
	}

	// Dedupe
	if _, ok := h.types[name]; ok {
		return name
	}

	// Generate SDL fragment
	frag := generateTypeSDL(name, t, kind)
	h.types[name] = frag

	return name
}

// AddOperation registers a GraphQL operation (query or mutation).
func (h *SchemaHelper) AddOperation(kind, name, inputType, resultType, desc string) {
	h.operations = append(h.operations, operationDef{
		kind:       strings.ToLower(kind),
		name:       name,
		inputType:  inputType,
		resultType: resultType,
		desc:       desc,
	})
}

// SDL returns the accumulated SDL string.
func (h *SchemaHelper) SDL() string {
	var sb strings.Builder

	// Output all type definitions
	for _, frag := range h.types {
		sb.WriteString(frag)
		sb.WriteString("\n\n")
	}

	// Build Query type if there are queries
	var queries []operationDef
	var mutations []operationDef
	for _, op := range h.operations {
		if op.kind == "query" {
			queries = append(queries, op)
		} else {
			mutations = append(mutations, op)
		}
	}

	if len(queries) > 0 {
		sb.WriteString("type Query {\n")
		for _, op := range queries {
			sb.WriteString(fmt.Sprintf("  %s(%s: %s): %s\n", op.name, op.name, op.inputType, op.resultType))
		}
		sb.WriteString("}\n\n")
	}

	if len(mutations) > 0 {
		sb.WriteString("type Mutation {\n")
		for _, op := range mutations {
			sb.WriteString(fmt.Sprintf("  %s(%s: %s): %s\n", op.name, op.name, op.inputType, op.resultType))
		}
		sb.WriteString("}\n\n")
	}

	return strings.TrimSpace(sb.String())
}

// generateTypeSDL creates the SDL fragment for a Go struct.
func generateTypeSDL(name string, t reflect.Type, kind string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s %s {\n", kind, name))

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			// Unexported field
			continue
		}

		tag := getJSONTag(field)
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = field.Name
		}

		// Determine if nullable (pointer)
		nullable := field.Type.Kind() == reflect.Ptr
		fieldType := field.Type
		if nullable {
			fieldType = fieldType.Elem()
		}

		sdlType := GoTypeToSDLFragment(fieldType, nullable)
		sb.WriteString(fmt.Sprintf("  %s: %s\n", tag, sdlType))
	}

	sb.WriteString("}")
	return sb.String()
}

func getJSONTag(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, ",")
	return parts[0]
}
