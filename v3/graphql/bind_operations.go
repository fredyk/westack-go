package graphql

import (
	"context"
	"reflect"
	"runtime"
	"strings"
)

// GraphQLField represents a registered GraphQL field.
type GraphQLField struct {
	Name    string
	Kind    string // "query" or "mutation"
	Schema  *SchemaHelper
	Handler any
}

// RemoteOperationReq wraps a typed input with its per-request context.
// Handlers bound via BindGraphQLOperationWithContext receive this so that
// request-scoped values (auth principal, deadline, cancellation) flow into
// the resolver. Ctx is the HTTP request context; Input is the decoded args.
type RemoteOperationReq[T any] struct {
	Ctx   context.Context
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

// BindGraphQLOperation registers a GraphQL operation with auto-SDL.
// T = input type, R = result type.
func BindGraphQLOperation[T any, R any](m Model, handler func(req T) (R, error)) *GraphQLField {
	return bindGraphQL[T, R](m, "query", "", handler)
}

// BindGraphQLOperationWithOptions applies options before binding.
func BindGraphQLOperationWithOptions[T any, R any](m Model, handler func(req T) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQL[T, R](m, "query", nameOr(o.Name, handler), handler)
}

// BindGraphQLQueryWithOptions registers a Query with an explicit name (o.Name),
// falling back to the handler's function name when empty.
func BindGraphQLQueryWithOptions[T any, R any](m Model, handler func(req T) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQL[T, R](m, "query", nameOr(o.Name, handler), handler)
}

// BindGraphQLMutationWithOptions registers a Mutation with an explicit name
// (o.Name), falling back to the handler's function name when empty. It is the
// mutation twin of BindGraphQLOperationWithOptions, enabling apps to register
// named side-effect operations without reimplementing the GraphQL engine.
func BindGraphQLMutationWithOptions[T any, R any](m Model, handler func(req T) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQL[T, R](m, "mutation", nameOr(o.Name, handler), handler)
}

// nameOr returns explicit if non-empty, else the handler's derived function name.
func nameOr(explicit string, handler any) string {
	if explicit != "" {
		return explicit
	}
	return getFunctionName(handler)
}

// BindGraphQLOperationWithContext is the full-power variant accepting *RemoteOperationReq.
func BindGraphQLOperationWithContext[T any, R any](m Model, handler func(req *RemoteOperationReq[T]) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQLCtx[T, R](m, "query", o.Name, handler, o)
}

// BindGraphQLMutationWithContext is the mutation twin of BindGraphQLOperationWithContext.
// Handlers receive *RemoteOperationReq[T] with the per-request context so that
// auth principal, deadlines and cancellation flow into mutation resolvers.
func BindGraphQLMutationWithContext[T any, R any](m Model, handler func(req *RemoteOperationReq[T]) (R, error), o *GraphQLOperationOptions) *GraphQLField {
	if o == nil {
		o = &GraphQLOperationOptions{}
	}
	return bindGraphQLCtx[T, R](m, "mutation", o.Name, handler, o)
}

// BindGraphQLQuery registers a read-only operation as a GraphQL Query.
func BindGraphQLQuery[T any, R any](m Model, handler func(req T) (R, error)) *GraphQLField {
	return bindGraphQL[T, R](m, "query", getFunctionName(handler), handler)
}

// BindGraphQLMutation registers a side-effect operation as a GraphQL Mutation.
func BindGraphQLMutation[T any, R any](m Model, handler func(req T) (R, error)) *GraphQLField {
	return bindGraphQL[T, R](m, "mutation", getFunctionName(handler), handler)
}

func bindGraphQL[T any, R any](m Model, kind, customName string, handler any) *GraphQLField {
	if m == nil {
		return &GraphQLField{Name: customName, Kind: kind}
	}

	schema := m.SchemaHelper()

	inputName := schema.RegisterInputType(new(T))
	resultName := schema.RegisterOutputType(new(R))

	name := customName
	if name == "" {
		name = "anonymous"
	}

	schema.AddOperation(kind, name, inputName, resultName, "")

	return &GraphQLField{
		Name:    name,
		Kind:    kind,
		Schema:  schema,
		Handler: handler,
	}
}

func bindGraphQLCtx[T any, R any](m Model, kind, name string, handler any, o *GraphQLOperationOptions) *GraphQLField {
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
		Name:    name,
		Kind:    kind,
		Schema:  schema,
		Handler: handler,
	}
}

func getFunctionName(fn any) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}
