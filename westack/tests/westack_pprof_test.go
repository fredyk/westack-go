package tests

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetHeap(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8019/debug/pprof/heap", nil)
	assert.Nil(t, err)

	basicAuthUsername := "test"
	basicAuthPassword := "abcd1234."
	request.SetBasicAuth(basicAuthUsername, basicAuthPassword)

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Greater(t, response.ContentLength, int64(0))
}

func Test_GetHeapUnauthorized1(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8019/debug/pprof/heap", nil)
	assert.Nil(t, err)

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func Test_GetHeapUnauthorized2(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8019/debug/pprof/heap", nil)
	assert.Nil(t, err)

	request.Header.Set("Authorization", "<invalid>")

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func Test_GetHeapUnauthorized3(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8019/debug/pprof/heap", nil)
	assert.Nil(t, err)

	request.Header.Set("Authorization", "Basic <invalid>")

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)

}

func Test_GetHeapUnauthorized4(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8019/debug/pprof/heap", nil)
	assert.Nil(t, err)

	toEncode := "test<skippingcolon>abcd1234."
	encoded := base64.StdEncoding.EncodeToString([]byte(toEncode))
	request.Header.Set("Authorization", "Basic "+encoded)

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func Test_GetHeapUnauthorized5(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8019/debug/pprof/heap", nil)
	assert.Nil(t, err)

	request.SetBasicAuth("test", "<invalid>")

	response, err := http.DefaultClient.Do(request)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}
