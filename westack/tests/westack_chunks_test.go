package tests

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
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

func Test_FixedBeforeLoadMock124401(t *testing.T) {

	t.Parallel()

	request, err := http.NewRequest("GET", "/api/v1/notes?mockResultTest124401=true", nil)
	assert.NoError(t, err)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	resp, err := executeRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	responseBody, err := parseResultAsJsonArray(resp)
	assert.NoError(t, err)
	assert.NotNil(t, responseBody)
	assert.Equal(t, 1, len(responseBody))
	assert.Equal(t, "mocked124401", responseBody[0].GetString("title"))

}

func Test_FixedBeforeLoadMock124402(t *testing.T) {

	t.Parallel()

	request, err := http.NewRequest("GET", "/api/v1/notes?mockResultTest124402=true", nil)
	assert.NoError(t, err)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	resp, err := executeRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	responseBody, err := parseResultAsJsonArray(resp)
	assert.NoError(t, err)
	assert.NotNil(t, responseBody)
	assert.Equal(t, 1, len(responseBody))
	assert.Equal(t, "mocked124402", responseBody[0].GetString("title"))

}

func Test_FixedBeforeLoadMock124403(t *testing.T) {

	t.Parallel()

	request, err := http.NewRequest("GET", "/api/v1/notes?mockResultTest124403=true", nil)
	assert.NoError(t, err)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	resp, err := executeRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	responseBody, err := parseResultAsJsonArray(resp)
	assert.NoError(t, err)
	assert.NotNil(t, responseBody)
	assert.Equal(t, 1, len(responseBody))
	assert.Equal(t, "mocked124403", responseBody[0].GetString("title"))

}

func Test_FixedBeforeLoadMock124404(t *testing.T) {

	t.Parallel()

	request, err := http.NewRequest("GET", "/api/v1/notes?mockResultTest124404=true", nil)
	assert.NoError(t, err)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	resp, err := executeRequest(request)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	responseBody, err := parseResultAsJsonArray(resp)
	assert.NoError(t, err)
	assert.NotNil(t, responseBody)
	assert.Equal(t, 1, len(responseBody))
	assert.Equal(t, "mocked124404", responseBody[0].GetString("title"))

}

func parseResultAsJsonArray(resp *http.Response) (responseBody wst.A, err error) {

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &responseBody)
	return responseBody, err

}

func executeRequest(request *http.Request) (*http.Response, error) {

	return executeRequestRaw(request)

}

func executeRequestRaw(request *http.Request) (*http.Response, error) {

	resp, err := app.Server.Test(request, 45000)
	if err != nil {
		return nil, err
	}
	return resp, nil

}
