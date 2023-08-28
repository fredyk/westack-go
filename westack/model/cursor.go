package model

type Cursor interface {
	Next() (result *Instance, err error)
	All() (result InstanceA, err error)
	Close() error
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
