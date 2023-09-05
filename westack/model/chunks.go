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

	SetDebug(bool)
	Reader(ctx *EventContext) io.Reader
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
	n = copy(p, reader.currentChunk.raw[reader.currentChunkReadIndex:])
	reader.currentChunkReadIndex += n

	if reader.debug {
		fmt.Printf("DEBUG: ChunkGeneratorReader.Read() returning %d bytes\n", n)
	}

	return
}

type cursorChunkGenerator struct {
	Debug        bool
	cursor       Cursor
	currentChunk Chunk
	isFirst      bool
	eof          bool
	docsCount    int
}

func (chunkGenerator *cursorChunkGenerator) ContentType() string {
	return "application/json"
}

func (chunkGenerator *cursorChunkGenerator) NextChunk() (chunk Chunk, err error) {
	if v, ok := chunkGenerator.cursor.(*ErrorCursor); ok {
		return chunk, v.Error()
	}
	err = chunkGenerator.GenerateNextChunk()
	chunk = chunkGenerator.currentChunk
	return
}

func (chunkGenerator *cursorChunkGenerator) GenerateNextChunk() (err error) {
	chunkGenerator.currentChunk.raw = nil
	chunkGenerator.currentChunk.length = 0
	if chunkGenerator.isFirst {
		chunkGenerator.currentChunk.raw = []byte{'['}
		chunkGenerator.currentChunk.length += 1
		chunkGenerator.isFirst = false
	} else if chunkGenerator.eof {
		return io.EOF
	} else {

		var nextInstance *Instance
		nextInstance, err = chunkGenerator.cursor.Next()
		if err != nil {
			if chunkGenerator.Debug {
				fmt.Printf("ERROR: ChunkGenerator.GenerateNextChunk() failed to get next instance: %v\n", err)
			}
			return err

		} else if nextInstance == nil {
			chunkGenerator.currentChunk.raw = []byte{']'}
			chunkGenerator.currentChunk.length += 1
			chunkGenerator.eof = true
		} else {
			nextInstance.HideProperties()
			asM := nextInstance.ToJSON()
			var asBytes []byte
			asBytes, err = easyjson.Marshal(asM)
			if err != nil {
				if chunkGenerator.Debug {
					fmt.Printf("ERROR: ChunkGenerator.GenerateNextChunk() failed to marshal instance: %v\n", err)
				}
				return err
			}

			if chunkGenerator.docsCount > 0 {
				chunkGenerator.currentChunk.raw = []byte{','}
				chunkGenerator.currentChunk.length += 1
			}

			chunkGenerator.currentChunk.raw = append(chunkGenerator.currentChunk.raw, asBytes...)
			chunkGenerator.currentChunk.length += len(asBytes)
			chunkGenerator.docsCount += 1
		}

	}

	return
}

func (chunkGenerator *cursorChunkGenerator) Reader(eventContext *EventContext) io.Reader {
	return &ChunkGeneratorReader{
		chunkGenerator: chunkGenerator,
		eventContext:   eventContext,
		debug:          chunkGenerator.Debug,
	}
}

func (chunkGenerator *cursorChunkGenerator) SetDebug(debug bool) {
	chunkGenerator.Debug = debug
}

func NewCursorChunkGenerator(loadedModel *Model, cursor Cursor) ChunkGenerator {
	result := cursorChunkGenerator{
		cursor:  cursor,
		Debug:   loadedModel.App.Debug,
		isFirst: true,
	}
	return &result
}
