package tests

import (
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
	chunkGenerator.Debug = true

	outBytes, err := io.ReadAll(chunkGenerator.Reader(systemContext))
	assert.Error(t, err)
	assert.Equal(t, 1, len(outBytes))
	assert.Equal(t, byte('['), outBytes[0])
}
