package graphql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

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
	if operationName == "" {
		writeGraphQLError(w, http.StatusBadRequest, "could not parse operation name from query")
		return
	}

	handler, ok := s.resolvers[operationName]
	if !ok {
		writeGraphQLError(w, http.StatusNotFound, fmt.Sprintf("unknown operation: %s", operationName))
		return
	}

	result, err := s.callResolver(handler, args)
	if err != nil {
		writeGraphQLError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeGraphQLResult(w, result)
}

func (s *GraphQLServer) callResolver(handler any, args map[string]any) (any, error) {
	// Wrap the handler in a reflect-based call.
	// The handler is a function of the form func(req T) (R, error).
	// We create a T from args and call the function.
	rv := reflect.ValueOf(handler)
	if rv.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler is not a function")
	}

	rType := rv.Type()
	if rType.NumIn() != 1 {
		return nil, fmt.Errorf("handler must accept exactly one argument")
	}

	inputType := rType.In(0)
	inputValue := reflect.New(inputType)
	if err := mapToStruct(args, inputValue.Interface()); err != nil {
		return nil, fmt.Errorf("cannot map args to input: %w", err)
	}

	results := rv.Call([]reflect.Value{inputValue.Elem()})
	if len(results) != 2 {
		return nil, fmt.Errorf("handler must return (R, error)")
	}

	if !results[1].IsNil() {
		err, _ := results[1].Interface().(error)
		return nil, err
	}

	return results[0].Interface(), nil
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
	return name, nil
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

func writeGraphQLResult(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": data,
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
