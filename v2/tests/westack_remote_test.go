package tests

import (
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestInput struct {
	Message string `json:"message"`
}

type TestOutput struct {
	Response string `json:"response"`
	Metadata struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	} `json:"metadata"`
	Ints        []int     `json:"ints"`
	Floats      []float64 `json:"floats"`
	Booleans    []bool    `json:"booleans"`
	SomeInt     int       `json:"someInt"`
	SomeFloat   float64   `json:"someFloat"`
	SomeBoolean bool      `json:"someBoolean"`
	SomeMap     wst.M     `json:"someMap"`
}

func RemoteOperationExample(req TestInput) (TestOutput, error) {
	return TestOutput{
		Response: "Hello, " + req.Message,
		Metadata: struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		}{
			Items: []struct {
				Name string `json:"name"`
			}{
				{
					Name: "Item 1",
				},
			},
		},
		Ints:        []int{1, 2, 3},
		Floats:      []float64{1.1, 2.2, 3.3},
		Booleans:    []bool{true, false, true},
		SomeInt:     -2,
		SomeFloat:   10.5,
		SomeBoolean: true,
		SomeMap: wst.M{
			"key1": "value1",
		},
	}, nil
}

func Test_BindRemoteOperation(t *testing.T) {

	// Invoke the registered remote operation at init()
	output, err := wstfuncs.InvokeApiTyped[TestOutput]("POST", "/notes/hooks/remote-operation-example", wst.M{
		"message": "World",
	}, wst.M{
		"Authorization": "Bearer " + adminAccountToken.GetString("id"),
		"Content-Type":  "application/json",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World", output.Response)
	assert.Greater(t, len(output.Metadata.Items), 0)
	assert.Equal(t, 3, len(output.Ints))
	assert.Equal(t, 3, len(output.Floats))
	assert.Equal(t, 3, len(output.Booleans))
	assert.Greater(t, len(output.SomeMap), 0)

	assert.Equal(t, "Item 1", output.Metadata.Items[0].Name)
	assert.Equal(t, 1, output.Ints[0])
	assert.Equal(t, 1.1, output.Floats[0])
	assert.Equal(t, true, output.Booleans[0])
	assert.Equal(t, -2, output.SomeInt)
	assert.Equal(t, 10.5, output.SomeFloat)
	assert.Equal(t, true, output.SomeBoolean)
	assert.Equal(t, "value1", output.SomeMap.GetString("key1"))

}
