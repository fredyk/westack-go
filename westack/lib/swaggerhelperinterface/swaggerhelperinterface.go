package swaggerhelperinterface

type SwaggerHelper interface {
	// GetOpenAPI returns the OpenAPI specification as a map, or an error if it fails
	GetOpenAPI() (map[string]interface{}, error)
	// CreateOpenAPI creates a new OpenAPI specification and saves it to disk, or returns an error if it fails
	CreateOpenAPI() error
	// AddPathSpec adds a path specification to the OpenAPI specification
	AddPathSpec(path string, verb string, verbSpec map[string]interface{})
	// Dump dumps the OpenAPI specification to disk, or returns an error if it fails
	Dump() error
}
