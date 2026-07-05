package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// contextType is context.Context's reflect.Type, used to detect RemoteOperationReq.
var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

// GraphQLServer provides HTTP handlers for GraphQL endpoints.
type GraphQLServer struct {
	model     Model
	resolvers map[string]any
}

// NewGraphQLServer creates a new GraphQLServer backed by a Model.
func NewGraphQLServer(m Model) *GraphQLServer {
	return &GraphQLServer{
		model:     m,
		resolvers: make(map[string]any),
	}
}

// RegisterField registers a GraphQLField and its handler with the server.
func (s *GraphQLServer) RegisterField(field *GraphQLField) {
	if field == nil || field.Handler == nil {
		return
	}
	s.resolvers[field.Name] = field.Handler
}

// RegisterFields registers multiple fields at once.
func (s *GraphQLServer) RegisterFields(fields ...*GraphQLField) {
	for _, f := range fields {
		s.RegisterField(f)
	}
}

// ServeSDL handles GET /graphql/sdl — returns the SDL as text/plain.
func (s *GraphQLServer) ServeSDL(w http.ResponseWriter, r *http.Request) {
	sdl := s.model.SchemaHelper().SDL()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sdl))
}

// ServeGraphiQL handles GET /graphiql — returns the GraphiQL IDE HTML.
func (s *GraphQLServer) ServeGraphiQL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(graphiqlHTML))
}

// ServeGraphQL handles POST /graphql — executes a GraphQL query.
func (s *GraphQLServer) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query         string         `json:"query"`
		OperationName string         `json:"operationName"`
		Variables     map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGraphQLError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	operationName, args := parseGraphQLQuery(req.Query, req.OperationName)
	args = resolveVariables(args, req.Variables)
	if operationName == "" {
		writeGraphQLError(w, http.StatusBadRequest, "could not parse operation name from query")
		return
	}

	// El SDL auto-generado declara `op(op: InputType)`: parseFieldArgs deja los
	// campos del input anidados bajo la clave del argumento (== nombre de la
	// operación). Desenvolvemos ese input-object para que sus campos mapeen al
	// struct del resolver. Si los args ya vienen planos (`op(campo: val)`), la
	// clave no existe y no se toca nada (retrocompatible).
	if inner, ok := args[operationName].(map[string]any); ok {
		args = inner
	}

	handler, ok := s.resolvers[operationName]
	if !ok {
		writeGraphQLError(w, http.StatusNotFound, fmt.Sprintf("unknown operation: %s", operationName))
		return
	}

	result, err := s.callResolver(r.Context(), handler, args)
	if err != nil {
		writeGraphQLError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeGraphQLResult(w, operationName, result)
}

func (s *GraphQLServer) callResolver(ctx context.Context, handler any, args map[string]any) (any, error) {
	// Reflect-based call. The handler is either:
	//   func(req T) (R, error)                      — simple, no context
	//   func(req *RemoteOperationReq[T]) (R, error) — carries per-request ctx
	rv := reflect.ValueOf(handler)
	if rv.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler is not a function")
	}

	rType := rv.Type()
	if rType.NumIn() != 1 {
		return nil, fmt.Errorf("handler must accept exactly one argument")
	}

	inputType := rType.In(0)

	var callArg reflect.Value
	if isRemoteOpReq(inputType) {
		// *RemoteOperationReq[T]: inyecta el context de la request y mapea los
		// argumentos GraphQL al campo Input (no al wrapper).
		reqPtr := reflect.New(inputType.Elem())
		elem := reqPtr.Elem()
		if ctx != nil {
			elem.FieldByName("Ctx").Set(reflect.ValueOf(ctx))
		}
		inputField := elem.FieldByName("Input")
		if err := mapToStruct(args, inputField.Addr().Interface()); err != nil {
			return nil, fmt.Errorf("cannot map args to input: %w", err)
		}
		callArg = reqPtr
	} else {
		// func(T): retrocompatible, sin context.
		inputValue := reflect.New(inputType)
		if err := mapToStruct(args, inputValue.Interface()); err != nil {
			return nil, fmt.Errorf("cannot map args to input: %w", err)
		}
		callArg = inputValue.Elem()
	}

	results := rv.Call([]reflect.Value{callArg})
	if len(results) != 2 {
		return nil, fmt.Errorf("handler must return (R, error)")
	}

	if !results[1].IsNil() {
		err, _ := results[1].Interface().(error)
		return nil, err
	}

	return results[0].Interface(), nil
}

