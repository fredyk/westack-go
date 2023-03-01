package model

import (
	"fmt"
	"io"

	"github.com/mailru/easyjson"
)

type Chunk struct {
	raw   []byte
	first bool
	last  bool
	error error
}

type ChunkGenerator interface {
	ContentType() string

	GenerateNextChunk() bool
	NextChunk() (Chunk, error)
}

type InstanceAChunkGenerator struct {
	input InstanceA

	chunks       []Chunk
	totalChunks  int
	currentChunk int
	contentType  string
}

func (chunkGenerator *InstanceAChunkGenerator) ContentType() string {
	return chunkGenerator.contentType
}

func (chunkGenerator *InstanceAChunkGenerator) NextChunk() (chunk Chunk, err error) {
	didGenerateChunk := chunkGenerator.GenerateNextChunk()
	if chunkGenerator.currentChunk == chunkGenerator.totalChunks {
		if !didGenerateChunk {
			return chunk, io.EOF
		}
	} else if chunkGenerator.currentChunk > chunkGenerator.totalChunks {
		return chunk, io.ErrUnexpectedEOF
	}
	chunk = chunkGenerator.chunks[chunkGenerator.currentChunk]
	if chunkGenerator.currentChunk == 0 {
		chunk.first = true
	} else if chunkGenerator.currentChunk == chunkGenerator.totalChunks-1 {
		chunk.last = true
	}
	chunkGenerator.currentChunk++
	return chunk, nil
}

func (chunkGenerator *InstanceAChunkGenerator) Reset() {
	chunkGenerator.currentChunk = 0
}

func (chunkGenerator *InstanceAChunkGenerator) GenerateNextChunk() bool {
	if chunkGenerator.currentChunk == chunkGenerator.totalChunks {
		//fmt.Printf("ERROR: ChunkGenerator.GenerateNextChunk() called after EOF\n")
		return false
	}
	var nextChunk Chunk
	if chunkGenerator.currentChunk == 0 {
		nextChunk.raw = []byte{'['}
	} else {
		nextChunk.raw = []byte{','}
	}

	nextInstance := chunkGenerator.input[chunkGenerator.currentChunk]
	nextInstance.HideProperties()
	asM := nextInstance.ToJSON()
	asBytes, err := easyjson.Marshal(asM)
	if err != nil {
		nextChunk.error = err
		return false
	}
	nextChunk.raw = append(nextChunk.raw, asBytes...)

	if chunkGenerator.currentChunk == chunkGenerator.totalChunks-1 {
		nextChunk.raw = append(nextChunk.raw, ']')
	}
	chunkGenerator.chunks = append(chunkGenerator.chunks, nextChunk)
	//fmt.Printf("Generated chunk %d/%d\n", chunkGenerator.currentChunk, chunkGenerator.totalChunks)
	return true
}

func (chunkGenerator *InstanceAChunkGenerator) Reader(eventContext *EventContext) io.Reader {
	return ChunkGeneratorReader{
		chunkGenerator,
		eventContext,
	}

}

type ChunkGeneratorReader struct {
	chunkGenerator ChunkGenerator
	eventContext   *EventContext
}

func (reader ChunkGeneratorReader) Read(p []byte) (n int, err error) {
	chunk, err := reader.chunkGenerator.NextChunk()
	if err != nil {
		return 0, err
	}
	//n, err = reader.eventContext.Ctx.Write(chunk.raw)
	n, err = writeToBuffer(p, chunk.raw)
	if err != nil {
		return 0, err
	}
	if n != len(chunk.raw) {
		return 0, fmt.Errorf("failed to write chunk")
	}
	return n, nil
}

func writeToBuffer(out []byte, in []byte) (int, error) {
	inputLen := len(in)
	if len(out) < inputLen {
		//return 0, fmt.Errorf("output buffer too small")
		growBy := inputLen - len(out)
		out = append(out, make([]byte, growBy)...)
	}
	for i := 0; i < inputLen; i++ {
		out[i] = in[i]
	}
	return inputLen, nil
}

func NewInstanceAChunkGenerator(input InstanceA, contentType string) InstanceAChunkGenerator {
	result := InstanceAChunkGenerator{
		contentType:  contentType,
		chunks:       []Chunk{},
		currentChunk: 0,
		totalChunks:  len(input),
		input:        input,
	}
	return result
}
