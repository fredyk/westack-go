package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/client/v2/wstfuncs"
	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/lib/uploads"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

type TestInput struct {
	Message string `json:"message"`
}

type EmptyStruct struct{}

type unexportedType struct {
	Field string
}

type TestOutput struct {
	Response string `json:"response"`
	Metadata struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	} `json:"metadata"`
	Ints        []int                `json:"ints"`
	Floats      []float64            `json:"floats"`
	Booleans    []bool               `json:"booleans"`
	SomeInt     int                  `json:"someInt"`
	SomeFloat   float64              `json:"someFloat"`
	SomeBoolean bool                 `json:"someBoolean"`
	SomeMap     wst.M                `json:"someMap"`
	EmptySlice  []struct{}           `json:"emptySlice"`
	File        multipart.FileHeader `json:"file"`
	TimeField   time.Time            `json:"timeField"`

	UntaggedField   EmptyStruct
	unexportedField unexportedType
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
		EmptySlice: []struct{}{{}},
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
	assert.Equal(t, 1, len(output.EmptySlice))

	assert.Equal(t, "Item 1", output.Metadata.Items[0].Name)
	assert.Equal(t, 1, output.Ints[0])
	assert.Equal(t, 1.1, output.Floats[0])
	assert.Equal(t, true, output.Booleans[0])
	assert.Equal(t, -2, output.SomeInt)
	assert.Equal(t, 10.5, output.SomeFloat)
	assert.Equal(t, true, output.SomeBoolean)
	assert.Equal(t, "value1", output.SomeMap.GetString("key1"))

}

func Test_MinioUpload(t *testing.T) {

	t.Parallel()

	// read file from disk
	f, err := os.Open("fixtures/upload_file.txt")
	assert.NoError(t, err)
	assert.NotNil(t, f)
	defer f.Close()

	reqBody := uploads.MinioUpload{
		Name:      "changed_filename.txt",
		Directory: "examples",
	}
	req, err := createMultipartRequest("POST", fmt.Sprintf("%s/apps/upload", wstfuncs.GetBaseUrl()),
		reqBody.Name, reqBody.Directory, f, wst.M{
			"Authorization": "Bearer " + adminAccountToken.GetString("id"),
		})
	assert.NoError(t, err)
	assert.NotNil(t, req)

	resp, err := (&http.Client{Timeout: time.Duration(120) * time.Second}).Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode, readBodyOnError(resp))

	var output uploads.MinioUploadResponse
	// Notice the original '_' in the filename was automatically replaced with '-'
	expectedUrl := fmt.Sprintf("https://%s/wstuploadstest/examples/changed-filename.txt", os.Getenv("MINIO_DOMAIN"))
	err = readMultipartResponse(resp, &output)
	assert.NoError(t, err)
	assert.Equal(t, expectedUrl, output.Url)

	// Now read the file
	req, err = http.NewRequest("GET", output.Url, nil)
	assert.NoError(t, err)
	resp, err = (&http.Client{Timeout: time.Duration(120) * time.Second}).Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode, readBodyOnError(resp))

	expectedContents := "This file should be uploaded to the server."

	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(expectedContents)), n)
	assert.Equal(t, expectedContents, buf.String())

}

func readBodyOnError(resp *http.Response) string {
	if resp.StatusCode == 200 {
		return ""
	}
	body := resp.Body
	defer body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(body)
	return buf.String()
}

func readMultipartResponse(resp *http.Response, u *uploads.MinioUploadResponse) error {
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, u)
	return err
}

func createMultipartRequest(method string, url string, fileName string, directory string, f *os.File, headers wst.M) (*http.Request, error) {
	b, boundary, err := uploads.CreateRawMultipart(fileName, directory, f, "text/plain")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		return req, err
	}

	headers["Content-Type"] = fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
	for k, v := range headers {
		req.Header.Add(k, v.(string))
	}

	return req, err
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

func SubTestRateLimitedOperation5Seconds(t *testing.T) {
	// only 3 operations allowed every 5 seconds

	testRateLimitIterations(t, 4)

}

func SubTestRateLimitedOperation14Seconds(t *testing.T) {
	// only 9 operations every 10 seconds
	start := time.Now()
	runIterations(t, 4)
	time.Sleep(5200*time.Millisecond - time.Since(start))
	runIterations(t, 4)
	time.Sleep(10300*time.Millisecond - time.Since(start))
	runIterations(t, 2)
	//time.Sleep(11200*time.Millisecond - time.Since(start))
	// only 1 operation in the next 2 seconds
	testRateLimitIterations(t, 0)
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
	for i := 0; i < iterations; i++ {
		iterationStart := time.Now()
		output, err := invokeRateLimited()
		assert.NoError(t, err, "i=%d/%d", i, iterations)
		assert.Equal(t, "Hello, World", output.Response, "i=%d", i)

		time.Sleep(1001*time.Millisecond - time.Since(iterationStart))
		time.Sleep(time.Duration(1000*(i+1)+100)*time.Millisecond - time.Since(start))
	}
}

func Test_RateLimits(t *testing.T) {

	//t.Parallel()

	start := time.Now()
	t.Run("TestRateLimitedOperation1Second", SubTestRateLimitedOperation1Second)
	time.Sleep(5100*time.Millisecond - time.Since(start))
	t.Run("TestRateLimitedOperation5Seconds", SubTestRateLimitedOperation5Seconds)
	time.Sleep(13100*time.Millisecond - time.Since(start))
	t.Run("TestRateLimitedOperation14Seconds", SubTestRateLimitedOperation14Seconds)

}

func invokeRateLimited() (TestOutput, error) {
	return wstfuncs.InvokeApiTyped[TestOutput]("POST", "/notes/hooks/rate-limited-operation", wst.M{
		"message": "World",
	}, wst.M{
		"Authorization": "Bearer " + adminAccountToken.GetString("id"),
		"Content-Type":  "application/json",
	})
}

func Test_OptionsVerb(t *testing.T) {

	t.Parallel()

	resp, err := wstfuncs.InvokeApiFullResponse("OPTIONS", "/notes/hooks/remote-operation-example",
		nil, wst.M{
			"Authorization": "Bearer " + adminAccountToken.GetString("id"),
		})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
