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
	return result, cursor.Error()
}

func (cursor *ErrorCursor) All() (result InstanceA, err error) {
	return result, cursor.Error()
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

type FixedLengthCursor struct {
	instances InstanceA
	index     int
}

func (cursor *FixedLengthCursor) Next() (result *Instance, err error) {
	if cursor.index >= len(cursor.instances) {
		return result, nil
	}
	result = &cursor.instances[cursor.index]
	cursor.index++
	return
}

func (cursor *FixedLengthCursor) All() (result InstanceA, err error) {
	return cursor.instances, nil
}

func (cursor *FixedLengthCursor) Close() error {
	return nil
}

func newFixedLengthCursor(instances InstanceA) Cursor {
	return &FixedLengthCursor{
		instances: instances,
		index:     0,
	}
}
