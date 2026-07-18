package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/fredyk/westack-go/v3/datasource"
	"github.com/fredyk/westack-go/v3/hooks"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// Repository is a generic CRUD repository parameterized by a Go struct type T.
// It uses reflection on json struct tags to map between T and map[string]interface{}.
type Repository[T any] struct {
	conn  datasource.PersistedConnector
	model datasource.ModelDef
	hooks *hooks.Registry[T]
}

// New creates a new Repository[T] backed by the given connector and model definition.
func New[T any](conn datasource.PersistedConnector, model datasource.ModelDef) *Repository[T] {
	return &Repository[T]{conn: conn, model: model}
}

// NewWithHooks creates a new Repository[T] with a hook registry for before/after dispatch.
func NewWithHooks[T any](conn datasource.PersistedConnector, model datasource.ModelDef, hookReg *hooks.Registry[T]) *Repository[T] {
	return &Repository[T]{conn: conn, model: model, hooks: hookReg}
}

// Create persists a new entity, returning the persisted version with any generated fields.
func (r *Repository[T]) Create(ctx context.Context, entity *T) (*T, error) {
	if entity == nil {
		return nil, fmt.Errorf("repository: Create: entity is nil")
	}
	if err := r.runBefore(ctx, hooks.OpCreate, entity); err != nil {
		return nil, err
	}
	data, err := structToMap(entity)
	if err != nil {
		return nil, fmt.Errorf("repository: structToMap: %w", err)
	}
	doc, err := r.conn.Create(ctx, r.model.Collection, data)
	if err != nil {
		return nil, fmt.Errorf("repository: Create: %w", err)
	}
	result := new(T)
	if err := mapToStruct(doc, result); err != nil {
		return nil, fmt.Errorf("repository: mapToStruct: %w", err)
	}
	if err := r.runAfter(ctx, hooks.OpCreate, result); err != nil {
		return nil, err
	}
	return result, nil
}

// FindById retrieves an entity by its primary key. Returns (nil, nil) if not found.
func (r *Repository[T]) FindById(ctx context.Context, id interface{}) (*T, error) {
	doc, err := r.conn.FindById(ctx, r.model.Collection, id)
	if err != nil {
		return nil, fmt.Errorf("repository: FindById: %w", err)
	}
	if doc == nil {
		return nil, nil
	}
	result := new(T)
	if err := mapToStruct(doc, result); err != nil {
		return nil, fmt.Errorf("repository: mapToStruct: %w", err)
	}
	return result, nil
}

// FindMany retrieves all entities matching the given query. Pass nil for no filters.
func (r *Repository[T]) FindMany(ctx context.Context, query *datasource.Query) ([]T, error) {
	cursor, err := r.conn.FindMany(ctx, r.model.Collection, query)
	if err != nil {
		return nil, fmt.Errorf("repository: FindMany: %w", err)
	}
	defer cursor.Close(ctx)

	var results []T
	for cursor.Next(ctx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("repository: cursor.Decode: %w", err)
		}
		var item T
		if err := mapToStruct(doc, &item); err != nil {
			return nil, fmt.Errorf("repository: mapToStruct: %w", err)
		}
		results = append(results, item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("repository: cursor.Err: %w", err)
	}
	return results, nil
}

