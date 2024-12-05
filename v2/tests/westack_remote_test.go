package tests

import (
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

func RateLimitedOperation(req TestInput) (TestOutput, error) {
	return TestOutput{
		Response: "Hello, " + req.Message,
	}, nil
}

func Test_BindRemoteOperation(t *testing.T) {

	t.Parallel()

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

func SubTestRateLimitedOperation1Second(t *testing.T) {
	// First call should succeed
	output, err := invokeRateLimited()
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World", output.Response)
	// Second call should fail
	output, err = invokeRateLimited()
	assert.Error(t, err)
	// Server returns Rate limit exceeded
	assert.Equal(t, "invalid character 'R' looking for beginning of value", err.Error())
}

func SubTestRateLimitedOperation3Seconds(t *testing.T) {
	// only 2 operations allowed every 3 seconds

	testRateLimitIterations(t, 2)

}

func SubTestRateLimitedOperation5Seconds(t *testing.T) {
	// only 2 operations in the first 3 seconds
	start := time.Now()
	runIterations(t, 2)
	time.Sleep(3100*time.Millisecond - time.Since(start))
	// only 1 operation in the next 2 seconds
	testRateLimitIterations(t, 1)
}

func testRateLimitIterations(t *testing.T, iterations int) {
	runIterations(t, iterations)

	// Third call should fail
	output, err := invokeRateLimited()
	assert.Error(t, err)
	assert.Empty(t, output)
	// Server returns Rate limit exceeded
	assert.Equal(t, "invalid character 'R' looking for beginning of value", err.Error())
}

func runIterations(t *testing.T, iterations int) {
	start := time.Now()
	output, err := invokeRateLimited()
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World", output.Response)

	for i := 1; i < iterations; i++ {
		time.Sleep(time.Duration(1000*i+100)*time.Millisecond - time.Since(start))
		output, err := invokeRateLimited()
		assert.NoError(t, err, "i=%d", i)
		assert.Equal(t, "Hello, World", output.Response, "i=%d", i)
	}
}

func Test_RateLimits(t *testing.T) {

	t.Parallel()
	t.Run("RateLimitedOperation1Second", SubTestRateLimitedOperation1Second)
	time.Sleep(1100 * time.Millisecond)
	t.Run("RateLimitedOperation3Seconds", SubTestRateLimitedOperation3Seconds)
	time.Sleep(3100 * time.Millisecond)
	t.Run("RateLimitedOperation5Seconds", SubTestRateLimitedOperation5Seconds)

}

func invokeRateLimited() (TestOutput, error) {
	return wstfuncs.InvokeApiTyped[TestOutput]("POST", "/notes/hooks/rate-limited-operation", wst.M{
		"message": "World",
	}, wst.M{
		"Authorization": "Bearer " + adminAccountToken.GetString("id"),
		"Content-Type":  "application/json",
	})
}
