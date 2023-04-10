package datasource

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
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
}

func (cursor *fixedMongoCursor) Next(ctx context.Context) bool {
	if cursor.index >= cursor.totalCount {
		return false
	}
	return true
}

func (cursor *fixedMongoCursor) Decode(val interface{}) error {
	err := bson.Unmarshal(cursor.rawInputs[cursor.index], val)
	cursor.index++
	return err
}

func (cursor *fixedMongoCursor) All(ctx context.Context, results interface{}) error {
	// Unmarshal all the raw inputs
	// Treat results as a slice, grow it as needed
	if v, ok := results.(*[]interface{}); ok {
		for idx, rawInput := range cursor.rawInputs {
			// Grow the slice if needed, respecting it's type
			if idx >= len(*v) {
				// Get the type of the slice
				sliceType := reflect.TypeOf(*v)
				// Get the type of the elements of the slice
				elemType := sliceType.Elem()
				// Create a new element of the slice's type
				newElem := reflect.New(elemType)
				// Unmarshal the raw input into the new element
				err := bson.Unmarshal(rawInput, newElem.Interface())
				if err != nil {
					return err
				}
				// Append the new element to the slice
				*v = append(*v, newElem.Elem().Interface())
			} else {
				// Unmarshal the raw input into the existing element
				err := bson.Unmarshal(rawInput, (*v)[idx])
				if err != nil {
					return err
				}
			}

		}
	} else {
		return errors.New("results is not a pointer to a slice")
	}
	return nil
}

func (cursor *fixedMongoCursor) Close(ctx context.Context) error {
	return nil
}

func NewFixedMongoCursor(rawInputs [][]byte) MongoCursorI {
	return &fixedMongoCursor{
		rawInputs:  rawInputs,
		index:      0,
		totalCount: len(rawInputs),
	}
}