// UpdateById updates an entity by ID with the given patch and returns the updated version.
func (r *Repository[T]) UpdateById(ctx context.Context, id interface{}, patch *T) (*T, error) {
	if err := r.runBefore(ctx, hooks.OpUpdate, patch); err != nil {
		return nil, err
	}
	data, err := structToMap(patch)
	if err != nil {
		return nil, fmt.Errorf("repository: structToMap: %w", err)
	}
	doc, err := r.conn.UpdateById(ctx, r.model.Collection, id, data)
	if err != nil {
		return nil, fmt.Errorf("repository: UpdateById: %w", err)
	}
	if doc == nil {
		return nil, nil
	}
	result := new(T)
	if err := mapToStruct(doc, result); err != nil {
		return nil, fmt.Errorf("repository: mapToStruct: %w", err)
	}
	if err := r.runAfter(ctx, hooks.OpUpdate, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteById removes an entity by ID.
func (r *Repository[T]) DeleteById(ctx context.Context, id interface{}) error {
	_, err := r.conn.DeleteById(ctx, r.model.Collection, id)
	if err != nil {
		return fmt.Errorf("repository: DeleteById: %w", err)
	}
	return nil
}

// Count returns the number of entities matching the given filter. Pass nil for all.
func (r *Repository[T]) Count(ctx context.Context, filter *datasource.Filter) (int64, error) {
	n, err := r.conn.Count(ctx, r.model.Collection, filter)
	if err != nil {
		return 0, fmt.Errorf("repository: Count: %w", err)
	}
	return n, nil
}

// ── Hook dispatch helpers ────────────────────────────────────────────────────

func (r *Repository[T]) runBefore(ctx context.Context, op hooks.Operation, entity *T) error {
	if r.hooks == nil {
		return nil
	}
	return r.hooks.RunBefore(op, ctx, entity)
}

func (r *Repository[T]) runAfter(ctx context.Context, op hooks.Operation, entity *T) error {
	if r.hooks == nil {
		return nil
	}
	return r.hooks.RunAfter(op, ctx, entity)
}

// ── Reflection mappers ───────────────────────────────────────────────────────

// structToMap converts a struct to a map using json struct tag names.
// Unexported fields and fields with json:"-" are skipped.
// Embedded structs are flattened: their exported fields appear at the top level.
// Pointer fields are dereferenced; nil pointers are stored as nil.
// Fields with json:",omitempty" are skipped when their value is the zero value.
func structToMap(v interface{}) (map[string]interface{}, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("structToMap: expected struct, got %s", rv.Kind())
	}

	result := make(map[string]interface{})
	for i := 0; i < rv.Type().NumField(); i++ {
		if err := collectField(result, rv.Field(i), rv.Type().Field(i)); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// collectField processes a single struct field and adds it to result.
// For embedded (anonymous) fields, it recursively collects the inner struct's fields.
func collectField(result map[string]interface{}, fieldVal reflect.Value, field reflect.StructField) error {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		return nil
	}

	// Embedded (anonymous) struct: flatten its fields
	if field.Anonymous {
		if jsonTag != "" {
			// Embedded field has an explicit json name: treat as a single field
			key := strings.Split(jsonTag, ",")[0]
			if !fieldVal.CanInterface() {
				return nil
			}
			result[key] = fieldVal.Interface()
		}
		// If no json tag on embedded field, recurse into its fields
		elemKind := fieldVal.Kind()
		if elemKind == reflect.Ptr {
			if fieldVal.IsNil() {
				return nil
			}
			elemKind = fieldVal.Elem().Kind()
		}
		if elemKind == reflect.Struct {
			for j := 0; j < fieldVal.Type().NumField(); j++ {
				if err := collectField(result, fieldVal.Field(j), fieldVal.Type().Field(j)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	key := strings.Split(jsonTag, ",")[0]
	if key == "" {
		// No json tag: skip unexported fields
		if !field.IsExported() {
			return nil
		}
		return nil
	}

	if !fieldVal.CanInterface() {
		return nil
	}

	// Check omitempty
	hasOmitEmpty := strings.Contains(jsonTag, ",omitempty")
	if hasOmitEmpty && isZeroValue(fieldVal) {
		return nil
	}

	result[key] = derefPointer(fieldVal)
	return nil
}

// isZeroValue reports whether a reflect.Value holds the zero value of its type.
func isZeroValue(v reflect.Value) bool {
	return v.IsZero()
}

// derefPointer returns the dereferenced value for pointer fields.
// If the pointer is nil, returns nil.
func derefPointer(v reflect.Value) interface{} {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		return v.Elem().Interface()
	}
	return v.Interface()
}

// mapToStruct decodes a map into a struct using json struct tag names.
// Embedded structs are handled by delegating to their fields.
func mapToStruct(m map[string]interface{}, val interface{}) error {
	rv := reflect.ValueOf(val).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldVal := rv.Field(i)

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Embedded struct: delegate to its fields regardless of json tag
		if field.Anonymous {
			elemKind := fieldVal.Kind()
			if elemKind == reflect.Ptr {
				if fieldVal.IsNil() {
					fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				}
				elemKind = fieldVal.Elem().Kind()
			}
			if elemKind == reflect.Struct {
				if err := mapToStruct(m, fieldVal.Addr().Interface()); err != nil {
					return fmt.Errorf("embedded %s: %w", field.Name, err)
				}
				continue
			}
		}

		key := strings.Split(jsonTag, ",")[0]
		if key == "" {
			if !field.IsExported() {
				continue
			}
			continue
		}

		if !fieldVal.CanSet() {
			continue
		}

		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}

		if err := setField(fieldVal, raw); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

// setField assigns a value from an interface{} to a reflect.Value, handling type conversions.
func setField(fieldVal reflect.Value, raw interface{}) error {
	// Handle pointer target
	if fieldVal.Kind() == reflect.Ptr {
		if fieldVal.IsNil() {
			fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
		}
		return setField(fieldVal.Elem(), raw)
	}

	rawVal := reflect.ValueOf(raw)
	if rawVal.Type() == fieldVal.Type() {
		fieldVal.Set(rawVal)
		return nil
	}

	switch fieldVal.Kind() {
	case reflect.String:
		if s, ok := raw.(string); ok {
			fieldVal.SetString(s)
		}
	case reflect.Int, reflect.Int64:
		switch n := raw.(type) {
		case int64:
			fieldVal.SetInt(n)
		case int:
			fieldVal.SetInt(int64(n))
		case float64:
			fieldVal.SetInt(int64(n))
		}
	case reflect.Float64:
		if f, ok := raw.(float64); ok {
			fieldVal.SetFloat(f)
		}
	case reflect.Bool:
		if b, ok := raw.(bool); ok {
			fieldVal.SetBool(b)
		}
	case reflect.Slice:
		if fb, ok := raw.([]byte); ok {
			fieldVal.SetBytes(fb)
		}
	}
	return nil
}
