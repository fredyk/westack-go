package modelentities

import "github.com/fredyk/westack-go/lambdas/entities"

type ChannelCursor struct {
	channel      chan Instance
	Err          error
	UsedPipeline []entities.M
}

func (cursor *ChannelCursor) Next() (result Instance, err error) {
	err = cursor.Err
	if err != nil {
		return
	}
	result = <-cursor.channel
	err = cursor.Err
	return
}

func (cursor *ChannelCursor) All() (result []Instance, err error) {
	for {
		if err = cursor.Err; err != nil {
			return
		}
		instance := <-cursor.channel
		if instance == nil {
			break
		}
		result = append(result, instance)
	}
	err = cursor.Err
	return
}

func (cursor *ChannelCursor) Close() error {
	close(cursor.channel)
	return nil
}

func (cursor *ChannelCursor) Error(err error) {
	cursor.Err = err
}

func NewChannelCursor(channel chan Instance) Cursor {
	return &ChannelCursor{
		channel: channel,
	}
}
