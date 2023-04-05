package model

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mailru/easyjson"
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

	input        InstanceA
	chunks       []Chunk
	totalChunks  int
	currentChunk int
	contentType  string
}

func (chunkGenerator *InstanceAChunkGenerator) ContentType() string {
	return chunkGenerator.contentType
}

func (chunkGenerator *InstanceAChunkGenerator) NextChunk() (chunk Chunk, err error) {
	if chunkGenerator.currentChunk == chunkGenerator.totalChunks {
		return chunk, io.EOF
	}
	err = chunkGenerator.GenerateNextChunk()
	if err != nil {
		return
	}
	chunk = chunkGenerator.chunks[chunkGenerator.currentChunk]
	chunkGenerator.currentChunk++
	return
}

var activeChunks int32
var activeChunksMutex sync.RWMutex

func (chunkGenerator *InstanceAChunkGenerator) GenerateNextChunk() (err error) {
	activeChunksMutex.RLock()
	for activeChunks >= 6 {
		// fmt.Printf("Waiting for active chunks to finish: %d\n", activeChunks)
		activeChunksMutex.RUnlock()
		time.Sleep(16 * time.Millisecond)
		activeChunksMutex.RLock()
	}
	activeChunksMutex.RUnlock()
	activeChunksMutex.Lock()
	activeChunks++
	activeChunksMutex.Unlock()
	defer func() {
		activeChunksMutex.Lock()
		activeChunks--
		activeChunksMutex.Unlock()
	}()
	var nextChunk Chunk
	if chunkGenerator.currentChunk == 0 {
		nextChunk.raw = []byte{'['}
		nextChunk.length += 1
	} else if chunkGenerator.currentChunk == chunkGenerator.totalChunks-1 {
		nextChunk.raw = []byte{']'}
		nextChunk.length += 1
	} else {
		if chunkGenerator.currentChunk > 1 {
			nextChunk.raw = []byte{','}
			nextChunk.length += 1
		}

		nextInstance := chunkGenerator.input[chunkGenerator.currentChunk-1]
		nextInstance.HideProperties()
		asM := nextInstance.ToJSON()
		var asBytes []byte
		asBytes, err = easyjson.Marshal(asM)
		if err != nil {
			if chunkGenerator.Debug {
				fmt.Printf("ERROR: ChunkGenerator.GenerateNextChunk() failed to marshal instance %d/%d: %v\n", chunkGenerator.currentChunk, chunkGenerator.totalChunks, err)
			}
			return
		}
		nextChunk.raw = append(nextChunk.raw, asBytes...)
		nextChunk.length += len(asBytes)
	}
	chunkGenerator.chunks = append(chunkGenerator.chunks, nextChunk)
	if chunkGenerator.Debug {
		fmt.Printf("Generated chunk %d/%d\n", chunkGenerator.currentChunk, chunkGenerator.totalChunks)
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
	//	p[i] = reader.currentChunk.raw[reader.currentChunkReadIndex]
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
		contentType:  contentType,
		chunks:       []Chunk{},
		currentChunk: 0,
		totalChunks:  len(input) + 2,
		input:        input,
		Debug:        loadedModel.App.Debug,
	}
	return &result
}
