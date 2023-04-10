package model

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type Cursor interface {
	Next() (result *Instance, err error)
	All() (result InstanceA, err error)
}

type mongoCursor struct {
	mCursor *mongo.Cursor
	ctx     context.Context
}

func (cursor *mongoCursor) Next() (result *Instance, err error) {
	if cursor.mCursor.Next(cursor.ctx) {
		err = cursor.mCursor.Decode(&result)
		return
	} else {
		return result, nil
	}
}

func (cursor *mongoCursor) All() (result InstanceA, err error) {
	err = cursor.mCursor.All(cursor.ctx, &result)
	return
}

func (cursor *mongoCursor) Close() error {
	return cursor.mCursor.Close(cursor.ctx)
}

func newMongoCursor(ctx context.Context, mCursor *mongo.Cursor) Cursor {
	return &mongoCursor{
		mCursor: mCursor,
		ctx:     ctx,
	}
}

type errorCursor struct {
	err error
}

func (cursor *errorCursor) Next() (result *Instance, err error) {
	return result, cursor.err
}

func (cursor *errorCursor) All() (result InstanceA, err error) {
	return result, cursor.err
}

func newErrorCursor(err error) Cursor {
	return &errorCursor{
		err: err,
	}
}

type fixedLengthCursor struct {
	instances InstanceA
	index     int
}

func (cursor *fixedLengthCursor) Next() (result *Instance, err error) {
	if cursor.index >= len(cursor.instances) {
		return result, nil
	}
	result = &cursor.instances[cursor.index]
	cursor.index++
	return
}

func (cursor *fixedLengthCursor) All() (result InstanceA, err error) {
	return cursor.instances, nil
}

func newFixedLengthCursor(instances InstanceA) Cursor {
	return &fixedLengthCursor{
		instances: instances,
		index:     0,
	}
}
