package tests

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	wst "github.com/fredyk/westack-go/westack/common"
)

func createUserThroughNetwork(t *testing.T) wst.M {
	randUserN := 100000000 + rand.Intn(899999999)
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/users", jsonToReader(wst.M{
		"username": fmt.Sprintf("user%v", randUserN),
		"email":    fmt.Sprintf("user.%v@example.com", randUserN),
		"password": "abcd1234.",
	}))
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	var out []byte
	var parsed wst.M

	// read response body bytes
	_, err = response.Body.Read(out)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		t.Errorf("Error: %v", err)
		fmt.Printf("Received bytes <-- %v\n", out)
	}

	return parsed
}

func jsonToReader(m wst.M) io.Reader {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(out)
}

func createNoteForUser(userId string, token string, t *testing.T) (note wst.M, err error) {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/notes", jsonToReader(wst.M{
		"title":   "Test Note",
		"content": "This is a test note",
		"userId":  userId,
	}))
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	var out []byte
	var parsed wst.M

	// read response body bytes
	_, err = response.Body.Read(out)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	return parsed, err

}

func Test_FindMany(t *testing.T) {
	var err error

	//user := createUserThroughNetwork(t)
	//token := loginUser(user["email"].(string), "abcd1234.", t)
	//_, err = createNoteForUser(user["id"].(string), token["id"].(string), t)
	//if err != nil {
	//	t.Errorf("Error: %v", err)
	//}

	request, err := http.NewRequest("GET", "http://localhost:8019/api/v1/notes", nil)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	assert.Equal(t, 200, response.StatusCode)

	var out []byte
	var parsed wst.A
	// read response body bytes
	//_, err = response.Body.Read(out)
	//if err != nil {
	//	t.Errorf("Error: %v", err)
	//}
	for {
		var buf [4096]byte
		n, err := response.Body.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				break
			} else if err == io.ErrUnexpectedEOF {
				t.Errorf("Error: %v", err)
				out = append(out, buf[:n]...)
				break
			}
			t.Errorf("Error: %v", err)
		}
		out = append(out, buf[:n]...)
	}

	assert.Greaterf(t, len(out), 0, "Received bytes <-- %v, %v\n", len(out), string(out))

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		t.Errorf("Error: %v", err)
		fmt.Printf("Received bytes <-- %v, %v\n", len(out), string(out))
		time.Sleep(5 * time.Minute)
	}

	//fmt.Printf("parsed: %v", parsed[:1])

	assert.Greaterf(t, len(parsed), 0, "parsed: %v\n", parsed)

}

func loginUser(email string, password string, t *testing.T) wst.M {
	request, err := http.NewRequest("POST", "http://localhost:8019/api/v1/users/login", jsonToReader(wst.M{
		"email":    email,
		"password": password,
	}))
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	var out []byte
	var parsed wst.M

	// read response body bytes
	_, err = response.Body.Read(out)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// parse response body bytes
	err = json.Unmarshal(out, &parsed)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	return parsed
}
