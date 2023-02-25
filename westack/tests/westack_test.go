package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack"
	wst "github.com/fredyk/westack-go/westack/common"
)

var app *westack.WeStack

func init() {
	app = westack.New(westack.Options{})
	app.Boot(func(app *westack.WeStack) {

	})
	go func() {
		err := app.Start()
		if err != nil {
			fmt.Printf("Error while starting server: %v", err)
			os.Exit(1)
		}
	}()
	time.Sleep(300 * time.Millisecond)
}

func createUser(t *testing.T, b *bytes.Buffer) {

	request := httptest.NewRequest("POST", "/api/v1/users", b)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Server.Test(request)
	if err != nil {
		t.Error(err)
		return
	}
	if !assert.Equal(t, 200, response.StatusCode) {
		return
	}
}

func login(t *testing.T, b *bytes.Buffer) (string, string) {
	request := httptest.NewRequest("POST", "/api/v1/users/login", b)
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Server.Test(request)
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

	var loginResponse wst.M
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
	body := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	bodyBytes := createBody(t, body)
	createUser(t, bodyBytes)

}

func createBody(t *testing.T, body wst.M) *bytes.Buffer {
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
	body := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	bodyBytes := createBody(t, body)
	createUser(t, bodyBytes)

	bodyBytes = createBody(t, body)
	login(t, bodyBytes)

}

func Test_WeStackDelete(t *testing.T) {

	n, _ := rand.Int(rand.Reader, big.NewInt(899999999))
	email := fmt.Sprintf("email%v@example.com", 100000000+n.Int64())
	password := "test"
	body := wst.M{"email": email, "password": password, "username": fmt.Sprintf("user%v", n)}
	bodyBytes := createBody(t, body)
	createUser(t, bodyBytes)

	bodyBytes = createBody(t, body)
	bearer, userId := login(t, bodyBytes)

	request := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%v", userId), nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", bearer))
	response, err := app.Server.Test(request)
	if err != nil {
		t.Error(err)
		return
	}

	if !assert.Equal(t, 204, response.StatusCode) {
		return
	}

}

//// after all tests, stop the server
//func TestMain(m *testing.M) {
//	code := m.Run()
//	err := app.Stop()
//	if err != nil {
//		log.Fatal(err)
//	}
//	os.Exit(code)
//}
