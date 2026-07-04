package datasource

import "context"

// Cursor is the generic cursor interface replacing MongoCursorI.
// It iterates and decodes rows without coupling to BSON or any specific serialization format.
type Cursor interface {
	// Next advances the cursor to the next result. Returns false when exhausted or on error.
	Next(ctx context.Context) bool
	// Decode decodes the current result into val.
	Decode(val interface{}) error
	// All decodes all remaining results into val (must be a pointer to a slice).
	All(ctx context.Context, val interface{}) error
	// Close releases resources held by the cursor.
	Close(ctx context.Context) error
	// Err returns the first error encountered by the cursor.
	Err() error
}
