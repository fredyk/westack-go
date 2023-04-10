package model

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

type Cursor interface {
	Next() (result *Instance, err error)
	HasNext() bool
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

func (cursor *mongoCursor) HasNext() bool {
	return cursor.mCursor.Next(cursor.ctx)
}

func (cursor *mongoCursor) All() (result InstanceA, err error) {
	err = cursor.mCursor.All(cursor.ctx, &result)
	return
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

func (cursor *errorCursor) HasNext() bool {
	return false
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

func (cursor *fixedLengthCursor) HasNext() bool {
	return cursor.index < len(cursor.instances)
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

type ChannelCursor struct {
	channel chan *Instance
	Err     error
}

func (cursor *ChannelCursor) Next() (result *Instance, err error) {
	if cursor.Err != nil {
		return result, cursor.Err
	}
	result = <-cursor.channel
	if result == nil {
		return result, nil
	}
	return
}

func (cursor *ChannelCursor) HasNext() bool {
	return cursor.Err == nil
}

func (cursor *ChannelCursor) All() (result InstanceA, err error) {
	for {
		if cursor.Err != nil {
			return result, cursor.Err
		}
		instance := <-cursor.channel
		if instance == nil {
			break
		}
		result = append(result, *instance)
	}
	return
}

func (cursor *ChannelCursor) Close() error {
	close(cursor.channel)
	return nil
}

func (cursor *ChannelCursor) Error(err error) {
	cursor.Err = err
}

func newChannelCursor(channel chan *Instance) Cursor {
	return &ChannelCursor{
		channel: channel,
	}
}
