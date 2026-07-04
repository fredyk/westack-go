package graphql

import (
	"reflect"
	"runtime"
)

// GraphQLField represents a registered GraphQL field.
type GraphQLField struct {
	Name   string
	Kind   string // "query" or "mutation"
	Schema *SchemaHelper
}

// RemoteOperationReq wraps a typed input with its context.
type RemoteOperationReq[T any] struct {
	Input T
}

// GraphQLOperationOptions holds configuration for a GraphQL operation binding.
type GraphQLOperationOptions struct {
	Name        string
	Description string
}

// GetFunctionName extracts the function name from a function value using reflection.
func GetFunctionName(fn any) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	parts := splitPath(name)
	return parts[len(parts)-1]
}

func splitPath(s string) []string {
	var parts []string
	for i := 0; i < len(s); i++ {
		if s[i] == '/' || s[i] == '.' {
			continue
		}
		j := i
		for j < len(s) && s[j] != '/' && s[j] != '.' {
			j++
		}
		parts = append(parts, s[i:j])
		i = j
	}
	return parts
}

// BindGraphQLOperation is a stub gemelo de BindRemoteOperation.
// T = input type, R = result type. Registers a GraphQL operation with auto-SDL.
func BindGraphQLOperation[T any, R any](m *localPlaceholder, handler func(req T) (R, error)) *GraphQLField {
	return bindGraphQL[T, R](m, "query", "", handler)
}

// BindGraphQLOperationWithOptions applies options before binding.
func BindGraphQLOperationWithOptions[T any, R any](m *localPlaceholder, handler func(req T) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQL[T, R](m, "query", o.Name, handler)
}

// BindGraphQLOperationWithContext is the full-power variant accepting *RemoteOperationReq.
func BindGraphQLOperationWithContext[T any, R any](m *localPlaceholder, handler func(req *RemoteOperationReq[T]) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQLCtx[T, R](m, "query", o.Name, handler, o)
}

// BindGraphQLQuery registers a read-only operation as a GraphQL Query.
func BindGraphQLQuery[T any, R any](m *localPlaceholder, handler func(req T) (R, error)) *GraphQLField {
	return func() *GraphQLField {
		return bindGraphQL[T, R](m, "query", "", handler)
	}()
}

// BindGraphQLMutation registers a side-effect operation as a GraphQL Mutation.
func BindGraphQLMutation[T any, R any](m *localPlaceholder, handler func(req T) (R, error)) *GraphQLField {
	return func() *GraphQLField {
		return bindGraphQL[T, R](m, "mutation", "", handler)
	}()
}

func bindGraphQL[T any, R any](m *localPlaceholder, kind, customName string, handler any) *GraphQLField {
	if m == nil {
		return &GraphQLField{Name: customName, Kind: kind}
	}

	schema := m.SchemaHelper()

	// Register input type T
	inputName := schema.RegisterInputType(new(T))

	// Register result type R
	resultName := schema.RegisterOutputType(new(R))

	name := customName
	if name == "" {
		name = "anonymous"
	}

	schema.AddOperation(kind, name, inputName, resultName, "")

	return &GraphQLField{
		Name:   name,
		Kind:   kind,
		Schema: schema,
	}
}

func bindGraphQLCtx[T any, R any](m *localPlaceholder, kind, name string, _ any, o *GraphQLOperationOptions) *GraphQLField {
	if m == nil {
		return &GraphQLField{Name: name, Kind: kind}
	}

	schema := m.SchemaHelper()

	inputName := schema.RegisterInputType(new(T))
	resultName := schema.RegisterOutputType(new(R))

	if o != nil && o.Name != "" {
		name = o.Name
	}

	schema.AddOperation(kind, name, inputName, resultName, "")

	return &GraphQLField{
		Name:   name,
		Kind:   kind,
		Schema: schema,
	}
}
