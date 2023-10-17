package model

import (
	"fmt"
	"github.com/mailru/easyjson"
	"io"
)

type InstanceAChunkGenerator struct {
	Debug bool

	input             InstanceA
	totalChunks       int
	currentChunkIndex int
	currentChunk      Chunk
	contentType       string
}

func (chunkGenerator *InstanceAChunkGenerator) ContentType() string {
	return chunkGenerator.contentType
}

func (chunkGenerator *InstanceAChunkGenerator) obtainNextChunk() (chunk Chunk, err error) {
	err = chunkGenerator.GenerateNextChunk()
	if err != nil {
		return
	}
	chunk = chunkGenerator.currentChunk
	return
}

func (chunkGenerator *InstanceAChunkGenerator) NextChunk() (chunk Chunk, err error) {
	if chunkGenerator.currentChunkIndex == chunkGenerator.totalChunks {
		return chunk, io.EOF
	}
	chunk, err = chunkGenerator.obtainNextChunk()
	if err != nil {
		return
	}
	chunkGenerator.currentChunkIndex++
	return
}

func (chunkGenerator *InstanceAChunkGenerator) GenerateNextChunk() (err error) {
	chunkGenerator.currentChunk.raw = nil
	chunkGenerator.currentChunk.length = 0
	if chunkGenerator.currentChunkIndex == 0 {
		chunkGenerator.currentChunk.raw = []byte{'['}
		chunkGenerator.currentChunk.length += 1
	} else if chunkGenerator.currentChunkIndex == chunkGenerator.totalChunks-1 {
		chunkGenerator.currentChunk.raw = []byte{']'}
		chunkGenerator.currentChunk.length += 1
	} else {
		if chunkGenerator.currentChunkIndex > 1 {
			chunkGenerator.currentChunk.raw = []byte{','}
			chunkGenerator.currentChunk.length += 1
		}

		nextInstance := chunkGenerator.input[chunkGenerator.currentChunkIndex-1]
		nextInstance.HideProperties()
		asM := nextInstance.ToJSON()
		var asBytes []byte
		asBytes, err = easyjson.Marshal(&asM)
		if err != nil {
			if chunkGenerator.Debug {
				fmt.Printf("ERROR: ChunkGenerator.GenerateNextChunk() failed to marshal instance %d/%d: %v\n", chunkGenerator.currentChunkIndex, chunkGenerator.totalChunks, err)
			}
			return
		}
		chunkGenerator.currentChunk.raw = append(chunkGenerator.currentChunk.raw, asBytes...)
		chunkGenerator.currentChunk.length += len(asBytes)
	}
	if chunkGenerator.Debug {
		fmt.Printf("Generated chunk %d/%d\n", chunkGenerator.currentChunkIndex, chunkGenerator.totalChunks)
	}
	return
}

func (chunkGenerator *InstanceAChunkGenerator) Reader(eventContext *EventContext) io.Reader {
	return &ChunkGeneratorReader{
		chunkGenerator: chunkGenerator,
		eventContext:   eventContext,
		debug:          chunkGenerator.Debug,
	}

}

func (chunkGenerator *InstanceAChunkGenerator) SetDebug(debug bool) {
	chunkGenerator.Debug = debug
}

func NewInstanceAChunkGenerator(loadedModel *Model, input InstanceA, contentType string) ChunkGenerator {
	result := InstanceAChunkGenerator{
		contentType:       contentType,
		currentChunkIndex: 0,
		totalChunks:       len(input) + 2,
		input:             input,
		Debug:             loadedModel.App.Debug,
	}
	return &result
}
