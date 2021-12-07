package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/fredyk/westack-go/westack"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"math/big"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

var app *westack.WeStack

func init() {
	app = westack.New(westack.WeStackOptions{
		Debug:       true,
		RestApiRoot: "/api/v1",
		Port:        8021,
	})
	app.Boot(func(app *westack.WeStack) {

	})
	go app.Start(fmt.Sprintf("localhost:%v", app.Port))
	time.Sleep(300 * time.Millisecond)
}

func createUser(t *testing.T, b *bytes.Buffer) {

	response, err := app.Server.Test(httptest.NewRequest("POST", "/api/v1/users", b))
	if err != nil {
		t.Error(err)
		return
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return
	}
}

func login(t *testing.T, b *bytes.Buffer) (string, string) {
	response, err := app.Server.Test(httptest.NewRequest("POST", "/api/v1/users/login", b))
	if err != nil {
		t.Error(err)
		return "", ""
	}

	contentLength, err := strconv.Atoi(response.Header["Content-Length"][0])
	responseBytes := make([]byte, contentLength)
	count, err := response.Body.Read(responseBytes)
	if err != nil && err != io.EOF {
		t.Error(err)
		return "", ""
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return "", ""
	}

	if !assert.Greater(t, count, 0) {
		return "", ""
	}

	var loginResponse map[string]interface{}
	err = json.Unmarshal(responseBytes, &loginResponse)
	if err != nil {
		t.Error(err)
		return "", ""
	}

	if assert.NotEmpty(t, loginResponse["id"]) && assert.NotEmpty(t, loginResponse["userId"]) {
		return loginResponse["id"].(string), loginResponse["userId"].(string)
	} else {
		t.Error("Wrong response")
		return "", ""
	}
}

func Test_WeStackCreateUser(t *testing.T) {

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"
	body := map[string]interface{}{"email": email, "password": password}
	bodyBytes := createBody(t, body)
	createUser(t, bodyBytes)

}

func createBody(t *testing.T, body map[string]interface{}) *bytes.Buffer {
	bodyBytes := new(bytes.Buffer)
	if err := json.NewEncoder(bodyBytes).Encode(body); err != nil {
		t.Error(err)
		return nil
	}
	return bodyBytes
}

func Test_WeStackLogin(t *testing.T) {

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"

	log.Println("Email", email)
	body := map[string]interface{}{"email": email, "password": password}
	bodyBytes := createBody(t, body)
	createUser(t, bodyBytes)

	bodyBytes = createBody(t, body)
	login(t, bodyBytes)

}

func Test_WeStackDelete(t *testing.T) {

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"
	body := map[string]interface{}{"email": email, "password": password}
	bodyBytes := createBody(t, body)
	createUser(t, bodyBytes)

	bodyBytes = createBody(t, body)
	_, userId := login(t, bodyBytes)

	response, err := app.Server.Test(httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%v", userId), nil))
	if err != nil {
		t.Error(err)
		return
	}

	if !assert.Equal(t, 204, response.StatusCode) {
		return
	}

}
