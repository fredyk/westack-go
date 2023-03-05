package tests

import (
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_GetM(t *testing.T) {

	t.Parallel()

	m := wst.M{"a": wst.M{"b": "z"}}
	assert.Equal(t, "z", m.GetM("a").GetString("b"))
}

func Test_GetM_WithNativeMap(t *testing.T) {

	t.Parallel()

	m := wst.M{"a": map[string]interface{}{"b": "z"}}
	assert.Equal(t, "z", m.GetM("a").GetString("b"))
}

func Test_GetM_NonexistentKey(t *testing.T) {

	t.Parallel()

	m := wst.M{"a": wst.M{"b": "z"}}
	assert.Nil(t, m.GetM("c"))
}

func Test_GetM_NilInput(t *testing.T) {

	t.Parallel()

	var m wst.M = nil
	assert.Nil(t, m.GetM("a"))
}

func Test_GetString_NilMap(t *testing.T) {

	t.Parallel()

	var m wst.M = nil
	assert.Equal(t, "", m.GetString("a"))
}

func Test_GetString_NonexistentKey(t *testing.T) {

	t.Parallel()

	m := wst.M{"a": wst.M{"b": "z"}}
	assert.Equal(t, "", m.GetString("c"))
}

func Test_Error(t *testing.T) {

	t.Parallel()

	err := wst.WeStackError{
		FiberError: fiber.ErrUnauthorized,
		Code:       "UNAUTHORIZED",
		Details:    fiber.Map{"message": "Unauthorized"},
		Name:       "Error",
	}
	// test err.Error()
	assert.Equal(t, "401 Unauthorized: {\"message\":\"Unauthorized\"}", err.Error())
}

func Test_Error_WithErorOnMarshalDetails(t *testing.T) {

	t.Parallel()

	err := wst.WeStackError{
		FiberError: fiber.ErrUnauthorized,
		Code:       "UNAUTHORIZED",
		Details:    fiber.Map{"message": make(chan int)},
		Name:       "Error",
	}
	// test err.Error()
	assert.Regexpf(t, "401 Unauthorized: map\\[message:0x[0-9a-f]+\\]", err.Error(), "Error message should contain the address of the channel")
}

func Test_LoadFile_InvalidPath(t *testing.T) {

	t.Parallel()

	var m wst.M
	err := wst.LoadFile("invalid_path", &m)
	assert.Error(t, err)
}

func Test_LoadFile_InvalidJson(t *testing.T) {

	t.Parallel()

	var m wst.M
	// write the json:
	err := os.WriteFile("invalid.json", []byte("invalid json"), 0644)
	if err != nil {
		t.Errorf("Error writing file: %v", err)
		return
	}
	// clean up:
	defer func() {
		err := os.Remove("invalid.json")
		if err != nil {
			t.Errorf("Error deleting file: %v", err)
		}
	}()
	err = wst.LoadFile("invalid.json", &m)

	assert.Error(t, err)
}

func Test_DashedCase(t *testing.T) {

	t.Parallel()

	assert.Equal(t, "hello-world", wst.DashedCase("helloWorld"))
}

func Test_Transform(t *testing.T) {

	t.Parallel()

	type OutputType struct {
		Foo string `json:"foo"`
	}
	inputMap := wst.M{"foo": "bar"}
	var output OutputType
	err := wst.Transform(inputMap, &output)
	assert.NoError(t, err)
	assert.Equal(t, "bar", output.Foo)
}

func Test_Transform_WithOutputError(t *testing.T) {

	t.Parallel()

	type OutputType struct {
		Foo chan int `json:"foo"`
	}
	inputMap := wst.M{"foo": "bar"}
	var output OutputType
	err := wst.Transform(inputMap, &output)
	assert.Error(t, err)
}

func Test_Transform_WithInputError(t *testing.T) {

	t.Parallel()

	type OutputType struct {
		Foo string `json:"foo"`
	}
	inputMap := wst.M{"foo": make(chan int)}
	var output OutputType
	err := wst.Transform(inputMap, &output)
	assert.Error(t, err)
}

func Test_ParseDate1(t *testing.T) {

	t.Parallel()

	d, err := wst.ParseDate("2021-01-01T00:00:00-0500")
	assert.NoError(t, err)
	// convert to UTC:
	d = d.UTC()
	assert.Equal(t, "2021-01-01T05:00:00Z", d.Format("2006-01-02T15:04:05Z"))
}

func Test_ParseDate2(t *testing.T) {

	t.Parallel()

	d, err := wst.ParseDate("2021-01-01T00:00:00.000+0600")
	assert.NoError(t, err)
	// convert to UTC:
	d = d.UTC()
	assert.Equal(t, "2020-12-31T18:00:00Z", d.Format("2006-01-02T15:04:05Z"))
}

func Test_ParseDate3(t *testing.T) {

	t.Parallel()

	d, err := wst.ParseDate("2021-01-01T00:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, int64(1609459200), d.Unix())
}

func Test_ParseDate4(t *testing.T) {

	t.Parallel()

	d, err := wst.ParseDate("2021-01-01T00:00:00.000Z")
	assert.NoError(t, err)
	assert.Equal(t, "2021-01-01T00:00:00Z", d.Format("2006-01-02T15:04:05Z"))
}

func Test_ParseDateWithImpossibleDate(t *testing.T) {

	t.Parallel()

	_, err := wst.ParseDate("2021-02-29T00:00:00Z")
	assert.Error(t, err)
}

// test CreateError(fiberError *fiber.Error, code string, details fiber.Map, name string)
func Test_CreateError(t *testing.T) {

	t.Parallel()

	err := wst.CreateError(fiber.ErrBadRequest, "BAD_REQUEST", fiber.Map{"message": "You sent a bad request"}, "Error")
	assert.Equal(t, "400 Bad Request: {\"message\":\"You sent a bad request\"}", err.Error())
}

func Test_AFromGenericSlice_ValidEntries(t *testing.T) {

	t.Parallel()

	slice := &[]interface{}{wst.M{"a": 1}, wst.M{"a": 2}, wst.M{"a": 3}}
	var a = *wst.AFromGenericSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	assert.Equal(t, 3, a[2]["a"])
}

func Test_AFromGenericSlice_NilInput(t *testing.T) {

	t.Parallel()

	var a = wst.AFromGenericSlice(nil)
	assert.Nil(t, a)
}

func Test_AFromGenericSlice_WrongType(t *testing.T) {

	t.Parallel()

	slice := &[]interface{}{wst.M{"a": 1}, map[string]interface{}{"a": 3}}
	var a = *wst.AFromGenericSlice(slice)
	assert.Equal(t, 2, len(a))
	assert.Equal(t, 1, a[0]["a"])
	// Ensure the second 1 is an empty wst.M
	assert.Equal(t, 0, len(a[1]))
}

func Test_AFromPrimitiveSlice_ValidEntries(t *testing.T) {

	t.Parallel()

	var slice *primitive.A = &primitive.A{primitive.M{"a": 1}, primitive.M{"a": 2}, primitive.M{"a": 3}}
	var a = *wst.AFromPrimitiveSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	assert.Equal(t, 3, a[2]["a"])
}

func Test_AFromPrimitiveSlice_ValidEntriesAsM(t *testing.T) {

	t.Parallel()

	var slice *primitive.A = &primitive.A{wst.M{"a": 1}, wst.M{"a": 2}, wst.M{"a": 3}}
	var a = *wst.AFromPrimitiveSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	assert.Equal(t, 3, a[2]["a"])
}

func Test_AFromPrimitiveSlice_NilInput(t *testing.T) {

	t.Parallel()

	var a = wst.AFromPrimitiveSlice(nil)
	assert.Nil(t, a)
}

func Test_AFromPrimitiveSlice_WrongType(t *testing.T) {

	t.Parallel()

	var slice *primitive.A = &primitive.A{primitive.M{"a": 1}, wst.M{"a": 2}, map[string]interface{}{"a": 3}}
	var a = *wst.AFromPrimitiveSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	// Ensure the second 1 is an empty wst.M
	assert.Equal(t, 0, len(a[2]))
}
