package graphql

import (
	"testing"
)

type testInput struct {
	Name string
	Age  int
}

type testResult struct {
	ID    string
	Name  string
	Email string
}

type nestedInput struct {
	User  testInput
	Tags  []string
	Notes []float32
}

func TestSchemaHelper_RegisterGenericType(t *testing.T) {
	h := NewSchemaHelper()

	// Register an input type
	inputName := h.RegisterGenericType(testInput{})
	if inputName != "testInput" {
		t.Errorf("RegisterGenericType(testInput) = %q, want %q", inputName, "testInput")
	}

	// Register the same type again — should dedupe
	inputName2 := h.RegisterGenericType(testInput{})
	if inputName2 != "testInput" {
		t.Errorf("RegisterGenericType(testInput) second call = %q, want %q", inputName2, "testInput")
	}

	// Register a result type
	resultName := h.RegisterGenericType(testResult{})
	if resultName != "testResult" {
		t.Errorf("RegisterGenericType(testResult) = %q, want %q", resultName, "testResult")
	}

	// Check SDL contains the input definition
	sdl := h.SDL()
	if sdl == "" {
		t.Fatal("SDL is empty")
	}
}

func TestSchemaHelper_AddOperation(t *testing.T) {
	h := NewSchemaHelper()

	h.RegisterInputType(testInput{})
	h.RegisterOutputType(testResult{})
	h.AddOperation("query", "getUser", "testInput", "testResult", "Gets a user by input")

	sdl := h.SDL()
	if sdl == "" {
		t.Fatal("SDL is empty")
	}

	// SDL should contain the input type
	if !contains(sdl, "input testInput") {
		t.Errorf("SDL does not contain 'input testInput':\n%s", sdl)
	}

	// SDL should contain the output type
	if !contains(sdl, "type testResult") {
		t.Errorf("SDL does not contain 'type testResult':\n%s", sdl)
	}

	// SDL should contain the query field
	if !contains(sdl, "getUser") {
		t.Errorf("SDL does not contain 'getUser':\n%s", sdl)
	}
}

func TestSchemaHelper_QueryVsMutation(t *testing.T) {
	h := NewSchemaHelper()

	h.RegisterInputType(testInput{})
	h.RegisterOutputType(testResult{})

	// GET → query
	h.AddOperation("query", "getUser", "testInput", "testResult", "Gets a user")

	// POST → mutation
	h.AddOperation("mutation", "createUser", "testInput", "testResult", "Creates a user")

	sdl := h.SDL()

	// Should have a Query type with fields
	if !contains(sdl, "type Query") {
		t.Errorf("SDL missing 'type Query':\n%s", sdl)
	}

	// Should have a Mutation type with fields
	if !contains(sdl, "type Mutation") {
		t.Errorf("SDL missing 'type Mutation':\n%s", sdl)
	}
}

func TestSchemaHelper_DedupeNestedTypes(t *testing.T) {
	h := NewSchemaHelper()

	// Register nestedInput which contains testInput as a field
	h.RegisterGenericType(nestedInput{})

	// Register testInput explicitly — should dedupe
	h.RegisterGenericType(testInput{})

	sdl := h.SDL()

	// testInput should appear only once as an input
	count := countOccurrences(sdl, "input testInput")
	if count != 1 {
		t.Errorf("testInput appears %d times in SDL, want 1:\n%s", count, sdl)
	}
}

func TestSchemaHelper_NonPointerRequired(t *testing.T) {
	h := NewSchemaHelper()

	// Non-pointer structs should be registered by name
	name := h.RegisterInputType(testInput{})
	if name != "testInput" {
		t.Errorf("RegisterGenericType(testInput) = %q, want %q", name, "testInput")
	}

	sdl := h.SDL()
	if !contains(sdl, "input testInput") {
		t.Errorf("SDL missing 'input testInput':\n%s", sdl)
	}
}

func TestSchemaHelper_SDLEmpty(t *testing.T) {
	h := NewSchemaHelper()
	sdl := h.SDL()
	if sdl != "" {
		t.Errorf("Empty SchemaHelper.SDL() = %q, want empty string", sdl)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func countOccurrences(s, substr string) int {
	count := 0
	start := 0
	for {
		idx := -1
		for i := start; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		count++
		start = idx + len(substr)
	}
	return count
}
