package tests

import (
	"errors"
	"fmt"
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	"io"
	"net/http"
	"testing"

	"github.com/mailru/easyjson"

	"github.com/stretchr/testify/assert"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
)

func Test_GenerateNextChunk_Error(t *testing.T) {

	t.Parallel()

	var err error

	// unmarshable map
	build, err := noteModel.Build(wst.M{
		"title":   "Note 0015",
		"body":    "This is a note",
		"invalid": make(chan int),
	}, systemContext)
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
	}, systemContext)
	assert.NoError(t, err)
	var input chan model.Instance = make(chan model.Instance)
	go func() {
		input <- build
		close(input)
	}()
	cursor := model.NewChannelCursor(input)

	chunkGenerator := model.NewCursorChunkGenerator(noteModel, cursor)
	chunkGenerator.SetDebug(true)

	outBytes, err := io.ReadAll(chunkGenerator.Reader(systemContext))
	assert.Error(t, err)
	assert.Equal(t, 0, len(outBytes))

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
	var input chan model.Instance = make(chan model.Instance)
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
	assert.Equal(t, 0, len(outBytes))

}

func Test_FixedBeforeLoadMock124401(t *testing.T) {

	t.Parallel()

	resp, err := wstfuncs.InvokeApiFullResponse("GET", "/notes?mockResultTest124401=true", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
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

	resp, err := wstfuncs.InvokeApiFullResponse("GET", "/notes?mockResultTest124402=true", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
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

	resp, err := wstfuncs.InvokeApiFullResponse("GET", "/notes?mockResultTest124403=true", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
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

	resp, err := wstfuncs.InvokeApiFullResponse("GET", "/notes?mockResultTest124404=true", nil, wst.M{
		"Authorization": fmt.Sprintf("Bearer %s", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	responseBody, err := parseResultAsJsonArray(resp)
	assert.NoError(t, err)
	assert.NotNil(t, responseBody)
	assert.Equal(t, 1, len(responseBody))
	assert.Equal(t, "mocked124404", responseBody[0].GetString("title"))

}

func Test_AfterLoadShouldReturnEmpty(t *testing.T) {

	t.Parallel()

	// create 5 notes
	for i := 0; i < 5; i++ {
		note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
			"title": fmt.Sprintf("Note %d", i+1),
			"body":  fmt.Sprintf("This is note %d", i+1),
		}, wst.M{"Content-Type": "application/json"})
		assert.NoError(t, err)
		assert.NotEmpty(t, note.GetString("id"))
	}

	resp, err := invokeApiAsRandomAccount(t, "GET", "/notes?forceError1753=true", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "forced error 1753", resp.GetString("error.message"))
	// "after load" cannot handle errors. It skips failed instances.
	//assert.Equal(t, 0, len(resp))

}

func Test_BeforeBuildReturnsError(t *testing.T) {

	t.Parallel()

	resp, err := invokeApiAsRandomAccount(t, "GET", "/notes?forceError1556=true", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "error in __operation__before_build: forced error 1556", resp.GetString("error.message"))
}

func parseResultAsJsonArray(resp *http.Response) (responseBody wst.A, err error) {

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = easyjson.Unmarshal(bytes, &responseBody)
	return responseBody, err

}
