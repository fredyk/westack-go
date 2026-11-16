package tests

import (
	"fmt"
	"testing"

	"github.com/fredyk/westack-go/client/v2/wstfuncs"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/stretchr/testify/assert"
)

func Test_IntegerPropertyIsStoredAsInt64OnCreate(t *testing.T) {
	t.Parallel()

	created, err := noteModel.Create(wst.M{"wholeCount": float64(1002000)}, systemContext)
	assert.NoError(t, err)

	found, err := noteModel.FindById(created.GetID(), nil, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(1002000), found.ToJSON()["wholeCount"])
}

func Test_IntegerPropertyIsStoredAsInt64OnUpdate(t *testing.T) {
	t.Parallel()

	created, err := noteModel.Create(wst.M{"title": "integer update"}, systemContext)
	assert.NoError(t, err)

	_, err = created.UpdateAttributes(wst.M{"wholeCount": float64(7)}, systemContext)
	assert.NoError(t, err)
	found, err := noteModel.FindById(created.GetID(), nil, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(7), found.ToJSON()["wholeCount"])

	_, err = noteModel.UpdateById(created.GetID(), wst.M{"wholeCount": float64(8)}, systemContext)
	assert.NoError(t, err)
	found, err = noteModel.FindById(created.GetID(), nil, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, int64(8), found.ToJSON()["wholeCount"])
}

func Test_IntegerPropertyWithDecimalsIsAValidationError(t *testing.T) {
	t.Parallel()

	_, err := noteModel.Create(wst.M{"wholeCount": 2.5}, systemContext)
	assertIntegerValidationError(t, err)

	created, err := noteModel.Create(wst.M{"title": "integer reject"}, systemContext)
	assert.NoError(t, err)
	_, err = created.UpdateAttributes(wst.M{"wholeCount": 3.25}, systemContext)
	assertIntegerValidationError(t, err)
	_, err = noteModel.UpdateById(created.GetID(), wst.M{"wholeCount": 3.75}, systemContext)
	assertIntegerValidationError(t, err)
}

func Test_IntegerPropertyIsAValidationErrorOverHttp(t *testing.T) {
	t.Parallel()

	resp, err := wstfuncs.InvokeApiFullResponse("POST", "/notes", wst.M{"wholeCount": 1.5}, wst.M{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %v", randomAccountToken.GetString("id")),
	})
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	created, err := invokeApiAsRandomAccount("POST", "/notes", wst.M{"wholeCount": 1002000}, wst.M{"Content-Type": "application/json"})
	assert.NoError(t, err)
	assert.EqualValues(t, 1002000, created["wholeCount"])
}

func assertIntegerValidationError(t *testing.T, err error) {
	t.Helper()
	if !assert.Error(t, err) {
		return
	}
	var wstErr *wst.WeStackError
	if !assert.ErrorAs(t, err, &wstErr) {
		return
	}
	assert.Equal(t, 400, wstErr.FiberError.Code)
	assert.Equal(t, "ERR_VALIDATION", wstErr.Code)
	assert.Equal(t, "ValidationError", wstErr.Name)
	assert.Equal(t, wst.M{"wholeCount": []string{"integer"}}, wstErr.Details["codes"])
}
