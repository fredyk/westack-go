package tests

import (
	"bytes"
	"encoding/json"
	"github.com/fredyk/westack-go/westack"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
)

var app *westack.WeStack

func init() {
	app = westack.New(westack.WeStackOptions{
		Debug:       true,
		RestApiRoot: "/api/v1",
		Port:        8023,
	})
	app.Boot(func(app *westack.WeStack) {

	})
	go app.Start("localhost:8021")
}

func Test_WeStackCreateUser(t *testing.T) {

	m, b := map[string]interface{}{"email": "email1@example.com", "password": "test"}, new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(m)
	if err != nil {
		t.Error(err)
		return
	}
	response, err := app.Server.Test(httptest.NewRequest("POST", "/api/v1/users", b))
	if err != nil {
		t.Error(err)
		return
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return
	}

}

func Test_WeStackLogin(t *testing.T) {

	m, b := map[string]interface{}{"email": "email1@example.com", "password": "test"}, new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(m)
	if err != nil {
		t.Error(err)
		return
	}
	response, err := app.Server.Test(httptest.NewRequest("POST", "/api/v1/users/login", b))
	if err != nil {
		t.Error(err)
		return
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return
	}

	contentLength, err := strconv.Atoi(response.Header["Content-Length"][0])
	responseBytes := make([]byte, contentLength)
	count, err := response.Body.Read(responseBytes)
	if err != nil && err != io.EOF {
		t.Error(err)
		return
	}
	if !assert.Greater(t, count, 0) {
		return
	}

	var loginResponse map[string]interface{}
	err = json.Unmarshal(responseBytes, &loginResponse)
	if err != nil {
		t.Error(err)
		return
	}

	if !assert.NotEmpty(t, loginResponse["id"]) {
		return
	}

}
