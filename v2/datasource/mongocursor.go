package datasource

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"reflect"
)

type MongoCursorI interface {
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	All(ctx context.Context, val interface{}) error
	Close(ctx context.Context) error
}

type CursorIterator chan interface{}

type fixedMongoCursor struct {
	rawInputs  [][]byte
	index      int
	totalCount int
	registry   *bsoncodec.Registry
}

func (cursor *fixedMongoCursor) Next(ctx context.Context) bool {
	if cursor.index >= cursor.totalCount {
		return false
	}
	return true
}

func (cursor *fixedMongoCursor) Decode(val interface{}) error {
	err := bson.UnmarshalWithRegistry(cursor.registry, cursor.rawInputs[cursor.index], val)
	cursor.index++
	return err
}

func (cursor *fixedMongoCursor) All(ctx context.Context, results interface{}) error {

	resultsVal := reflect.ValueOf(results)
	if resultsVal.Kind() != reflect.Ptr {
		return errors.New("results is not a pointer")
	}

	// Dereference the pointer
	resultsVal = resultsVal.Elem()

	// Check if the pointer is to a slice
	if resultsVal.Kind() != reflect.Slice {
		return errors.New("results is not a pointer to a slice")
	}

	// Get the slice's type
	sliceType := resultsVal.Type()

	// Get the slice's element type
	elemType := sliceType.Elem()

	// Create a new slice
	newSlice := reflect.MakeSlice(sliceType, 0, 0)

	//// Unmarshal all the raw inputs
	//// Treat results as a slice, grow it as needed
	//if v, ok := results.(*[]interface{}); ok {
	for _, rawInput := range cursor.rawInputs {

		// Create a new element of the slice's type
		newElem := reflect.New(elemType)
		// Unmarshal the raw input into the new element
		err := bson.UnmarshalWithRegistry(cursor.registry, rawInput, newElem.Interface())
		if err != nil {
			return err
		}
		// Append the new element to the slice
		newSlice = reflect.Append(newSlice, newElem.Elem())

	}
	//} else {
	//	return errors.New("results is not a pointer to a slice")
	//}

	// Set the results to the new slice
	resultsVal.Set(newSlice)

	return nil
}

func (cursor *fixedMongoCursor) Close(ctx context.Context) error {
	return nil
}

func NewFixedMongoCursor(registry *bsoncodec.Registry, rawInputs [][]byte) MongoCursorI {
	return &fixedMongoCursor{
		rawInputs:  rawInputs,
		index:      0,
		totalCount: len(rawInputs),
		registry:   registry,
	}
}
