package model

import (
	"fmt"
	"github.com/mailru/easyjson"
	"io"
)

type Chunk struct {
	raw    []byte
	length int
}

type ChunkGenerator interface {
	ContentType() string

	GenerateNextChunk() error
	NextChunk() (Chunk, error)
}

type InstanceAChunkGenerator struct {
	Debug bool

	input             InstanceA
	totalChunks       int
	currentChunkIndex int
	currentChunk      *Chunk
	nextChunk         *Chunk
	contentType       string
}

func (chunkGenerator *InstanceAChunkGenerator) ContentType() string {
	return chunkGenerator.contentType
}

func (chunkGenerator *InstanceAChunkGenerator) NextChunk() (chunk Chunk, err error) {
	if chunkGenerator.currentChunkIndex == chunkGenerator.totalChunks {
		return chunk, io.EOF
	}
	err = chunkGenerator.GenerateNextChunk()
	if err != nil {
		return
	}
	chunk = *chunkGenerator.currentChunk
	chunkGenerator.currentChunkIndex++
	return
}

func (chunkGenerator *InstanceAChunkGenerator) GenerateNextChunk() (err error) {
	var nextChunk Chunk
	if chunkGenerator.currentChunkIndex == 0 {
		nextChunk.raw = []byte{'['}
		nextChunk.length += 1
	} else if chunkGenerator.currentChunkIndex == chunkGenerator.totalChunks-1 {
		nextChunk.raw = []byte{']'}
		nextChunk.length += 1
	} else {
		if chunkGenerator.currentChunkIndex > 1 {
			nextChunk.raw = []byte{','}
			nextChunk.length += 1
		}

		nextInstance := chunkGenerator.input[chunkGenerator.currentChunkIndex-1]
		nextInstance.HideProperties()
		asM := nextInstance.ToJSON()
		var asBytes []byte
		asBytes, err = easyjson.Marshal(asM)
		if err != nil {
			if chunkGenerator.Debug {
				fmt.Printf("ERROR: ChunkGenerator.GenerateNextChunk() failed to marshal instance %d/%d: %v\n", chunkGenerator.currentChunkIndex, chunkGenerator.totalChunks, err)
			}
			return
		}
		nextChunk.raw = append(nextChunk.raw, asBytes...)
		nextChunk.length += len(asBytes)
	}
	chunkGenerator.currentChunk = &nextChunk
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

type ChunkGeneratorReader struct {
	chunkGenerator        ChunkGenerator
	eventContext          *EventContext
	currentChunk          Chunk
	currentChunkReadIndex int
	debug                 bool
}

func (reader *ChunkGeneratorReader) Read(p []byte) (n int, err error) {
	if reader.debug {
		fmt.Printf("DEBUG: ChunkGeneratorReader.Read() called with len(p)=%d\n", len(p))
	}
	if reader.currentChunkReadIndex == reader.currentChunk.length {
		if reader.debug {
			fmt.Printf("DEBUG: ChunkGeneratorReader.Read() reached end of chunk (%d, %d)\n", reader.currentChunkReadIndex, reader.currentChunk.length)
		}
		reader.currentChunk, err = reader.chunkGenerator.NextChunk()
		if err != nil {
			if err == io.EOF {
				if reader.debug {
					fmt.Printf("DEBUG: ChunkGeneratorReader.Read() reached EOF\n")
				}
			}
			return n, err
		}
		reader.currentChunkReadIndex = 0
	}
	//for i := 0; i < len(p); i++ {
	//	p[i] = reader.currentChunkIndex.raw[reader.currentChunkReadIndex]
	//	reader.currentChunkReadIndex++
	//	n++
	//}
	n = copy(p, reader.currentChunk.raw[reader.currentChunkReadIndex:])
	reader.currentChunkReadIndex += n

	if reader.debug {
		fmt.Printf("DEBUG: ChunkGeneratorReader.Read() returning %d bytes\n", n)
	}

	return
}

func NewInstanceAChunkGenerator(loadedModel *Model, input InstanceA, contentType string) *InstanceAChunkGenerator {
	result := InstanceAChunkGenerator{
		contentType:       contentType,
		currentChunkIndex: 0,
		totalChunks:       len(input) + 2,
		input:             input,
		Debug:             loadedModel.App.Debug,
	}
	return &result
}
