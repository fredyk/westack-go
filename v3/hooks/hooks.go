package hooks

import (
	"context"
	"fmt"
)

// Operation identifies the repository operation that triggers hooks.
type Operation string

const (
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
	OpFind   Operation = "find"
)

// HookFunc is a function that runs before or after a repository operation.
// For "before" hooks: receives a pointer to the entity; can modify it in place.
// Return a non-nil error to abort the operation.
// For "after" hooks: receives the persisted result; can post-process.
type HookFunc[T any] func(ctx context.Context, entity *T) error

// Registry holds registered before/after hooks per operation.
type Registry[T any] struct {
	before map[Operation][]HookFunc[T]
	after  map[Operation][]HookFunc[T]
}

// NewRegistry creates an empty hook registry.
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		before: make(map[Operation][]HookFunc[T]),
		after:  make(map[Operation][]HookFunc[T]),
	}
}

// Before registers a hook that runs before the given operation.
// Hooks run in the order they are registered.
func (r *Registry[T]) Before(op Operation, fn HookFunc[T]) {
	r.before[op] = append(r.before[op], fn)
}

// After registers a hook that runs after the given operation.
// Hooks run in the order they are registered.
func (r *Registry[T]) After(op Operation, fn HookFunc[T]) {
	r.after[op] = append(r.after[op], fn)
}

// RunBefore executes all before-hooks for the given operation.
// Returns the first non-nil error (which aborts the operation).
// If a hook panics, the panic is recovered and returned as an error.
func (r *Registry[T]) RunBefore(op Operation, ctx context.Context, entity *T) error {
	for _, fn := range r.before[op] {
		if err := callHook(fn, ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// RunAfter executes all after-hooks for the given operation.
// Returns the first non-nil error.
// If a hook panics, the panic is recovered and returned as an error.
func (r *Registry[T]) RunAfter(op Operation, ctx context.Context, entity *T) error {
	for _, fn := range r.after[op] {
		if err := callHook(fn, ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// callHook wraps a HookFunc to recover from panics and return them as errors.
func callHook[T any](fn HookFunc[T], ctx context.Context, entity *T) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("hook panic: %w", e)
			} else {
				err = fmt.Errorf("hook panic: %v", r)
			}
		}
	}()
	return fn(ctx, entity)
}