// isRemoteOpReq reports whether t is *RemoteOperationReq[T] — a pointer to a
// struct carrying both a Ctx context.Context and an Input field. Lets callResolver
// pick the context-injection path without matching the generic instantiation.
func isRemoteOpReq(t reflect.Type) bool {
	if t.Kind() != reflect.Ptr {
		return false
	}
	e := t.Elem()
	if e.Kind() != reflect.Struct {
		return false
	}
	ctxField, ok := e.FieldByName("Ctx")
	if !ok || ctxField.Type != contextType {
		return false
	}
	_, ok = e.FieldByName("Input")
	return ok
}

func mapToStruct(args map[string]any, dest any) error {
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func parseGraphQLQuery(query string, operationName string) (string, map[string]any) {
	name := operationName
	if name == "" {
		// Try to guess the operation name from the query: { fieldName(...) }
		name = extractOperationName(query)
	}
	// Parsear los argumentos inline de la operación (id, limit, input:{...}, …).
	args := parseFieldArgs(query, name)
	return name, args
}

// resolveVariables sustituye los valores de args que sean referencias "$var"
// por su valor en el mapa de variables del cuerpo de la petición.
func resolveVariables(args, variables map[string]any) map[string]any {
	if len(variables) == 0 {
		return args
	}
	if args == nil {
		args = map[string]any{}
	}
	for k, v := range args {
		if s, ok := v.(string); ok && strings.HasPrefix(s, "$") {
			if rv, found := variables[strings.TrimPrefix(s, "$")]; found {
				args[k] = rv
			}
		}
	}
	// Variables declaradas y no referenciadas inline: inyectarlas si no colisionan.
	for k, v := range variables {
		if _, exists := args[k]; !exists {
			args[k] = v
		}
	}
	return args
}

func extractOperationName(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Skip "mutation" or "query" keyword
	idx := 0
	for idx < len(query) && (query[idx] == ' ' || query[idx] == '\t' || query[idx] == '\n') {
		idx++
	}
	rest := query[idx:]

	// Handle "query" or "mutation" keyword
	lower := strings.ToLower(rest)
	if strings.HasPrefix(lower, "query") {
		rest = rest[5:]
		rest = strings.TrimSpace(rest)
	} else if strings.HasPrefix(lower, "mutation") {
		rest = rest[8:]
		rest = strings.TrimSpace(rest)
	}

	// Now we expect either '{' (anonymous) or an identifier
	if len(rest) == 0 {
		return ""
	}

	if rest[0] == '{' {
		// Anonymous query: extract first field name inside braces
		inner := rest[1:]
		inner = strings.TrimSpace(inner)
		end := strings.IndexAny(inner, " ({\n\t\r")
		if end == -1 {
			end = len(inner)
		}
		field := strings.TrimSpace(inner[:end])
		parts := strings.Fields(field)
		if len(parts) > 0 {
			return parts[0]
		}
		return ""
	}

	// Named operation: "operationName(...)"
	end := strings.IndexAny(rest, " (")
	if end == -1 {
		end = len(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func writeGraphQLError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{{"message": message}},
	})
}

// writeGraphQLResult escribe la respuesta GraphQL. Conforme a la spec, el
// resultado va anidado bajo el nombre de la operación: {"data":{"<op>":result}}.
func writeGraphQLResult(w http.ResponseWriter, operationName string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{operationName: data},
	})
}

const graphiqlHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>GraphiQL</title>
  <link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css">
</head>
<body style="margin:0;width:100%;height:100vh;overflow:hidden">
  <div id="graphiql" style="height:100vh">Loading...</div>
  <script src="https://unpkg.com/react/umd/react.production.min.js"></script>
  <script src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"></script>
  <script src="https://unpkg.com/graphiql/graphiql.min.js"></script>
  <script>
    var fetcher = GraphiQL.createFetcher({url: '/graphql'});
    ReactDOM.createRoot(document.getElementById('graphiql')).render(
      React.createElement(GraphiQL, {fetcher: fetcher})
    );
  </script>
</body>
</html>`
