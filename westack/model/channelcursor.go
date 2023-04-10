package model

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

func NewChannelCursor(channel chan *Instance) Cursor {
	return &ChannelCursor{
		channel: channel,
	}
}
