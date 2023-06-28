package model

import (
	"context"

	"github.com/fredyk/westack-go/westack/datasource"
)

type Cursor interface {
	Next() (result *Instance, err error)
	All() (result InstanceA, err error)
	Close() error
}

type MongoCursor struct {
	mCursor datasource.MongoCursorI
	ctx     context.Context
	Err     error
}

func (cursor *MongoCursor) Next() (result *Instance, err error) {
	if cursor.Err != nil {
		return result, cursor.Err
	}
	if cursor.mCursor.Next(cursor.ctx) {
		err = cursor.mCursor.Decode(&result)
		return
	} else {
		return result, nil
	}
}

func (cursor *MongoCursor) All() (result InstanceA, err error) {
	err = cursor.mCursor.All(cursor.ctx, &result)
	return
}

func (cursor *MongoCursor) Close() error {
	return cursor.mCursor.Close(cursor.ctx)
}

func (cursor *MongoCursor) Error(err error) {
	cursor.Err = err
}

func newMongoCursor(ctx context.Context, mCursor datasource.MongoCursorI) Cursor {
	return &MongoCursor{
		mCursor: mCursor,
		ctx:     ctx,
	}
}

type ErrorCursor struct {
	err error
}

func (cursor *ErrorCursor) Next() (result *Instance, err error) {
	return result, cursor.err
}

func (cursor *ErrorCursor) All() (result InstanceA, err error) {
	return result, cursor.err
}

func (cursor *ErrorCursor) Close() error {
	return nil
}

func (cursor *ErrorCursor) Error() error {
	return cursor.err
}

func newErrorCursor(err error) Cursor {
	return &ErrorCursor{
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

func (cursor *fixedLengthCursor) Close() error {
	return nil
}

func newFixedLengthCursor(instances InstanceA) Cursor {
	return &fixedLengthCursor{
		instances: instances,
		index:     0,
	}
}
