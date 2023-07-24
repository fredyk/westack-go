package tests

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	wst "github.com/fredyk/westack-go/westack/common"
	"github.com/fredyk/westack-go/westack/model"
)

func Test_GenerateNextChunk_Error(t *testing.T) {

	t.Parallel()

	var err error

	// unmarshable map
	build, err := noteModel.Build(wst.M{
		"title":   "Note 0015",
		"body":    "This is a note",
		"invalid": make(chan int),
	}, model.NewBuildCache(), systemContext)
	assert.NoError(t, err)
	var input = model.InstanceA{build}

	chunkGenerator := model.NewInstanceAChunkGenerator(noteModel, input, "application/json")
	chunkGenerator.SetDebug(true)

	outBytes, err := io.ReadAll(chunkGenerator.Reader(systemContext))
	assert.Error(t, err)
	assert.Equal(t, 1, len(outBytes))
	assert.Equal(t, byte('['), outBytes[0])
}

func Test_ChannelChunkGeneratorError(t *testing.T) {

	t.Parallel()

	var err error

	// unmarshable map
	build, err := noteModel.Build(wst.M{
		"title":   "Note 0015",
		"body":    "This is a note",
		"invalid": make(chan int),
	}, model.NewBuildCache(), systemContext)
	assert.NoError(t, err)
	var input chan *model.Instance = make(chan *model.Instance)
	go func() {
		input <- &build
		close(input)
	}()
	cursor := model.NewChannelCursor(input)

	chunkGenerator := model.NewCursorChunkGenerator(noteModel, cursor)
	chunkGenerator.SetDebug(true)

	outBytes, err := io.ReadAll(chunkGenerator.Reader(systemContext))
	assert.Error(t, err)
	assert.Equal(t, 1, len(outBytes))
	assert.Equal(t, byte('['), outBytes[0])

}

func Test_ChannelChunkGeneratorClosedError(t *testing.T) {

	t.Parallel()

	var err error

	// unmarshable map
	//build, err := noteModel.Build(wst.M{
	//	"title":   "Note 0015",
	//	"body":    "This is a note",
	//	"invalid": make(chan int),
	//}, model.NewBuildCache(), systemContext)
	//assert.NoError(t, err)
	var input chan *model.Instance = make(chan *model.Instance)
	go func() {
		//input <- &build
		close(input)
	}()
	cursor := model.NewChannelCursor(input)
	cursor.(*model.ChannelCursor).Error(errors.New("closed"))

	chunkGenerator := model.NewCursorChunkGenerator(noteModel, cursor)
	chunkGenerator.SetDebug(true)

	outBytes, err := io.ReadAll(chunkGenerator.Reader(systemContext))
	assert.Error(t, err)
	assert.Equal(t, 1, len(outBytes))
	assert.Equal(t, byte('['), outBytes[0])

}
