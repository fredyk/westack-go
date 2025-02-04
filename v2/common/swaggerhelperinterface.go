package wst

type SwaggerHelper interface {
	// GetOpenAPI returns the OpenAPI specification as a map, or an error if it fails
	GetOpenAPI() (M, error)
	// CreateOpenAPI creates a new OpenAPI specification and saves it to disk, or returns an error if it fails
	CreateOpenAPI() error
	// AddPathSpec adds a path specification to the OpenAPI specification
	AddPathSpec(path string, verb string, verbSpec M, operationName string, modelName string)
	// RemovePathSpec removes a path specification from the OpenAPI specification
	RemovePathSpec(path string, verb string)
	// GetComponents returns the components of the OpenAPI specification
	GetComponents() M
	// Dump dumps the OpenAPI specification to disk, or returns an error if it fails
	Dump() error
}
